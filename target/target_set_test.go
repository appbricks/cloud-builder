package target_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/appbricks/cloud-builder/target"

	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/appbricks/cloud-builder/test/data"
	cloud_test_data "github.com/mevansam/gocloud/test/data"

	target_mocks "github.com/appbricks/cloud-builder/test/mocks"
)

var _ = Describe("TargetSet", func() {

	var (
		err error

		outputBuffer,
		errorBuffer strings.Builder

		ctx *target_mocks.FakeTargetContext
	)

	BeforeEach(func() {

		var (
			testRecipePath string
		)

		outputBuffer.Reset()
		errorBuffer.Reset()

		testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes", sourceDirPath))
		Expect(err).NotTo(HaveOccurred())

		ctx = target_mocks.NewTargetMockContext(testRecipePath)
	})

	Context("target set persistance", func() {

		var (
			ts *target.TargetSet
		)

		BeforeEach(func() {
			ts = target.NewTargetSet(ctx)
		})

		It("deserializes a list of target configurations", func() {

			var (
				tgt1, tgt2 *target.Target
			)

			// deserialize test target set and validate
			err = json.Unmarshal([]byte(targetConfigDocument), ts)
			Expect(err).NotTo(HaveOccurred())

			tgt1 = ts.GetTarget(tgt1Key)
			Expect(tgt1).ToNot(BeNil())
			Expect(tgt1.RecipeName).To(Equal("basic"))
			Expect(tgt1.RecipeIaas).To(Equal("aws"))
			cloud_test_data.ValidateAWSConfigDocument(tgt1.Provider)

			tgtDeps := tgt1.Dependencies()
			Expect(len(tgtDeps)).To(Equal(1))
			Expect(tgtDeps[0].Key()).To(Equal(tgt2Key))

			test_data.ValidatePersistedVariables(
				tgt1.Recipe.GetVariables(),
				test_data.AWSBasicRecipeVariables1AsMap,
			)

			tgt2 = ts.GetTarget(tgt2Key)
			Expect(tgt2).ToNot(BeNil())
			Expect(tgt2.RecipeName).To(Equal("basic"))
			Expect(tgt2.RecipeIaas).To(Equal("aws"))
			cloud_test_data.ValidateAWSConfigDocument(tgt1.Provider)

			test_data.ValidatePersistedVariables(
				tgt2.Recipe.GetVariables(),
				test_data.AWSBasicRecipeVariables2AsMap,
			)
		})

		It("serializes a list of target configurations", func() {

			var (
				ok    bool
				value interface{}

				tgt       *target.Target
				inputForm forms.InputForm

				actualTargetConfig  map[string]interface{}
				expectedVariableMap map[string]interface{}
			)

			// deserialize test target set to modify and serialize
			err = json.Unmarshal([]byte(targetConfigDocument), ts)
			Expect(err).NotTo(HaveOccurred())

			// modify targets

			tgt = ts.GetTarget(tgt1Key)
			Expect(tgt).ToNot(BeNil())
			inputForm, err = tgt.Recipe.InputForm()
			Expect(err).NotTo(HaveOccurred())

			err = inputForm.SetFieldValue("test_input_2", "cookbook")
			Expect(err).NotTo(HaveOccurred())
			err = inputForm.SetFieldValue("test_input_4", "test_input_4 updated")
			Expect(err).NotTo(HaveOccurred())

			tgt.DependentTargets = []string{}
			err = ts.SaveTarget(tgt1Key, tgt)
			Expect(err).NotTo(HaveOccurred())
			Expect(ts.GetTarget(tgt1Key)).To(BeNil())
			Expect(ts.GetTarget("aa/cookbook")).ToNot(BeNil())

			tgt = ts.GetTarget(tgt2Key)
			Expect(tgt).ToNot(BeNil())
			inputForm, err = tgt.Recipe.InputForm()
			Expect(err).NotTo(HaveOccurred())

			err = inputForm.SetFieldValue("test_input_1", "bb")
			Expect(err).NotTo(HaveOccurred())
			err = inputForm.SetFieldValue("test_input_6", "test_input_6 updated")
			Expect(err).NotTo(HaveOccurred())

			tgt.DependentTargets = []string{"aa/cookbook"}
			err = ts.SaveTarget(tgt2Key, tgt)
			Expect(err).NotTo(HaveOccurred())
			Expect(ts.GetTarget(tgt2Key)).To(BeNil())
			Expect(ts.GetTarget("bb/appbrickscookbook/<aa/cookbook")).ToNot(BeNil())

			// serialize targets

			encoder := json.NewEncoder(&outputBuffer)
			err := encoder.Encode(ts)
			Expect(err).NotTo(HaveOccurred())

			// validate serialized data

			actual := []interface{}{}
			err = json.Unmarshal([]byte(outputBuffer.String()), &actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(actual)).To(Equal(2))

			for _, v := range actual {
				actualTargetConfig, ok = v.(map[string]interface{})
				Expect(ok).To(BeTrue())

				deps := actualTargetConfig["dependentTargets"].([]interface{})

				key := ""
				vv := actualTargetConfig["recipe"].(map[string]interface{})["variables"].([]interface{})
				for _, v := range vv {
					name := v.(map[string]interface{})["name"]
					if name == "test_input_1" {
						key = v.(map[string]interface{})["value"].(string)
						break
					}
				}

				switch key {

				case "aa":
					Expect(actualTargetConfig["recipeName"]).To(Equal("basic"))
					Expect(actualTargetConfig["recipeIaas"]).To(Equal("aws"))
					Expect(len(deps)).To(Equal(0))
					expectedVariableMap = utils.Copy(test_data.AWSBasicRecipeVariables1AsMap).(map[string]interface{})

					expectedVariableMap["test_input_2"] = map[string]interface{}{
						"value":    "cookbook",
						"optional": false,
					}
					expectedVariableMap["test_input_4"] = map[string]interface{}{
						"value":    "test_input_4 updated",
						"optional": true,
					}

				case "bb":
					Expect(actualTargetConfig["recipeName"]).To(Equal("basic"))
					Expect(actualTargetConfig["recipeIaas"]).To(Equal("aws"))
					Expect(len(deps)).To(Equal(1))
					Expect(deps[0]).To(Equal("aa/cookbook"))
					expectedVariableMap = utils.Copy(test_data.AWSBasicRecipeVariables2AsMap).(map[string]interface{})

					expectedVariableMap["test_input_1"] = map[string]interface{}{
						"value":    "bb",
						"optional": false,
					}
					expectedVariableMap["test_input_6"] = map[string]interface{}{
						"value":    "test_input_6 updated",
						"optional": true,
					}

				default:
					Fail(fmt.Sprintf("invalid target '%s'", key))
				}

				value, err = utils.GetValueAtPath("recipe/variables", actualTargetConfig)
				Expect(err).NotTo(HaveOccurred())
				actualVariables, ok := value.([]interface{})

				Expect(ok).To(BeTrue())
				Expect(actualVariables).ToNot(BeNil())
				test_data.ValidateRecipeVariables(actualVariables, expectedVariableMap)

				value, err = utils.GetValueAtPath("provider", actualTargetConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(map[string]interface{}{
					"access_key": "83BFAD5B-FEAC-4019-A645-3858847CB3ED",
					"secret_key": "3BA9D494-5D49-4F1A-84CA-70D10A08ACDE",
					"region":     "us-east-1",
					"token":      "E4B22688-A369-4FB1-B375-732ACED7156F",
				}))
			}
		})
	})
})

const tgt1Key = `aa//<cc/appbrickscookbook`
const tgt2Key = `cc/appbrickscookbook`

const targetConfigDocument = `
[
	{
		"recipeName": "basic",
		"recipeIaas": "aws",
		"cookbookName": "cb1",
		"dependentTargets": [ "cc/appbrickscookbook" ],
		"recipe": {
			"variables": ` + test_data.AWSBasicRecipeVariables1 + `
		},
		"provider": ` + cloud_test_data.AWSProviderConfig + `,
		"backend": ` + cloud_test_data.S3BackendConfig + `
	},
	{
		"recipeName": "basic",
		"recipeIaas": "aws",
		"cookbookName": "cb2",
		"dependentTargets": [],
		"recipe": {
			"variables": ` + test_data.AWSBasicRecipeVariables2 + `
		},
		"provider": ` + cloud_test_data.AWSProviderConfig + `,
		"backend": ` + cloud_test_data.S3BackendConfig + `
	}
]	
`
