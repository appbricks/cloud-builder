package target_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/utils"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cloud_test_data "github.com/mevansam/gocloud/test/data"
	test_data "github.com/appbricks/cloud-builder/test/data"
)

var _ = Describe("Target", func() {

	var (
		err error

		outputBuffer strings.Builder

		r cookbook.Recipe
		p provider.CloudProvider
		b backend.CloudBackend

		t *target.Target

		form forms.InputForm
	)

	BeforeEach(func() {

		var (
			testRecipePath string
		)

		outputBuffer.Reset()

		testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes/basic/aws", sourceDirPath))
		Expect(err).NotTo(HaveOccurred())
		r, err = cookbook.NewRecipe("basic", "aws", testRecipePath, "", "", "")
		Expect(err).NotTo(HaveOccurred())

		p, err = provider.NewCloudProvider("aws")
		Expect(err).NotTo(HaveOccurred())
		b, err = backend.NewCloudBackend("s3")
		Expect(err).NotTo(HaveOccurred())

		t = target.NewTarget(r, p, b)
	})

	Context("target persistance", func() {

		It("serializes a target configuration", func() {

			form, err = r.InputForm()
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("test_input_1", "bb")
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("test_input_5", "test for input 5")
			Expect(err).NotTo(HaveOccurred())

			form, err = p.InputForm()
			err = form.SetFieldValue("access_key", "aws access key")
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("secret_key", "aws secret key")
			Expect(err).NotTo(HaveOccurred())

			form, err = b.InputForm()
			err = form.SetFieldValue("bucket", "s3 bucket")
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("key", "s3 object key")
			Expect(err).NotTo(HaveOccurred())

			encoder := json.NewEncoder(&outputBuffer)
			err := encoder.Encode(t)
			Expect(err).NotTo(HaveOccurred())

			actual := make(map[string]interface{})
			err = json.Unmarshal([]byte(outputBuffer.String()), &actual)
			Expect(err).NotTo(HaveOccurred())

			// ensure array of recipe variables is sorted
			// as otherwise the comparison will fail
			variables, err := utils.GetValueAtPath("recipe/variables", actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(variables).ToNot(BeNil())
			err = utils.SortValueMap("name", variables)
			Expect(err).NotTo(HaveOccurred())

			expected := make(map[string]interface{})
			err = json.Unmarshal([]byte(expectedTargetConfig), &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})

		It("deserializes a target configuration", func() {
			err = json.Unmarshal([]byte(testTargetConfig), t)
			Expect(err).NotTo(HaveOccurred())

			Expect(t.Key()).To(Equal("basic/aws/aa/"))
			Expect(t.DeploymentName()).To(Equal("NONAME"))
			Expect(t.RecipeName).To(Equal("basic"))
			Expect(t.RecipeIaas).To(Equal("aws"))

			test_data.ValidatePersistedVariables(
				t.Recipe.(cookbook.Recipe).GetVariables(),
				test_data.AWSBasicRecipeVariables1AsMap,
			)
		})
	})
})

const expectedTargetConfig = `{
  "recipeName": "basic",
  "recipeIaas": "aws",
  "recipe": {
    "variables": [
      {
        "name": "test_input_1",
        "value": "bb",
        "optional": false
      },
      {
        "name": "test_input_3",
        "value": "abcd3",
        "optional": true
      },
      {
        "name": "test_input_4",
        "value": "abcd4",
        "optional": true
      },
      {
        "name": "test_input_5",
        "value": "test for input 5",
        "optional": false
      },
      {
        "name": "test_input_6",
        "value": "abcd6",
        "optional": true
      },
      {
        "name": "test_input_7",
        "value": "us-east-1",
        "optional": true
      }
    ]
  },
  "provider": {
    "access_key": "aws access key",
		"secret_key": "aws secret key",
		"region": "us-east-1"
  },
  "backend": {
		"bucket": "s3 bucket",
		"key": "s3 object key"
	}
}`

const testTargetConfig = `{
  "recipeName": "basic",
	"recipeIaas": "aws",
	"recipe": {
		"variables": ` + test_data.AWSBasicRecipeVariables1 + `
	},
	"provider": ` + cloud_test_data.AWSProviderConfig + `,
	"backend": ` + cloud_test_data.S3BackendConfig + `
}`
