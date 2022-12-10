package terraform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	hcl "github.com/hashicorp/hcl/v2"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/logger"

	forms_config "github.com/appbricks/cloud-builder/forms"
)

/**
 * Terraform Configuration Reader
 */

type configReader struct {
	inputForm *forms.InputGroup

	// the description of the recipe declared
	// via a comment with @recipe_description
	// annotation
	recipeDescription string

	// indicates that this is a cloud builder
	// bastion recipe. this means that the cloud
	// builder apps can use this information to
	// provide additional services aganst on
	// targets.
	isBastion bool

	// list of resource identifiers of instances
	// of a deployed target. this information is
	// used to invoke clean rebuilds to fix
	// persistent problems.
	resourceInstanceList []string
	// list of resource identifiers of data stores
	// of a deployed target. this information is
	// used to invoke clean rebuilds to fix
	// persistent problems.
	resourceInstanceDataList []string

	// backend where recipe state will be saved
	backendType string

	// key fields
	keyFields []string

	// content of terraform templates which
	// contain variable declarations
	templatesWithVars map[string][]string

	// annotation search regex
	variableMetadataMatch *regexp.Regexp
}

// variable metadata
type variableMetadata struct {

	// from terraform variable declaration
	name        string
	description string

	optional     bool
	defaultValue string

	typeName string

	// @display_name
	displayName string
	// @accepted_values
	acceptedValues []string
	// @accepted_values_message
	acceptedValuesMessage string
	// @value_inclusion_filter
	valueInclusionFilter string
	// @value_inclusion_filter_message
	valueInclusionFilterMessage string
	// @value_exclusion_filter
	valueExclusionFilter string
	// @value_exclusion_filter_message
	valueExclusionFilterMessage string
	// @environment_variables
	environmentVariables []string
	// @depends_on:
	dependsOn []string
	// @tags
	tags []string
	// @sensitive
	sensitive bool
	// @target_key
	key bool

	// metadata for ordering fields

	// @order + fileName + lineNumber
	order int
	// used to infer order when
	// @order is not provided
	fileName   string
	lineNumber int
}

func NewConfigReader() *configReader {
	return &configReader{
		templatesWithVars: make(map[string][]string),

		keyFields: []string{},

		variableMetadataMatch: regexp.MustCompile(`^#\s*\@([_a-z]+):\s*(.*)$`),
	}
}

