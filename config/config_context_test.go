package config_test

import (
	"encoding/json"
	"strings"

	"github.com/gobuffalo/packr/v2"

	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/appbricks/cloud-builder/test/data"
	cloud_test_data "github.com/mevansam/gocloud/test/data"
)

var _ = Describe("Config Context", func() {

	var (
		err error

		// Configuration
		ctx config.Context

		outputBuffer, errorBuffer strings.Builder
	)

	BeforeEach(func() {

		var (
			cb *cookbook.Cookbook
		)

		outputBuffer.Reset()

		err = test_data.EnsureCookbookIsBuilt(workspacePath)
		Expect(err).NotTo(HaveOccurred())

		cookbookDistPath := workspacePath + "/dist"
		box := packr.New(cookbookDistPath, cookbookDistPath)

		cb, err = cookbook.NewCookbook(box, workspacePath, &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(cb).ToNot(BeNil())

		ctx, err = config.NewConfigContext(cb)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("cloud config document", func() {

		var (
			cpAWS, cpGoogle, cpAzure provider.CloudProvider
			tgt1, tgt2               *target.Target
		)

		BeforeEach(func() {

			err = ctx.Load(strings.NewReader(configDocument))
			Expect(err).NotTo(HaveOccurred())

			// get config context providers
			cpAWS, err = ctx.GetCloudProvider("aws")
			Expect(err).NotTo(HaveOccurred())
			cpGoogle, err = ctx.GetCloudProvider("google")
			Expect(err).NotTo(HaveOccurred())
			cpAzure, err = ctx.GetCloudProvider("azure")
			Expect(err).NotTo(HaveOccurred())

			// get config context targets
			tgt1, err = ctx.GetTarget("basic/aws/aa/")
			Expect(err).NotTo(HaveOccurred())
			tgt2, err = ctx.GetTarget("basic/aws/cc/appbrickscookbook")
			Expect(err).NotTo(HaveOccurred())
		})

		It("reads a configuration document", func() {

			cloud_test_data.ValidateAWSConfigDocument(cpAWS)
			cloud_test_data.ValidateGoogleConfigDocument(cpGoogle)
			cloud_test_data.ValidateAzureConfigDocument(cpAzure)

			test_data.ValidateCookbookConfigDocument(ctx.Cookbook())

			Expect(tgt1).ToNot(BeNil())
			Expect(tgt1.RecipeName).To(Equal("basic"))
			Expect(tgt1.RecipeIaas).To(Equal("aws"))
			cloud_test_data.ValidateAWSConfigDocument(tgt1.Provider.(provider.CloudProvider))

			test_data.ValidatePersistedVariables(
				tgt1.Recipe.(cookbook.Recipe).GetVariables(),
				test_data.AWSBasicRecipeVariables1AsMap,
			)

			Expect(tgt2).ToNot(BeNil())
			Expect(tgt2.RecipeName).To(Equal("basic"))
			Expect(tgt2.RecipeIaas).To(Equal("aws"))
			cloud_test_data.ValidateAWSConfigDocument(tgt1.Provider.(provider.CloudProvider))

			test_data.ValidatePersistedVariables(
				tgt2.Recipe.(cookbook.Recipe).GetVariables(),
				test_data.AWSBasicRecipeVariables2AsMap,
			)
		})

		It("writes a configuration document", func() {

			var (
				actual, expected interface{}
			)

			err = ctx.Save(&outputBuffer)
			Expect(err).NotTo(HaveOccurred())

			actualConfigData := make(map[string]interface{})
			err = json.Unmarshal([]byte(outputBuffer.String()), &actualConfigData)
			Expect(err).NotTo(HaveOccurred())

			expectedConfigData := make(map[string]interface{})
			err = json.Unmarshal([]byte(configDocument), &expectedConfigData)
			Expect(err).NotTo(HaveOccurred())

			// Validate cloud provider configs
			actual, err = utils.GetValueAtPath("cloud/providers", actualConfigData)
			Expect(err).NotTo(HaveOccurred())
			expected, err = utils.GetValueAtPath("cloud/providers", expectedConfigData)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))

			// Validate cloud provider configs
			actual, err = utils.GetValueAtPath("cloud/backends", actualConfigData)
			Expect(err).NotTo(HaveOccurred())
			expected, err = utils.GetValueAtPath("cloud/backends", expectedConfigData)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))

			// Validate cloud recipe configs
			actual, err = utils.GetValueAtPath("cloud/recipes", actualConfigData)
			Expect(err).NotTo(HaveOccurred())
			expected, err = utils.GetValueAtPath("cloud/recipes", expectedConfigData)
			Expect(err).NotTo(HaveOccurred())

			// variable array of marshalled basic recipe must be sorted
			// in same order as expected array for deep equal to work
			recipes, ok := actual.([]interface{})
			Expect(ok).To(BeTrue())
			err = utils.SortValueMap("name", recipes)
			Expect(err).NotTo(HaveOccurred())

			recipeConfigs, err := utils.GetItemsWithMatchAtPath("name", "^basic$", recipes)
			Expect(err).NotTo(HaveOccurred())
			variables, err := utils.GetValueAtPath("config/aws/variables", recipeConfigs[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(variables).ToNot(BeNil())
			err = utils.SortValueMap("name", variables)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))

			// validate target configs
			actual, err = utils.GetValueAtPath("cloud/targets", actualConfigData)
			Expect(err).NotTo(HaveOccurred())
			expected, err = utils.GetValueAtPath("cloud/targets", expectedConfigData)
			Expect(err).NotTo(HaveOccurred())

			// array of targets and array of recipe variables within each
			// target must be sorted in order for deep equal to work.
			targets, ok := actual.([]interface{})
			Expect(ok).To(BeTrue())
			err = utils.SortValueMap("recipeName", targets)
			Expect(err).NotTo(HaveOccurred())

			targetRecipeVariables, err := utils.GetValueAtPath("recipe/variables", targets[0])
			Expect(err).NotTo(HaveOccurred())
			err = utils.SortValueMap("name", targetRecipeVariables)
			Expect(err).NotTo(HaveOccurred())
			targetRecipeVariables, err = utils.GetValueAtPath("recipe/variables", targets[1])
			Expect(err).NotTo(HaveOccurred())
			err = utils.SortValueMap("name", targetRecipeVariables)
			Expect(err).NotTo(HaveOccurred())

			err = utils.SortValueMap("recipeName", actual)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})

		It("edits config elements without modifying the main config", func() {

			var (
				cp  provider.CloudProvider
				tgt *target.Target

				recipe1, recipe2 cookbook.Recipe

				form  forms.InputForm
				value *string
			)

			// provider elements
			form, err = cpAWS.InputForm()
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("access_key", "updated access_key")
			Expect(err).NotTo(HaveOccurred())

			// cloud provider key should remain unchanged in main config structure
			cp, err = ctx.GetCloudProvider("aws")
			Expect(err).NotTo(HaveOccurred())
			value, err = cp.GetValue("access_key")
			Expect(*value).To(Equal("83BFAD5B-FEAC-4019-A645-3858847CB3ED"))

			// updated provider key value should now be persisted to main config structure
			ctx.SaveCloudProvider(cpAWS)

			cp, err = ctx.GetCloudProvider("aws")
			Expect(err).NotTo(HaveOccurred())
			value, err = cp.GetValue("access_key")
			Expect(*value).To(Equal("updated access_key"))

			// recipe elements
			recipe1, err = ctx.GetCookbookRecipe("basic", "aws")
			Expect(err).NotTo(HaveOccurred())

			form, err = recipe1.InputForm()
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("test_input_6", "abcd666")
			Expect(err).NotTo(HaveOccurred())

			// recipe key should remain unchanged in main config structure
			recipe2, err = ctx.GetCookbookRecipe("basic", "aws")
			Expect(err).NotTo(HaveOccurred())
			value, err = recipe2.GetValue("test_input_6")
			Expect(*value).To(Equal("abcd66"))

			// updated recipe key value should now be persisted to main config structure
			ctx.SaveCookbookRecipe(recipe1)

			recipe2, err = ctx.GetCookbookRecipe("basic", "aws")
			Expect(err).NotTo(HaveOccurred())
			value, err = recipe2.GetValue("test_input_6")
			Expect(*value).To(Equal("abcd666"))

			// target elements
			form, err = tgt1.Recipe.InputForm()
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("test_input_7", "eu-central-1")
			Expect(err).NotTo(HaveOccurred())

			form, err = tgt1.Provider.InputForm()
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("region", "eu-central-1")
			Expect(err).NotTo(HaveOccurred())

			// updated target should now be persisted to main config structure
			tgt, err = ctx.GetTarget("basic/aws/aa/")
			Expect(err).NotTo(HaveOccurred())
			value, err = tgt.Recipe.GetValue("test_input_7")
			Expect(*value).To(Equal("us-east-1"))
			value, err = tgt.Provider.GetValue("region")
			Expect(*value).To(Equal("us-east-1"))

			// updated target should now be persisted to main config structure
			ctx.SaveTarget("basic/aws/aa/", tgt1)

			tgt, err = ctx.GetTarget("basic/aws/aa/")
			Expect(err).NotTo(HaveOccurred())
			value, err = tgt.Recipe.GetValue("test_input_7")
			Expect(*value).To(Equal("eu-central-1"))
			value, err = tgt.Provider.GetValue("region")
			Expect(*value).To(Equal("eu-central-1"))
		})
	})
})

