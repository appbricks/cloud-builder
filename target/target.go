package target

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goutils/logger"

	"github.com/mevansam/goforms/config"
)

// Input types
type TargetState int

const (
	Undeployed TargetState = iota
	Running
	Shutdown
	Pending
	Unknown
)

// a target is a recipe configured to be
// launched in a public cloud region
type Target struct {
	RecipeName string `json:"recipeName"`
	RecipeIaas string `json:"recipeIaas"`

	Recipe   cookbook.Recipe        `json:"recipe,omitempty"`
	Provider provider.CloudProvider `json:"provider,omitempty"`
	Backend  backend.CloudBackend   `json:"backend,omitempty"`

	Output *map[string]terraform.Output `json:"output,omitempty"`

	managedInstances []*managedInstance
	compute          cloud.Compute
}

type managedInstance struct {
	Name string

	Instance cloud.ComputeInstance
	Metadata map[string]interface{}

	order int
}

func NewTarget(
	r, p, b config.Configurable,
) *Target {

	return &Target{
		RecipeName: strings.Split(r.Name(), "/")[0],
		RecipeIaas: p.Name(),

		Recipe:   r.(cookbook.Recipe),
		Provider: p.(provider.CloudProvider),
		Backend:  b.(backend.CloudBackend),
	}
}

// load target cloud references
func (t *Target) LoadRemoteRefs() error {

	var (
		err error
		ok  bool

		managedInstancesOutput terraform.Output
		managedInstanceValues  []interface{}
		instanceMetaData       map[string]interface{}

		instance       *managedInstance
		cloudInstances []cloud.ComputeInstance

		value interface{}
		name  string
		order float64

		instanceRef map[string]*managedInstance
	)

	if t.compute == nil {
		if err = t.Provider.Connect(); err != nil {
			return err
		}
		if t.compute, err = t.Provider.GetCompute(); err != nil {
			return err
		}
	}
	if t.Output != nil {
		if managedInstancesOutput, ok = (*t.Output)["cb_managed_instances"]; ok {

			if managedInstanceValues, ok = managedInstancesOutput.Value.([]interface{}); !ok {
				return fmt.Errorf("managed instance output is not a list")
			}

			numInstance := len(managedInstanceValues)
			instanceRef = make(map[string]*managedInstance)
			t.managedInstances = make([]*managedInstance, 0, numInstance)
			ids := make([]string, numInstance)

			for i, managedInstanceValue := range managedInstanceValues {
				if instanceMetaData, ok = managedInstanceValue.(map[string]interface{}); !ok {
					return fmt.Errorf("managed instance metadata value is not a map of key value pairs")
				}

				if value, ok = instanceMetaData["id"]; !ok {
					return fmt.Errorf("managed instance metadata did no contain an id key")
				}
				if ids[i], ok = value.(string); !ok {
					return fmt.Errorf("managed instance metadata id key value is not a string")
				}

				if value, ok = instanceMetaData["name"]; !ok {
					return fmt.Errorf("managed instance metadata did no contain a name key")
				}
				if name, ok = value.(string); !ok {
					return fmt.Errorf("managed instance metadata name key value is not a string")
				}
				instance = &managedInstance{
					Name:     name,
					Metadata: instanceMetaData,
					order:    math.MaxInt64,
				}
				if value, ok = instanceMetaData["order"]; ok {
					if order, ok = value.(float64); !ok {
						return fmt.Errorf("managed instance metadata name key value is not a string")
					}
					instance.order = int(order)
				}
				instanceRef[ids[i]] = instance

				// insert instance into managed instance list in order
				j := sort.Search(i, func(j int) bool {
					managedInstance := t.managedInstances[j]
					return managedInstance.order > instance.order ||
						(managedInstance.order == instance.order &&
							strings.Compare(managedInstance.Name, instance.Name) == 1)
				})
				t.managedInstances = append(t.managedInstances, instance)
				if len(t.managedInstances) > 1 {
					copy(t.managedInstances[i+1:], t.managedInstances[i:])
					t.managedInstances[j] = instance
				}
			}

			if cloudInstances, err = t.compute.GetInstances(ids); err != nil {
				return err
			}
			for _, cloudInstance := range cloudInstances {
				instanceRef[cloudInstance.ID()].Instance = cloudInstance
			}

		} else {
			logger.DebugMessage(
				"Target '%s' does not appear have any managed instances.",
				t.Key(),
			)
		}
	}

	return nil
}

