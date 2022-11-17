package cookbook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/otiai10/copy"

	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/goforms/config"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"

	forms_config "github.com/appbricks/cloud-builder/forms"
)

const (
	lockFileName  = ".terraform.lock.hcl"
	contextFolder = ".terraform"
	modulesFolder = "modules"
)

type Recipe interface {
	config.Configurable

	CreateCLI(
		workingPath string,
		outputBuffer, errorBuffer io.Writer,
	) (run.CLI, error)

	ConfigPath() string
	PluginPath() string

	GetVariable(name string) (*Variable, bool)
	GetVariables() []*Variable
	GetKeyFieldValues() []string

	IsBastion() bool
	ResourceInstanceList() []string
	ResourceInstanceDataList() []string

	BackendType() string

	CookbookTimestamp() string
}

type Variable struct {
	Name     string  `json:"name"`
	Value    *string `json:"value"`
	Optional bool    `json:"optional"`
}

type recipe struct {
	name,
	description string

	variables map[string]*Variable
	keyFields []string

	isBastion                bool
	resourceInstanceList     []string
	resourceInstanceDataList []string

	backendType string

	// Paths to terraform templates and workspace
	tfConfigPath,
	tfPluginPath,
	tfCLIPath,
	workingDirectory,
	cookbookTimestamp string
}

func NewRecipe(
	name,
	iaas,
	tfConfigPath,
	tfPluginPath,
	tfCLIPath,
	workingDirectory,
	cookbookTimestamp string,
) (Recipe, error) {

	var (
		err error
	)

	// load terraform configuration
	reader := terraform.NewConfigReader()
	if err = reader.ReadMetadata(name, iaas, tfConfigPath); err != nil {
		return nil, err
	}

	recipe := &recipe{
		name:        reader.InputForm().Name(),
		description: reader.InputForm().Description(),

		variables: make(map[string]*Variable),
		keyFields: reader.KeyFields(),

		isBastion:                reader.IsBastion(),
		resourceInstanceList:     reader.ResourceInstanceList(),
		resourceInstanceDataList: reader.ResourceInstanceDataList(),

		backendType: reader.BackendType(),

		tfConfigPath:     tfConfigPath,
		tfPluginPath:     tfPluginPath,
		tfCLIPath:        tfCLIPath,
		workingDirectory: workingDirectory,

		cookbookTimestamp: cookbookTimestamp,
	}
	for _, f := range reader.InputForm().InputFields() {
		recipe.variables[f.Name()] = &Variable{
			Name:     f.Name(),
			Optional: f.Optional(),
		}
	}

	// Ensure variables are bound
	if _, err = recipe.InputForm(); err != nil {
		return nil, err
	}

	return recipe, nil
}

func (r *recipe) validate() error {

	var (
		err  error
		info os.FileInfo
	)

	if info, err = os.Stat(r.tfConfigPath); os.IsNotExist(err) {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"the recipe's terraform config template path at '%s' is not a directory",
			r.tfConfigPath)
	}

	if info, err = os.Stat(r.tfCLIPath); os.IsNotExist(err) {
		return err
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf(
			"the terraform cli at '%s' is not an executable binary",
			r.tfCLIPath)
	}

	if info, err = os.Stat(r.workingDirectory); os.IsNotExist(err) {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"the CLI's working path at '%s' is not a directory",
			r.workingDirectory)
	}

	return nil
}

// Prepare the runtime folder and return
// an instance of a CLI pointing to it
//
// out: terraform cli
func (r *recipe) CreateCLI(
	workingPath string,
	outputBuffer, errorBuffer io.Writer,
) (run.CLI, error) {

	var (
		err error
	)

	workingDirectory := filepath.Join(r.workingDirectory, workingPath)

	if err = os.MkdirAll(
		filepath.Join(workingDirectory, contextFolder),
		os.ModePerm,
	); err != nil {
		return nil, err
	}

	cloudTemplatePath := filepath.Join(r.tfConfigPath, "cloud.tf")
	if _, err = os.Stat(cloudTemplatePath); os.IsNotExist(err) {
		// create a backend template file with backend type declaration as
		// there seems to be a bug in terraform where 'output' and 'taint'
		// commands are unable to load the backend state when the working
		// directory does not have the backend resource declared even if
		// it is declared in the template directory.
		backendTemplateFile := filepath.Join(workingDirectory, "backend.tf")
		if _, err = os.Stat(backendTemplateFile); os.IsNotExist(err) {
			if err = os.WriteFile(
				backendTemplateFile,
				[]byte(
					fmt.Sprintf(
						"terraform {\n  backend \"%s\" {}\n}\n",
						r.backendType,
					),
				),
				0644,
			); err != nil {
				return nil, err
			}
		}
	} else {
		// the cloud.tf template should contain all
		// provider information for the recipe 
		if err = r.linkRecipeAsset(
			cloudTemplatePath, 
			filepath.Join(workingDirectory, "cloud.tf"),
		); err != nil {
			return nil, err
		}	
	}

	recipeLockPath := filepath.Join(r.tfConfigPath, lockFileName)
	runLockLink := filepath.Join(workingDirectory, lockFileName)
	if err = r.linkRecipeAsset(recipeLockPath, runLockLink); err != nil {
		return nil, err
	}

	recipeModulePath := filepath.Join(r.tfConfigPath, contextFolder, modulesFolder)
	runModuleLink := filepath.Join(workingDirectory, contextFolder, modulesFolder)
	if err = r.linkRecipeAsset(recipeModulePath, runModuleLink); err != nil {
		return nil, err
	}

	return run.NewCLI(
		r.tfCLIPath,
		workingDirectory,
		outputBuffer,
		errorBuffer,
	)
}

