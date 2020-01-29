package target

import (
	"fmt"
	"io"
	"os"
	"path"

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
		recipe:   recipe,
		provider: cloudProvider.(provider.CloudProvider),
		backend:  cloudBackend.(backend.CloudBackend),

		cli:          cli,
		configInputs: make(map[string]terraform.Input),
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

		inputForm  forms.InputForm
		inputField *forms.InputField

		value *string
		vars  map[string]string
	)

	// set environment variables
	if inputForm, err = b.provider.InputForm(); err != nil {
		return err
	}
	vars = make(map[string]string)
	for _, inputField = range inputForm.InputFields() {
		if value = inputField.Value(); value != nil {
			for _, envVar := range inputField.EnvVars() {
				vars[envVar] = *value
			}
		}
	}
	runner.SetEnv(vars)

	return nil
}

// prepare and return template variables
func (b *Builder) getTemplateVars() (map[string]string, error) {

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

		if value = inputField.Value(); value == nil {
			return nil, fmt.Errorf(
				"recipe '%s' input field '%s' was nil",
				b.recipe.Name(),
				inputField.Name())
		}
		vars[inputField.Name()] = *value
	}

	return vars, nil
}

// initialize the target
func (b *Builder) Initialize() error {

	var (
		err error

		runner *terraform.Runner

		inputForm  forms.InputForm
		inputField *forms.InputField

		value *string
		vars  map[string]string
	)

	if runner, err = b.newRunner(); err != nil {
		return err
	}

	// set backend arguments
	if inputForm, err = b.backend.InputForm(); err != nil {
		return err
	}
	vars = make(map[string]string)
	for _, inputField = range inputForm.InputFields() {

		if value = inputField.Value(); value == nil {
			return fmt.Errorf(
				"backend '%s' input field '%s' was nil",
				b.backend.Name(),
				inputField.Name())
		}
		vars[inputField.Name()] = *value
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
	statePath := path.Join(
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
		if vars, err = b.getTemplateVars(); err == nil {
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
		if vars, err = b.getTemplateVars(); err == nil {
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

		runner *terraform.Runner
	)

	if runner, err = b.newRunner(); err == nil {
		err = runner.Destroy()
	}
	return err
}
