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

// Instance state callback
type InstanceStateChange func(name string, instance cloud.ComputeInstance)

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

	CookbookTimestamp string `json:"cookbook_timestamp,omitempty"`

	description string
	version     string

	managedInstances []*managedInstance
	compute          cloud.Compute
}

type managedInstance struct {
	Instance cloud.ComputeInstance
	Metadata map[string]interface{}

	order int

	// recipe state output values
	// these can be IaaS specific
	id,
	name,
	description,
	fqdn,
	publicIP,
	sshPort,
	sshUser,
	sshKey,
	rootPasswd string
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

		output terraform.Output

		managedInstanceValues []interface{}
		instanceMetaData      map[string]interface{}

		instance       *managedInstance
		cloudInstances []cloud.ComputeInstance

		value    interface{}
		order    float64
		keyValue string

		instanceRef map[string]*managedInstance
	)

	readKeyValue := func(key string) (string, error) {
		if value, ok = instanceMetaData[key]; !ok {
			return "",
				fmt.Errorf("managed instance metadata did no contain a '%s' key", key)
		}
		if keyValue, ok = value.(string); !ok {
			return "",
				fmt.Errorf("managed instance metadata '%s' key value is not a string", key)
		}
		return keyValue, nil
	}

	if t.compute == nil {
		if err = t.Provider.Connect(); err != nil {
			return err
		}
		if t.compute, err = t.Provider.GetCompute(); err != nil {
			return err
		}
	}
	if t.Output != nil {
		if output, ok = (*t.Output)["cb_node_description"]; ok {
			if t.description, ok = output.Value.(string); !ok {
				return fmt.Errorf("node description key value is not a string")
			}
		}
		if output, ok = (*t.Output)["cb_bastion_version"]; ok {
			if t.version, ok = output.Value.(string); !ok {
				return fmt.Errorf("node version key value is not a string")
			}
		}
		if output, ok = (*t.Output)["cb_managed_instances"]; ok {

			if managedInstanceValues, ok = output.Value.([]interface{}); !ok {
				return fmt.Errorf("managed instance output is not a list")
			}

			numInstance := len(managedInstanceValues)
			t.managedInstances = make([]*managedInstance, 0, numInstance)

			ids := make([]string, numInstance)
			instanceRef = make(map[string]*managedInstance)

			for i, managedInstanceValue := range managedInstanceValues {
				if instanceMetaData, ok = managedInstanceValue.(map[string]interface{}); !ok {
					return fmt.Errorf("managed instance metadata value is not a map of key value pairs")
				}

				instance = &managedInstance{
					Metadata: instanceMetaData,
					order:    math.MaxInt64,
				}
				if value, ok = instanceMetaData["order"]; ok {
					if order, ok = value.(float64); !ok {
						return fmt.Errorf("managed instance metadata name key value is not a string")
					}
					instance.order = int(order)
				}
				if instance.id, err = readKeyValue("id"); err != nil {
					return err
				}
				if instance.name, err = readKeyValue("name"); err != nil {
					return err
				}
				if instance.description, err = readKeyValue("description"); err != nil {
					return err
				}
				if instance.publicIP, err = readKeyValue("public_ip"); err != nil {
					return err
				}
				if instance.sshPort, err = readKeyValue("ssh_port"); err != nil {
					return err
				}
				if instance.sshUser, err = readKeyValue("ssh_user"); err != nil {
					return err
				}
				if instance.sshKey, err = readKeyValue("ssh_key"); err != nil {
					return err
				}
				if instance.rootPasswd, err = readKeyValue("root_passwd"); err != nil {
					return err
				}

				ids[i] = instance.id
				instanceRef[ids[i]] = instance

				// insert instance into managed instance list in order
				j := sort.Search(i, func(j int) bool {
					managedInstance := t.managedInstances[j]
					return managedInstance.order > instance.order ||
						(managedInstance.order == instance.order &&
							strings.Compare(managedInstance.name, instance.name) == 1)
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
	return t.description
}

func (t *Target) Version() string {
	return t.version
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

func (t *Target) ManagedInstances() []*managedInstance {
	return t.managedInstances
}

func (t *Target) ManagedInstance(name string) *managedInstance {

	for _, managedInstance := range t.managedInstances {
		if managedInstance.name == name {
			return managedInstance
		}
	}
	return nil
}

func (t *Target) Resume(cb InstanceStateChange) error {

	var (
		err error
	)

	if t.Status() == Shutdown {
		for _, managedInstance := range t.managedInstances {
			cb(managedInstance.name, managedInstance.Instance)
			if err = managedInstance.Instance.Start(); err != nil {
				return err
			}
			cb(managedInstance.name, managedInstance.Instance)
		}
	} else {
		return fmt.Errorf("target is not in a 'shutdown' state")
	}

	return nil
}

func (t *Target) Suspend(cb InstanceStateChange) error {

	var (
		err error
	)

	if t.Status() == Running {
		for _, managedInstance := range t.managedInstances {
			cb(managedInstance.name, managedInstance.Instance)
			if err = managedInstance.Instance.Stop(); err != nil {
				return err
			}
			cb(managedInstance.name, managedInstance.Instance)
		}
	} else {
		return fmt.Errorf("target is not in a 'shutdown' state")
	}

	return nil
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

		Output: t.Output,

		CookbookTimestamp: t.CookbookTimestamp,
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

// managedInstance functions

func (i *managedInstance) Name() string {
	return i.name
}

func (i *managedInstance) Description() string {
	return i.description
}

func (i *managedInstance) PublicIP() string {
	return i.Instance.PublicIP()
}

func (i *managedInstance) FQDN() string {
	return i.fqdn
}

func (i *managedInstance) SSHAddress() string {
	return fmt.Sprintf("%s:%s", i.publicIP, i.sshPort)
}

func (i *managedInstance) SSHUser() string {
	return i.sshUser
}

func (i *managedInstance) SSHKey() []byte {
	return []byte(i.sshKey)
}

func (i *managedInstance) RootPassword() string {
	return i.rootPasswd
}