// returns a unique identifier for
// the target
func (t *Target) Key() string {

	var (
		key strings.Builder
	)
	key.WriteString(t.RecipeName)
	key.Write([]byte{'/'})
	key.WriteString(t.RecipeIaas)
	key.Write([]byte{'/'})
	key.WriteString(strings.Join(t.Recipe.GetKeyFieldValues(), "/"))
	return key.String()
}

func (t *Target) Description() string {
	return fmt.Sprintf(
		"Deployment \"%s\" on Cloud \"%s\" and Region \"%s\"",
		t.DeploymentName(),
		t.Provider.Name(),
		*t.Provider.Region(),
	)
}

func (t *Target) DeploymentName() string {

	if variable, exists := t.Recipe.GetVariable("name"); exists && variable.Value != nil {
		return *variable.Value
	} else {
		return "NONAME"
	}
}

func (t *Target) Status() TargetState {

	var (
		err error

		state cloud.InstanceState
	)

	if t.Output != nil {

		numInstances := len(t.managedInstances)
		numRunning := 0
		numStopped := 0
		numPending := 0
		numUnknown := 0
		for _, instance := range t.managedInstances {
			if state, err = instance.Instance.State(); err != nil {
				logger.TraceMessage(
					"Managed instance '%s' of target '%s' returned an error when querying its state: %s",
					instance.Instance.Name(), t.Key(), err.Error(),
				)
				numUnknown++
				continue
			}
			switch state {
			case cloud.StateRunning:
				numRunning++
			case cloud.StateStopped:
				numStopped++
			case cloud.StatePending:
				numPending++
			default:
				numUnknown++
			}
		}

		if numUnknown == numInstances {
			return Unknown
		}
		if numRunning == numInstances {
			return Running
		}
		if numStopped == numInstances {
			return Shutdown
		}
		if (numRunning + numStopped + numPending) == numInstances {
			return Pending
		}

		logger.DebugMessage(
			"Unable to determine state of target '%s'. Have %d instances with state %d running, %d stopped, %d pending and %d unknown.",
			t.Key(), numInstances, numRunning, numStopped, numPending, numUnknown,
		)
		return Unknown

	} else {
		return Undeployed
	}
}

// returns a copy of this target
func (t *Target) Copy() (*Target, error) {

	var (
		err error

		recipeCopy,
		providerCopy,
		backendCopy config.Configurable
	)

	if recipeCopy, err = t.Recipe.Copy(); err != nil {
		return nil, err
	}
	if providerCopy, err = t.Provider.Copy(); err != nil {
		return nil, err
	}
	if backendCopy, err = t.Backend.Copy(); err != nil {
		return nil, err
	}
	return &Target{
		RecipeName: t.RecipeName,
		RecipeIaas: t.RecipeIaas,

		Recipe:   recipeCopy.(cookbook.Recipe),
		Provider: providerCopy.(provider.CloudProvider),
		Backend:  backendCopy.(backend.CloudBackend),
	}, nil
}

// prepares the target backend
func (t *Target) PrepareBackend() error {

	var (
		err error

		storage cloud.Storage
	)

	if !t.Backend.IsValid() {
		return fmt.Errorf(
			"the backend configuration for target %s is not valid",
			t.Key(),
		)
	}
	if err = t.Provider.Connect(); err != nil {
		return err
	}
	if storage, err = t.Provider.GetStorage(); err != nil {
		return err
	}
	_, err = storage.NewInstance(t.Backend.GetStorageInstanceName())
	return err
}

// returns a launcher for this target
func (t *Target) NewBuilder(outputBuffer, errorBuffer io.Writer) (*Builder, error) {

	return NewBuilder(
		strings.Join(t.Recipe.GetKeyFieldValues(), "/"),
		t.Recipe,
		t.Provider,
		t.Backend,
		outputBuffer,
		errorBuffer)
}
