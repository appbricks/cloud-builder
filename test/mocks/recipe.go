package mocks

import (
	"io"

	"github.com/mevansam/goutils/run"

	"github.com/appbricks/cloud-builder/cookbook"

	. "github.com/onsi/gomega"

	config_mocks "github.com/mevansam/goforms/test/mocks"
)

type FakeRecipe struct {
	config_mocks.FakeConfig

	cli        run.CLI
	recipePath string

	isBastion bool
}

func NewFakeRecipe(cli run.CLI) *FakeRecipe {

	f := &FakeRecipe{
		cli: cli,
	}
	f.InitConfig("Test Recipe Input", "Input form for mock recipe for testing")
	return f
}

func (f *FakeRecipe) SetRecipePath(recipePath string) {
	f.recipePath = recipePath
}

func (f *FakeRecipe) Name() string {
	return "recipe"
}

func (f *FakeRecipe) Description() string {
	return "mock recipe for testing"
}

func (f *FakeRecipe) CreateCLI(
	workingPath string,
	outputBuffer, errorBuffer io.Writer,
) (run.CLI, error) {
	return f.cli, nil
}

func (f *FakeRecipe) ConfigPath() string {
	return f.recipePath
}

func (f *FakeRecipe) PluginPath() string {
	return "/fake/providerpath"
}

func (f *FakeRecipe) StatePath() string {
	return "/fake/statepath"
}

func (f *FakeRecipe) GetKeyFieldValues() []string {
	return nil
}

func (f *FakeRecipe) GetVariable(name string) (*cookbook.Variable, bool) {

	var (
		err    error
		exists bool
		value  *string
	)

	inputForm, _ := f.InputForm()
	inputField, err := inputForm.GetInputField(name)
	Expect(err).NotTo(HaveOccurred())

	value, exists = f.GetInternalValue(name)
	Expect(exists).To(BeTrue())

	return &cookbook.Variable{
		Name:     inputField.Name(),
		Optional: inputField.Optional(),
		Value:    value,
	}, true
}

func (f *FakeRecipe) GetVariables() []*cookbook.Variable {

	inputForm, _ := f.InputForm()
	fields := inputForm.InputFields()
	variables := make([]*cookbook.Variable, len(fields))

	for i, field := range fields {
		variable, _ := f.GetVariable(field.Name())
		variables[i] = variable
	}
	return variables
}

func (f *FakeRecipe) SetBastion() {
	f.isBastion = true
}

func (f *FakeRecipe) IsBastion() bool {
	return f.isBastion
}

func (f *FakeRecipe) ResourceInstanceList() []string {
	return []string{"instance1", "instance2", "instance3"}
}

func (f *FakeRecipe) ResourceInstanceDataList() []string {
	return []string{"data1", "data2"}
}

func (f *FakeRecipe) BackendType() string {
	return "fake"
}

func (f *FakeRecipe) RepoTimestamp() string {
	return "faketimestamp"
}

func (f *FakeRecipe) CookbookName() string {
	return "fakecookebook"
}

func (f *FakeRecipe) CookbookVersion() string {
	return "fakeversion"
}

func (f *FakeRecipe) RecipeName() string {
	return "fakerecipe"
}

func (f *FakeRecipe) RecipeIaaS() string {
	return "fakeiaas"
}
func (r *FakeRecipe) RecipeKey() string {
	return r.CookbookName() + ":" + r.RecipeName()
}

func (f *FakeRecipe) AddEnvVars(vars map[string]string) {
}