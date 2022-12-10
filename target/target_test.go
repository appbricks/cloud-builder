package target_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/backend"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goutils/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/appbricks/cloud-builder/test/data"
	cloud_test_data "github.com/mevansam/gocloud/test/data"
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
		r, err = cookbook.NewRecipe("basic", "aws", testRecipePath, "", "", "", "", "", "basic")
		Expect(err).NotTo(HaveOccurred())

		p, err = provider.NewCloudProvider("aws")
		Expect(err).NotTo(HaveOccurred())
		b, err = backend.NewCloudBackend("s3")
		Expect(err).NotTo(HaveOccurred())

		t, err = target.NewTarget(r, p, b).UpdateKeys()
		Expect(err).NotTo(HaveOccurred())
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
			err = form.SetFieldValue("bucket", "s3bucket")
			Expect(err).NotTo(HaveOccurred())
			err = form.SetFieldValue("key", "s3objectkey")
			Expect(err).NotTo(HaveOccurred())

			t.NodeKey = "abcd"
			t.NodeID = "1234"

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
			err = json.Unmarshal([]byte(fmt.Sprintf(expectedTargetConfig, t.RSAPrivateKey, t.RSAPublicKey)), &expected)
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
				t.Recipe.GetVariables(),
				test_data.AWSBasicRecipeVariables1AsMap,
			)
		})
	})
})

const expectedTargetConfig = `{
  "recipeName": "basic",
	"recipeIaas": "aws",
	"dependentTargets": [],
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
		"bucket": "s3bucket",
		"key": "s3objectkey"
	},
	"rsaPrivateKey": %q,
	"rsaPublicKey": %q,
	"spaceKey": "abcd",
	"spaceID": "1234"
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