func (r *configReader) ReadMetadata(
	key,
	iaas,
	configPath string,
) error {

	var (
		err error

		info os.FileInfo

		errMessage strings.Builder

		parser *configs.Parser
		module *configs.Module
		hdiag  hcl.Diagnostics

		cloudProvider provider.CloudProvider

		vm *variableMetadata

		defaultValue *string
	)

	logger.DebugMessage(
		"Loading Terraform templates for recipe '%s' and iaas '%s' at path '%s'.",
		key, iaas, configPath,
	)
	if info, err = os.Stat(configPath); err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("template path '%s' is not a directory", configPath)
	}

	// load terraform module
	parser = configs.NewParser(nil)
	module, hdiag = parser.LoadConfigDir(configPath)
	if hdiag != nil {

		for _, d := range hdiag {
			if d.Severity == hcl.DiagWarning {
				logger.DebugMessage("WARNING! %s (%s)", d.Summary, d.Detail)
			} else {
				errMessage.WriteString(fmt.Sprintf("%s (%s); ", d.Summary, d.Detail))
			}
		}
		if errMessage.Len() > 0 {
			return fmt.Errorf(
				"error parsing terraform tamplates at '%s': %s",
				configPath,
				errMessage.String(),
			)
		}
	}

	// check if recipe backend type is supported
	if module.Backend != nil {
		if !backend.IsValidCloudBackend(module.Backend.Type) {
			return fmt.Errorf("backend type '%s' is not supported", module.Backend.Type)
		}
		r.backendType = module.Backend.Type
	}

	l := len(module.Variables)
	ll := 0

	variableList := make([]*variableMetadata, 0, l)
	for _, tfVar := range module.Variables {

		logger.TraceMessage(
			"Loading metadata for terraform variable declared in template file '%s':\n%# v",
			tfVar.DeclRange.Filename, tfVar)

		if vm, err = r.readVariableMetadata(tfVar); err != nil {
			return err
		}
		if len(vm.description) > 0 {

			// add variables to array
			// sorting it along the way
			i := sort.Search(ll, func(j int) bool {
				if vm.order == variableList[j].order {
					if vm.fileName == variableList[j].fileName {
						return vm.lineNumber < variableList[j].lineNumber
					} else {
						return vm.fileName < variableList[j].fileName
					}
				} else {
					return vm.order < variableList[j].order
				}
			})
			variableList = append(variableList, nil)
			if variableList[i] != nil {
				copy(variableList[i+1:], variableList[i:])
			}
			variableList[i] = vm
			ll++
		}
	}

	logger.TraceMessage(
		"Sorted metadata of variables declared in recipe '%s':\n%# v",
		r.recipeDescription, variableList)

	// populate input form
	r.inputForm = forms_config.RecipeConfigForms.NewGroup(key + "/" + iaas, r.recipeDescription)
	for _, vm = range variableList {

		defaultValue = nil
		if vm.optional {
			defaultValue = &vm.defaultValue
		}

		if len(vm.acceptedValues) > 0 {
			switch vm.acceptedValues[0] {

			case "+iaas_regions":
				// special function populates accepted list with cloud regions
				if cloudProvider, err = provider.NewCloudProvider(iaas); err != nil {
					return err
				}
				vm.acceptedValues = []string{}
				for _, r := range cloudProvider.GetRegions() {
					vm.acceptedValues = append(vm.acceptedValues, r.Name)
				}
			}
		}

		if _, err = r.inputForm.NewInputField(forms.FieldAttributes{
			Name:         vm.name,
			DisplayName:  vm.displayName,
			Description:  vm.description,
			InputType:    forms.String,
			DefaultValue: defaultValue,
			Sensitive:    vm.sensitive,
			EnvVars:      vm.environmentVariables,
			DependsOn:    vm.dependsOn,
			Tags:         vm.tags,

			InclusionFilter:             vm.valueInclusionFilter,
			InclusionFilterErrorMessage: vm.valueInclusionFilterMessage,
			ExclusionFilter:             vm.valueExclusionFilter,
			ExclusionFilterErrorMessage: vm.valueExclusionFilterMessage,

			AcceptedValues:             vm.acceptedValues,
			AcceptedValuesErrorMessage: vm.acceptedValuesMessage,
		}); err != nil {
			return err
		}

		if vm.key {
			r.keyFields = append(r.keyFields, vm.name)
		}
	}

	return nil
}

