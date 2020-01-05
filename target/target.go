package target

import (
	"fmt"
	"io"
	"strings"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/gocloud/provider"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/terraform"

	"github.com/mevansam/goforms/config"
)

// Input types
type TargetState int

const (
	Undeployed TargetState = iota
	Running
	Shutdown
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
	if t.Output != nil {
		return Running
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