func (r *recipe) linkRecipeAsset(recipePath, runLink string) error {

	var (
		err    error
		assets []fs.DirEntry

		src, dest     string
		fiSrc, fiDest os.FileInfo
	)

	logger.TraceMessage(
		"Linking recipe runtime assets: %s => %s",
		recipePath, runLink)

	link := func(linkSrc, linkDest string) error {

		if runtime.GOOS == "windows" {
			// terraform does not follow symlinks in
			// windows so make physical copy of provider					
			if err = copy.Copy(
				linkSrc, 
				linkDest, 
				copy.Options{
					OnSymlink: func(src string) copy.SymlinkAction {
						return copy.Deep
					},
				},
			); err != nil {
				return err
			}

		} else {
			os.Remove(linkDest)
			if err = os.Symlink(linkSrc, linkDest); err != nil {
				return err
			}
		}
		return nil
	}

	if fiSrc, err = os.Stat(recipePath); !os.IsNotExist(err) {
		if fiSrc.Mode().IsDir() {

			if err = os.MkdirAll(runLink, os.ModePerm); err != nil {
				return err
			}
			if assets, err = os.ReadDir(recipePath); err != nil {
				return err
			}
			// link all assets in given path to run link path
			for _, f := range assets {
	
				src = filepath.Join(recipePath, f.Name())
				if fiSrc, err = os.Stat(src); err != nil {
					return err
				}
	
				dest = filepath.Join(runLink, f.Name())
				if fiDest, err = os.Stat(dest); os.IsNotExist(err) ||
					fiSrc.ModTime().After(fiDest.ModTime()) {
					
					if err = link(src, dest); err != nil {
						return err
					}
				}
			}

		} else {
			if fiDest, err = os.Stat(runLink); os.IsNotExist(err) ||
				fiSrc.ModTime().After(fiDest.ModTime()) {
				
				// make a direct link
				if err = link(recipePath, runLink); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// out: terraform configuration template path
func (r *recipe) ConfigPath() string {
	return r.tfConfigPath
}

func (r *recipe) PluginPath() string {
	return r.tfPluginPath
}

func (r *recipe) GetVariable(name string) (*Variable, bool) {

	var (
		exists   bool
		variable *Variable
	)

	if variable, exists = r.variables[name]; !exists {
		return nil, false
	}
	return variable, exists
}

func (r *recipe) GetVariables() []*Variable {

	ff := forms_config.RecipeConfigForms.Group(r.name).InputFields()
	l := len(ff)
	variables := make([]*Variable, l)

	for i, f := range ff {
		variables[i] = r.variables[f.Name()]
	}

	return variables
}

// out: the recipe config specific key value to use for the recipe target
func (r *recipe) GetKeyFieldValues() []string {

	var (
		variable *Variable
	)

	keyValues := make([]string, len(r.keyFields))
	for i, field := range r.keyFields {
		variable = r.variables[field]
		if variable != nil && variable.Value != nil {
			keyValues[i] = *variable.Value
		} else {
			keyValues[i] = ""
		}
	}

	return keyValues
}

// out: true if this is a cloud builder bastion recipe. this means that
//      the cloud builder apps can use this information to provide
//      additional services aganst on targets.
func (r *recipe) IsBastion() bool {
	return r.isBastion
}

// out: list of resource identifiers of instances of a deployed target.
//      this information is used to invoke clean rebuilds to fix
//      persistent problems.
func (r *recipe) ResourceInstanceList() []string {
	return r.resourceInstanceList
}

// out: list of resource identifiers of data stores of a deployed target.
//      this information is used to invoke clean rebuilds to fix
//      persistent problems.
func (r *recipe) ResourceInstanceDataList() []string {
	return r.resourceInstanceDataList
}

// out: backend type where the state of an executed recipe will be saved
func (r *recipe) BackendType() string {
	return r.backendType
}

// out: the version timestamp of the cookbook this recipe is
//      associated with.
func (r *recipe) CookbookTimestamp() string {
	return r.cookbookTimestamp
}

// interface: config/Config functions for base cloud provider

func (r *recipe) Name() string {
	return r.name
}

func (r *recipe) Description() string {
	return r.description
}

func (r *recipe) InputForm() (forms.InputForm, error) {

	var (
		err   error
		field *forms.InputField
	)

	form := forms_config.RecipeConfigForms.Group(r.name)
	for k, v := range r.variables {
		if field, err = form.GetInputField(k); err != nil {
			logger.DebugMessage(
				"Persisted recipe variable not found in loaded recipe. It will be removed from saved config: %s",
				 err.Error())

			delete(r.variables, k)
			continue
		}
		if err = field.SetValueRef(&v.Value); err != nil {
			return nil, err
		}
	}
	return form, nil
}

func (r *recipe) GetValue(name string) (*string, error) {

	var (
		err  error
		form forms.InputForm
	)

	if form, err = r.InputForm(); err != nil {
		return nil, err
	}
	return form.GetFieldValue(name)
}

func (r *recipe) Copy() (config.Configurable, error) {

	copy := &recipe{
		name:        r.name,
		description: r.description,

		variables: make(map[string]*Variable),
		keyFields: r.keyFields,

		isBastion:                r.isBastion,
		resourceInstanceList:     r.resourceInstanceList,
		resourceInstanceDataList: r.resourceInstanceDataList,

		backendType: r.backendType,

		tfConfigPath:     r.tfConfigPath,
		tfPluginPath:     r.tfPluginPath,
		tfCLIPath:        r.tfCLIPath,
		workingDirectory: r.workingDirectory,

		cookbookTimestamp: r.cookbookTimestamp,
	}

	for k, v := range r.variables {

		if v.Value == nil {
			copy.variables[k] = &Variable{
				Name:     v.Name,
				Value:    nil,
				Optional: v.Optional,
			}
		} else {
			value := *v.Value
			copy.variables[k] = &Variable{
				Name:     v.Name,
				Value:    &value,
				Optional: v.Optional,
			}
		}
	}
	return copy, nil
}

func (r *recipe) IsValid() bool {

	for _, v := range r.variables {

		if !v.Optional && v.Value == nil {
			logger.TraceMessage(
				"Required variable '%s' for recipe '%s' has not been set.",
				v.Name, r.name)

			return false
		}
	}
	return true
}

func (r *recipe) Reset() {
}

// interface: encoding/json/Unmarshaler

func (r *recipe) UnmarshalJSON(b []byte) error {

	var (
		err   error
		token json.Token
	)
	decoder := json.NewDecoder(bytes.NewReader(b))

	for decoder.More() {
		if token, err = decoder.Token(); err != nil {
			break
		}

		switch t := token.(type) {
		case string:
			if t == "variables" {
				if err = r.unmarshalVariables(decoder); err != nil {
					break
				}
			}
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func (r *recipe) unmarshalVariables(decoder *json.Decoder) error {

	var (
		err error
	)

	// read array open bracket
	if _, err = utils.ReadJSONDelimiter(decoder, utils.JsonArrayStartDelim); err != nil {
		return err
	}

	for decoder.More() {

		// read variable
		variable := &Variable{}
		if err = decoder.Decode(variable); err != nil {
			return err
		}
		r.variables[variable.Name] = variable
	}

	// read array close bracket
	_, err = utils.ReadJSONDelimiter(decoder, utils.JsonArrayEndDelim)
	return err
}

// interface: encoding/json/Marshaler

func (r *recipe) MarshalJSON() ([]byte, error) {

	var (
		err error
		out bytes.Buffer
	)

	out.WriteRune('{')

	// Marshal recipe variables
	if err = r.marshalVariables(&out); err != nil {
		return nil, err
	}

	out.WriteRune('}')
	return out.Bytes(), nil
}

func (r *recipe) marshalVariables(out *bytes.Buffer) error {

	var (
		err   error
		first bool
	)
	encoder := json.NewEncoder(out)

	out.WriteString("\"variables\":[")
	first = true
	for _, v := range r.variables {
		if v.Value != nil {

			if first {
				first = false
			} else {
				out.WriteRune(',')
			}
			if err = encoder.Encode(v); err != nil {
				return err
			}
		}
	}
	out.WriteRune(']')
	return nil
}
