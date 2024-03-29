package target

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/run"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/terraform"
)

type Builder struct {
	recipe   cookbook.Recipe
	provider provider.CloudProvider
	backend  backend.CloudBackend

	additonalInputs map[string]string

	cli          run.CLI
	configInputs map[string]terraform.Input

	output map[string]terraform.Output
}

// in: cookbookRecipe - the recipe to create a launcher for
// in: cloudProvider - the cloud provider for the cloud to launch the recipe in
// in: cloudBackend - the backend where launch state will be saved
// in: outputBuffer - buffer to redirect the stdout from CLI execution
// in: errorBuffer - buffer to redirect the stderr from CLI execution
func NewBuilder(
	pathKey string,
	cookbookRecipe,
	cloudProvider,
	cloudBackend config.Configurable,
	additonalInputs map[string]string,
	outputBuffer,
	errorBuffer io.Writer,
) (*Builder, error) {

	var (
		err error

		cli run.CLI
	)

	recipe := cookbookRecipe.(cookbook.Recipe)
	if cli, err = recipe.CreateCLI(
		pathKey,
		outputBuffer,
		errorBuffer,
	); err != nil {
		return nil, err
	}

	builder := &Builder{
		recipe: recipe,
		
		additonalInputs: additonalInputs,

		cli:          cli,
		configInputs: make(map[string]terraform.Input),
	}
	if cloudProvider != nil {
		builder.provider = cloudProvider.(provider.CloudProvider)
	}
	if cloudBackend != nil {
		builder.backend = cloudBackend.(backend.CloudBackend)
	}

	for _, variable := range recipe.GetVariables() {
		builder.configInputs[variable.Name] = terraform.Input{
			Optional: variable.Optional,
		}
	}

	return builder, nil
}

func (b *Builder) newRunner() (*terraform.Runner, error) {

	runner := terraform.NewRunner(
		b.cli,
		b.recipe.ConfigPath(),
		b.recipe.PluginPath(),
		b.configInputs,
	)
	err := b.setEnvVars(runner)
	return runner, err
}

func (b *Builder) setEnvVars(runner *terraform.Runner) error {
	
	var (
		err error
	)
	vars := make(map[string]string)

	// set environment variables
	if b.provider != nil {
		if err = b.provider.GetVars(vars); err != nil {
			return err
		}
	}
	// add additional inputs as TF_VAR_* environment variables
	for n, v := range b.additonalInputs {
		vars["TF_VAR_"+n] = v
	}
	// set the terraform data directory to the cli's working 
	// directory. otherwise it defaults to the directory where 
	// the root template is
	vars["TF_DATA_DIR"] = filepath.Join(b.cli.WorkingDirectory(), ".terraform")

	runner.SetEnv(vars)
	return nil
}

// prepare and return template variables
func (b *Builder) getTemplateVars(asTfEnv bool) (map[string]string, error) {

	var (
		err error

		inputForm  forms.InputForm
		inputField *forms.InputField

		value *string
		vars  map[string]string
	)

	// set terraform configuration inputs
	if inputForm, err = b.recipe.InputForm(); err != nil {
		return nil, err
	}
	vars = make(map[string]string)
	for _, inputField = range inputForm.InputFields() {
		if inputField.InputSet() {
			if value = inputField.Value(); value == nil {
				return nil, fmt.Errorf(
					"recipe '%s' input field '%s' was nil",
					b.recipe.Name(),
					inputField.Name())
			}
			
		} else {
			if _, exists := b.additonalInputs[inputField.Name()]; exists {
				// var has already been added as an additional input
				continue

			} else if value = inputField.Value(); value == nil {
				return nil, fmt.Errorf(
					"recipe '%s' input field '%s' was not set and does not have a default value",
					b.recipe.Name(),
					inputField.Name())
			}	
		}
		if asTfEnv {
			vars["TF_VAR_"+inputField.Name()] = *value
		} else {
			vars[inputField.Name()] = *value
		}
	}

	return vars, nil
}