// read variable metadata for variable declared
// in the given file and line number
func (r *configReader) readVariableMetadata(
	tfVar *configs.Variable,
) (
	*variableMetadata, error,
) {

	const maxint = int(^uint(0) >> 1)

	var (
		err        error
		exists, ok bool

		f  *os.File
		ll []string
		l  string
		m  [][]string
		o  int64
	)

	vm := &variableMetadata{
		name:        tfVar.Name,
		displayName: tfVar.Name,
		description: tfVar.Description,

		optional:     false,
		defaultValue: "",

		/// validation
		typeName:             tfVar.Type.FriendlyName(),
		acceptedValues:       []string{},
		valueInclusionFilter: "",
		valueExclusionFilter: "",
		environmentVariables: []string{},
		dependsOn:            []string{},
		sensitive:            false,

		order:      maxint,
		fileName:   strings.TrimSuffix(filepath.Base(tfVar.DeclRange.Filename), ".tf"),
		lineNumber: tfVar.DeclRange.Start.Line,
	}

	// get default value from variable
	// declaration in template
	if !tfVar.Default.IsNull() {
		vm.optional = true

		switch tfVar.Default.Type() {
		case cty.Bool:
			var val bool
			err = gocty.FromCtyValue(tfVar.Default, &val)
			if err == nil {
				vm.defaultValue = strconv.FormatBool(val)
			}
		case cty.Number:
			var val int64
			err = gocty.FromCtyValue(tfVar.Default, &val)
			if err == nil {
				vm.defaultValue = strconv.FormatInt(val, 10)
			}
		default:
			err = gocty.FromCtyValue(tfVar.Default, &vm.defaultValue)
		}
		if err != nil {
			return nil, err
		}
	}

	if ll, exists = r.templatesWithVars[vm.fileName]; !exists {
		// scan file for non-variable specific annotations

		if f, err = os.Open(tfVar.DeclRange.Filename); err != nil {
			return nil, err
		}

		ll = []string{}
		s := bufio.NewScanner(f)
		for s.Scan() {
			l = s.Text()

			if m = r.variableMetadataMatch.FindAllStringSubmatch(l, -1); len(m) > 0 {

				mval := m[0][2]
				vlen := len(mval)

				switch m[0][1] {

				case "recipe_description":
					r.recipeDescription = mval
				case "is_bastion":
					if vlen > 0 {
						if ok, err = strconv.ParseBool(mval); err != nil {
							return nil, err
						} else if ok {
							r.isBastion = true
						}
					}
				case "resource_instance_list":
					if vlen > 0 {
						r.resourceInstanceList = strings.Split(mval, ",")
					}
				case "resource_instance_data_list":
					if vlen > 0 {
						r.resourceInstanceDataList = strings.Split(mval, ",")
					}
				}
			}
			ll = append(ll, l)
		}
		r.templatesWithVars[vm.fileName] = ll
	}

	i := vm.lineNumber - 2
	for i >= 0 {
		l = ll[i]

		if strings.HasPrefix(l, "#") {
			if m = r.variableMetadataMatch.FindAllStringSubmatch(l, -1); len(m) > 0 {

				mval := m[0][2]
				vlen := len(mval)

				switch m[0][1] {

				case "display_name":
					vm.displayName = mval
				case "accepted_values":
					if vlen > 0 {
						vm.acceptedValues = strings.Split(mval, ",")
					}
				case "accepted_values_message":
					vm.acceptedValuesMessage = mval
				case "value_inclusion_filter":
					vm.valueInclusionFilter = mval
				case "value_inclusion_filter_message":
					vm.valueInclusionFilterMessage = mval
				case "value_exclusion_filter":
					vm.valueExclusionFilter = mval
				case "value_exclusion_filter_message":
					vm.valueExclusionFilterMessage = mval
				case "environment_variables":
					if vlen > 0 {
						vm.environmentVariables = strings.Split(mval, ",")
					}
				case "depends_on":
					if vlen > 0 {
						vm.dependsOn = strings.Split(mval, ",")
					}
				case "tags":
					if vlen > 0 {
						vm.tags = strings.Split(mval, ",")
					}
				case "sensitive":
					if vlen > 0 {
						if vm.sensitive, err = strconv.ParseBool(mval); err != nil {
							return nil, err
						}
					}
				case "target_key":
					if vlen > 0 {
						if ok, err = strconv.ParseBool(mval); err != nil {
							return nil, err
						} else if ok {
							vm.key = true
						}
					}
				case "order":
					if vlen > 0 {
						if o, err = strconv.ParseInt(mval, 10, 32); err != nil {
							return nil, err
						}
						vm.order = int(o)
					}
				}
			}

		} else {
			break
		}

		i--
	}

	logger.TraceMessage(
		"Loaded variable declared in template file '%s':\n%# v",
		tfVar.DeclRange.Filename, vm)

	return vm, nil
}

func (r *configReader) InputForm() forms.InputForm {
	return r.inputForm
}

func (r *configReader) KeyFields() []string {
	return r.keyFields
}

func (r *configReader) IsBastion() bool {
	return r.isBastion
}

func (r *configReader) ResourceInstanceList() []string {
	return r.resourceInstanceList
}

func (r *configReader) ResourceInstanceDataList() []string {
	return r.resourceInstanceDataList
}

func (r *configReader) BackendType() string {
	return r.backendType
}