const configDocument = `
{
	"cloud": {
		"providers": {
			"aws": ` + cloud_test_data.AWSProviderConfig + `,
			"azure": ` + cloud_test_data.AzureProviderConfig + `,
			"google": ` + cloud_test_data.GoogleProviderConfig + `
		},
		"backends": {
			"s3": ` + cloud_test_data.S3BackendConfig + `,
			"azurerm": ` + cloud_test_data.AzureRMBackendConfig + `,
			"gcs": ` + cloud_test_data.GCSBackendConfig + `
		},
		"recipes": ` + test_data.CookbookConfigDocument + `,
		"targets": [
			{
				"recipeName": "basic",
				"recipeIaas": "aws",
				"recipe": {
					"variables": ` + test_data.AWSBasicRecipeVariables1 + `
				},
				"provider": ` + cloud_test_data.AWSProviderConfig + `,
				"backend": ` + cloud_test_data.S3BackendConfig + `
			},
			{
				"recipeName": "basic",
				"recipeIaas": "aws",
				"recipe": {
					"variables": ` + test_data.AWSBasicRecipeVariables2 + `
				},
				"provider": ` + cloud_test_data.AWSProviderConfig + `,
				"backend": ` + cloud_test_data.S3BackendConfig + `
			}
		]
	}
}
`