// this build's local state
// - is build's run state present
// - is build's resource state local
func (b *Builder) GetLocalBuildState() (bool, bool) {

	var (
		err error

		tfStateData []byte
	)

	stateData := struct {
		Backend struct {
			Config struct {
				Path string
			}
		}
	}{}

	// check if terraform state file exists
	statePath := filepath.Join(
		b.cli.WorkingDirectory(),
		".terraform", "terraform.tfstate",
	)
	if tfStateData, err = os.ReadFile(statePath); err != nil {
		return false, false
	}
	if err = json.Unmarshal(tfStateData, &stateData); err != nil {
		return true, false
	}
	return true, len(stateData.Backend.Config.Path) > 0
}

// initialize the target
func (b *Builder) Initialize() error {

	var (
		err error

		runner *terraform.Runner
		vars   map[string]string
	)
	vars = make(map[string]string)

	if runner, err = b.newRunner(); err != nil {
		return err
	}

	// set backend arguments
	if b.backend != nil {
		if err = b.backend.GetVars(vars); err != nil {
			return err
		}
	}
	runner.SetBackend(vars)

	// initialize terraform configuration
	return runner.Init()
}

// initialize if not initialized
func (b *Builder) AutoInitialize() error {

	var (
		err error
	)

	// check if terraform state has been initialized
	statePath := filepath.Join(
		b.cli.WorkingDirectory(),
		".terraform", "terraform.tfstate",
	)
	if _, err = os.Stat(statePath); os.IsNotExist(err) {
		err = b.Initialize()
	}
	return err
}

// show launch plan
func (b *Builder) ShowLaunchPlan() error {

	var (
		err error

		runner *terraform.Runner
		vars   map[string]string
	)

	if runner, err = b.newRunner(); err == nil {
		if vars, err = b.getTemplateVars(false); err == nil {
			err = runner.Plan(vars)
		}
	}
	return err
}

// taints deployed instance resources so they
// are rebuilt next time launch is called
func (b *Builder) SetRebuildInstances() error {

	var (
		err error

		runner *terraform.Runner
	)

	if runner, err = b.newRunner(); err == nil {
		err = runner.Taint(b.recipe.ResourceInstanceList())
	}
	return err
}

// taints deployed instance's attached data
// resources so they are rebuilt next time
// launch is called
func (b *Builder) SetRebuildInstanceData() error {

	var (
		err error

		runner *terraform.Runner
	)

	if runner, err = b.newRunner(); err == nil {
		err = runner.Taint(b.recipe.ResourceInstanceDataList())
	}
	return err
}

// launch the target
func (b *Builder) Launch() error {

	var (
		err error

		runner *terraform.Runner
		vars   map[string]string
	)

	if runner, err = b.newRunner(); err == nil {
		if vars, err = b.getTemplateVars(false); err == nil {
			b.output, err = runner.Apply(vars)
		}
	}
	return err
}

// outputs from last launch
func (b *Builder) Output() *map[string]terraform.Output {
	return &b.output
}

// suspend the target
func (b *Builder) Suspend() error {

	return nil
}

// resume a suspended target
func (b *Builder) Resume() error {

	return nil
}

// migrate a target
func (b *Builder) Migrate() error {

	return nil
}

// delete all resources created for the target
func (b *Builder) Delete() error {

	var (
		err error

		vars map[string]string

		runner *terraform.Runner
	)

	if runner, err = b.newRunner(); err == nil {
		if vars, err = b.getTemplateVars(true); err == nil {
			runner.AddToEnv(vars)
			if err = runner.Destroy(); err == nil {
				
				// remove state file 
				// of deleted deployment
				os.RemoveAll(
					filepath.Join(
						b.cli.WorkingDirectory(),
						".terraform", "terraform.tfstate",
					),
				)
			}
		}
	}
	return err
}
