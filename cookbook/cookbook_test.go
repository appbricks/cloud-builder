package cookbook_test

import (
	"encoding/json"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/appbricks/cloud-builder/cookbook"

	test_data "github.com/appbricks/cloud-builder/test/data"
)

var _ = Describe("Cookbook", func() {

	var (
		err error

		outputBuffer,
		errorBuffer strings.Builder
		c *cookbook.Cookbook
	)

	BeforeEach(func() {
		err = test_data.EnsureCookbookIsBuilt(workspacePath)
		Expect(err).NotTo(HaveOccurred())

		cookbookDistPath := workspacePath + "/dist"
		box := packr.New(cookbookDistPath, cookbookDistPath)

		c, err = cookbook.NewCookbook(box, workspacePath, &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).ToNot(BeNil())
	})

	Describe("Cookbook Recipes", func() {

		It("validates", func() {
			err = c.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("info", func() {

			It("lists all the IaaS's that the Cookbook has recipes for", func() {

				iaasSet := []string{"aws", "google"}

				iaasList := c.IaaSList()
				Expect(len(iaasList)).To(Equal(len(iaasSet)))
				Expect(iaasList[0].Name()).To(Equal(iaasSet[0]))
				Expect(iaasList[1].Name()).To(Equal(iaasSet[1]))
			})

			It("lists all the recipes in the Cookbook and the IaaS's a recipe can be launched in", func() {

				var (
					exists  bool
					iaasSet []string
				)

				recipeSet := map[string][]string{
					"basic":  []string{"aws", "google"},
					"simple": []string{"google"},
				}

				recipeList := c.RecipeList()
				Expect(len(recipeList)).To(Equal(len(recipeSet)))

				for _, info := range recipeList {
					iaasSet, exists = recipeSet[info.Name]
					Expect(exists).To(BeTrue())

					Expect(len(info.IaaSList)).To(Equal(len(iaasSet)))
					for i, iaas := range iaasSet {
						Expect(info.IaaSList[i].Name()).To(Equal(iaas))
					}
				}
			})
		})
	})

	Describe("Cookbook Persistance", func() {

		Context("persist cookbook config", func() {

			BeforeEach(func() {
				outputBuffer.Reset()
			})

			It("correctly unmarshalls cookbook configuration", func() {

				err = json.Unmarshal([]byte(test_data.CookbookConfigDocument), c)
				Expect(err).NotTo(HaveOccurred())
				test_data.ValidateCookbookConfigDocument(c)
			})

			It("correctly marshals cookbook configuration", func() {

				var (
					variables     interface{}
					recipeConfigs []interface{}
				)

				encoder := json.NewEncoder(&outputBuffer)
				err := encoder.Encode(c)
				Expect(err).NotTo(HaveOccurred())

				actual := []interface{}{}
				err = json.Unmarshal([]byte(outputBuffer.String()), &actual)
				Expect(err).NotTo(HaveOccurred())

				recipeConfigs, err = utils.GetItemsWithMatchAtPath("name", "^basic$", actual)
				Expect(err).NotTo(HaveOccurred())
				actualRecipeConfig := recipeConfigs[0]
				logger.TraceMessage("Marshalled config parsed into a nest map structure: %# v", actualRecipeConfig)

				expected := []interface{}{}
				err = json.Unmarshal([]byte(cookbookConfigDocumentDefault), &expected)
				Expect(err).NotTo(HaveOccurred())

				recipeConfigs, err = utils.GetItemsWithMatchAtPath("name", "^basic$", expected)
				Expect(err).NotTo(HaveOccurred())
				expectedRecipeConfig := recipeConfigs[0]

				// array should be sorted in same order as expected array to for deep equal to work
				variables, err = utils.GetValueAtPath("config/aws/variables", actualRecipeConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(variables).ToNot(BeNil())
				err = utils.SortValueMap("name", variables)
				Expect(err).NotTo(HaveOccurred())

				Expect(actualRecipeConfig).To(Equal(expectedRecipeConfig))
			})
		})
	})
})

const cookbookConfigDocumentDefault = `
[
  {
    "name": "basic",
    "config": {
      "aws": {
        "variables": [
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
      "google": {
        "variables": []
      }
    }
	},
	{
		"name": "simple",
		"config": {
      "google": {
        "variables": []
			}
		}
	}
]
`
