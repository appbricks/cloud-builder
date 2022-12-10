package data

import (
	"github.com/appbricks/cloud-builder/cookbook"
	. "github.com/onsi/gomega"
)

// coookbook configuration test data

const CookbookConfigDocument = `
[
  {
    "name": "test:basic",
    "config": {
      "aws": {
        "variables": [
          {
            "name": "test_input_1",
            "value": "bb",
            "optional": true
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
            "value": "abcd5",
            "optional": true
          },
          {
            "name": "test_input_6",
            "value": "abcd66",
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
        "variables": [
					{
            "name": "test_input",
            "value": "google test input",
            "optional": true
					}
				]
      }
    }
	},
	{
		"name": "test:simple",
		"config": {
      "google": {
        "variables": [
					{
            "name": "test_simple_input_1",
            "value": "simple test input",
            "optional": true
					}
				]
			}
		}
	}
]
`

func ValidateCookbookConfigDocument(ckbk *cookbook.Cookbook) {

	var (
		err error

		recipe cookbook.Recipe
		value  *string
	)

	recipe = ckbk.GetRecipe("test:basic", "aws")
	Expect(recipe).ToNot(BeNil())

	value, err = recipe.GetValue("test_input_1")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("bb"))

	value, err = recipe.GetValue("test_input_2")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).To(BeNil())

	value, err = recipe.GetValue("test_input_3")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("abcd3"))

	value, err = recipe.GetValue("test_input_4")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("abcd4"))

	value, err = recipe.GetValue("test_input_5")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("abcd5"))

	value, err = recipe.GetValue("test_input_6")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("abcd66"))

	value, err = recipe.GetValue("test_input_7")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("us-east-1"))

	recipe = ckbk.GetRecipe("test:basic", "google")
	Expect(recipe).ToNot(BeNil())

	value, err = recipe.GetValue("test_input")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("google test input"))

	recipe = ckbk.GetRecipe("test:simple", "google")
	Expect(recipe).ToNot(BeNil())

	value, err = recipe.GetValue("test_simple_input_1")
	Expect(err).NotTo(HaveOccurred())
	Expect(value).ToNot(BeNil())
	Expect(*value).To(Equal("simple test input"))
}

// shared recipe variable data

const AWSBasicRecipeVariables1 = `[
	{
		"name": "test_input_1",
		"value": "aa",
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
		"value": "abcd5",
		"optional": false
	},
	{
		"name": "test_input_6",
		"value": "abcd66",
		"optional": true
	},
	{
		"name": "test_input_7",
		"value": "us-east-1",
		"optional": true
	}
]
`

var AWSBasicRecipeVariables1AsMap = map[string]interface{}{
	"test_input_1": map[string]interface{}{
		"value":    "aa",
		"optional": false,
	},
	"test_input_3": map[string]interface{}{
		"value":    "abcd3",
		"optional": true,
	},
	"test_input_4": map[string]interface{}{
		"value":    "abcd4",
		"optional": true,
	},
	"test_input_5": map[string]interface{}{
		"value":    "abcd5",
		"optional": false,
	},
	"test_input_6": map[string]interface{}{
		"value":    "abcd66",
		"optional": true,
	},
	"test_input_7": map[string]interface{}{
		"value":    "us-east-1",
		"optional": true,
	},
}

const AWSBasicRecipeVariables2 = `[
	{
		"name": "test_input_1",
		"value": "cc",
		"optional": false
	},
	{
		"name": "test_input_2",
		"value": "appbrickscookbook",
		"optional": false
	},
	{
		"name": "test_input_3",
		"value": "abcd3",
		"optional": true
	},
	{
		"name": "test_input_4",
		"value": "abcd44",
		"optional": true
	},
	{
		"name": "test_input_5",
		"value": "abcd55",
		"optional": false
	},
	{
		"name": "test_input_6",
		"value": "abcd66",
		"optional": true
	},
	{
		"name": "test_input_7",
		"value": "eu-central-1",
		"optional": true
	}
]
`

var AWSBasicRecipeVariables2AsMap = map[string]interface{}{
	"test_input_1": map[string]interface{}{
		"value":    "cc",
		"optional": false,
	},
	"test_input_2": map[string]interface{}{
		"value":    "appbrickscookbook",
		"optional": false,
	},
	"test_input_3": map[string]interface{}{
		"value":    "abcd3",
		"optional": true,
	},
	"test_input_4": map[string]interface{}{
		"value":    "abcd44",
		"optional": true,
	},
	"test_input_5": map[string]interface{}{
		"value":    "abcd55",
		"optional": false,
	},
	"test_input_6": map[string]interface{}{
		"value":    "abcd66",
		"optional": true,
	},
	"test_input_7": map[string]interface{}{
		"value":    "eu-central-1",
		"optional": true,
	},
}

func ValidateRecipeVariables(actualVariables []interface{}, expectVariables map[string]interface{}) {

	var (
		ok   bool
		name string

		actual, expected map[string]interface{}
	)

	Expect(len(actualVariables)).To(Equal(len(expectVariables)))
	for _, v := range actualVariables {
		actual, ok = v.(map[string]interface{})
		Expect(ok).To(BeTrue())
		name, ok = actual["name"].(string)
		Expect(ok).To(BeTrue())
		expected, ok = expectVariables[name].(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(expected).ToNot(BeNil())
		Expect(actual["value"]).To(Equal(expected["value"]))
		Expect(actual["optional"]).To(Equal(expected["optional"]))
	}
}

func ValidatePersistedVariables(variables []*cookbook.Variable, expectVariables map[string]interface{}) {

	numVars := 0
	for _, v := range variables {

		if v.Value != nil {
			expected, exists := expectVariables[v.Name]
			Expect(exists).To(BeTrue())
			expectedVar := expected.(map[string]interface{})["value"]
			Expect(*v.Value).To(Equal(expectedVar))
			numVars++
		}
	}
	Expect(numVars).To(Equal(len(expectVariables)))
}
