package cookbook_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr/v2"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

		cookbookDistPath := filepath.Join(workspacePath, "dist")
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
				
				validateCoobookRecipes(c, map[string][]string{
					"test:basic":  {"aws", "google"},
					"test:simple": {"google"},
				})
			})
		})
	})

	Describe("Cookbook Runtime", func() {

		BeforeEach(func() {
			outputBuffer.Reset()
			errorBuffer.Reset()
		})

		It("creates the correct runtime directory structure", func() {

			var (
				cli run.CLI

				fi os.FileInfo
			)

			runPath := filepath.Join(workspacePath, "run", "test", "recipes", "basic", "aws", "test")
			lockFile := filepath.Join(runPath, ".terraform.lock.hcl")
			labelModule := filepath.Join(runPath, ".terraform", "modules", "label")
			moduleMeta := filepath.Join(runPath, ".terraform", "modules", "modules.json")

			r := c.GetRecipe("test:basic", "aws")
			Expect(r).NotTo(BeNil())
			
			cli, err = r.CreateCLI("test", &outputBuffer, &errorBuffer)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.WorkingDirectory()).To(Equal(runPath))

			fi, err = os.Lstat(cli.ExecutablePath())
			Expect(os.IsNotExist(err)).To(BeFalse())
			Expect(fi.Mode().Perm()&0111).NotTo(Equal(0))

			fi, err = os.Lstat(lockFile)
			Expect(os.IsNotExist(err)).To(BeFalse())
			Expect(fi.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
			fi, err = os.Lstat(labelModule)
			Expect(os.IsNotExist(err)).To(BeFalse())
			Expect(fi.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
			fi, err = os.Lstat(moduleMeta)
			Expect(os.IsNotExist(err)).To(BeFalse())
			Expect(fi.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
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

				recipeConfigs, err = utils.GetItemsWithMatchAtPath("name", "^test:basic$", actual)
				Expect(err).NotTo(HaveOccurred())
				actualRecipeConfig := recipeConfigs[0]
				logger.TraceMessage("Marshalled config parsed into a nest map structure: %# v", actualRecipeConfig)

				expected := []interface{}{}
				err = json.Unmarshal([]byte(cookbookConfigDocumentDefault), &expected)
				Expect(err).NotTo(HaveOccurred())

				recipeConfigs, err = utils.GetItemsWithMatchAtPath("name", "^test:basic$", expected)
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

var _ = Describe("Cookbook Import", func() {

	var (
		err error

		outputBuffer,
		errorBuffer strings.Builder
	)

	It("imports a cookbook", func() {
		err = test_data.EnsureCookbookIsBuilt(workspacePath)
		Expect(err).NotTo(HaveOccurred())

		cookbookDistPath := filepath.Join(workspacePath, "dist")
		box := packr.New(cookbookDistPath, cookbookDistPath)

		c1, err := cookbook.NewCookbook(box, filepath.Join(workspacePath, "import"), &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(c1).ToNot(BeNil())

		err = c1.ImportCookbook(filepath.Join(workspacePath, "import", "cookbook.zip"))
		Expect(err).NotTo(HaveOccurred())

		err = c1.Validate()
		Expect(err).NotTo(HaveOccurred())

		validateCoobookRecipes(c1, map[string][]string{
			"test:basic":  {"aws", "google"},
			"test:simple": {"google"},
			"minecraft:server": {"aws", "azure", "docker", "google"},
		})

		c2, err := cookbook.NewCookbook(box, filepath.Join(workspacePath, "import"), &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(c1).ToNot(BeNil())

		err = c2.Validate()
		Expect(err).NotTo(HaveOccurred())

		validateCoobookRecipes(c2, map[string][]string{
			"test:basic":  {"aws", "google"},
			"test:simple": {"google"},
			"minecraft:server": {"aws", "azure", "docker", "google"},
		})

		cookbookList := c2.CookbookList(false)
		Expect(len(cookbookList) == 2)
		Expect(cookbookList[0].CookbookName).To(Equal("test"))
		Expect(cookbookList[0].Imported).To(BeFalse())
		Expect(cookbookList[0].Recipes).To(Equal([]string{"basic", "simple"}))
		Expect(cookbookList[1].CookbookName).To(Equal("minecraft"))
		Expect(cookbookList[1].Imported).To(BeTrue())
		Expect(cookbookList[1].Recipes).To(Equal([]string{"server"}))

		err = c2.DeleteImportedCookbook("minecraft")
		Expect(err).NotTo(HaveOccurred())

		// ensure cookbook recipes have been deleted frm c2 instance
		validateCoobookRecipes(c2, map[string][]string{
			"test:basic":  {"aws", "google"},
			"test:simple": {"google"},
		})

		// ensure cookbook folder has been deleted
		c3, err := cookbook.NewCookbook(box, filepath.Join(workspacePath, "import"), &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(c3).ToNot(BeNil())

		validateCoobookRecipes(c3, map[string][]string{
			"test:basic":  {"aws", "google"},
			"test:simple": {"google"},
		})
	})
})

func validateCoobookRecipes(c *cookbook.Cookbook, recipeSet map[string][]string) {

	var (
		exists bool

		recipeKey string
		iaasSet   []string
	)

	recipeList := c.RecipeList()
	Expect(len(recipeList)).To(Equal(len(recipeSet)))

	for _, info := range recipeList {
		recipeKey = info.CookbookName + ":" + info.RecipeName
		iaasSet, exists = recipeSet[recipeKey]
		Expect(exists).To(BeTrue())

		Expect(len(info.IaaSList)).To(Equal(len(iaasSet)))
		for i, iaas := range iaasSet {
			Expect(info.IaaSList[i].Name()).To(Equal(iaas))
		}
	}
}

const cookbookConfigDocumentDefault = `
[
  {
    "name": "test:basic",
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
		"name": "test:simple",
		"config": {
      "google": {
        "variables": []
			}
		}
	}
]
`
