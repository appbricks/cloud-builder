package cookbook_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/term"

	test_data "github.com/appbricks/cloud-builder/test/data"
)

var _ = Describe("Recipe", func() {

	var (
		err    error
		exists bool

		outputBuffer strings.Builder

		testRecipePath string

		r cookbook.Recipe
	)

	It("returns an error when an unknown backend is declared in the template", func() {

		testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes/basic/azure", sourceDirPath))
		Expect(err).NotTo(HaveOccurred())

		r, err = cookbook.NewRecipe("basic", "azure", testRecipePath, "", "", "", "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("backend type 'error' is not supported"))
	})

	It("accepts a valid backend", func() {

		testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes/basic/google", sourceDirPath))
		Expect(err).NotTo(HaveOccurred())

		r, err = cookbook.NewRecipe("basic", "google", testRecipePath, "", "", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.BackendType()).To(Equal("gcs"))
	})

	Describe("recipe variables", func() {

		var (
			variable *cookbook.Variable
			form     forms.InputForm
			field    *forms.InputField
		)

		BeforeEach(func() {

			testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes/basic/aws", sourceDirPath))
			Expect(err).NotTo(HaveOccurred())

			r, err = cookbook.NewRecipe("basic", "aws", testRecipePath, "", "", "", "")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("variables", func() {

			It("has parsed variable metadata within comments in tf templates", func() {

				var (
					value string
					cp    provider.CloudProvider
				)

				form, err = r.InputForm()
				Expect(err).NotTo(HaveOccurred())

				field, err = form.GetInputField("test_input_1")
				Expect(err).NotTo(HaveOccurred())
				Expect(field.AcceptedValues()).To(Equal([]string{"aa", "bb", "cc", "dd"}))
				value = "dd"
				err = field.SetValue(&value)
				Expect(err).NotTo(HaveOccurred())
				value = "ee"
				err = field.SetValue(&value)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Error value #1"))

				field, err = form.GetInputField("test_input_2")
				Expect(err).NotTo(HaveOccurred())
				value = "appbrickscookbook test"
				err = field.SetValue(&value)
				Expect(err).NotTo(HaveOccurred())
				value = "test"
				err = field.SetValue(&value)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Error value inclusion #2"))
				value = "cookbook test appbricks"
				err = field.SetValue(&value)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Error value exclusion #2"))

				field, err = form.GetInputField("test_input_7")
				Expect(err).NotTo(HaveOccurred())

				cp, err = provider.NewCloudProvider("aws")
				Expect(err).NotTo(HaveOccurred())
				expcetedValues := []string{}
				for _, r := range cp.GetRegions() {
					expcetedValues = append(expcetedValues, r.Name)
				}
				Expect(field.AcceptedValues()).To(Equal(expcetedValues))
			})

			It("returns variables from all templates", func() {

				var (
					value *string
				)

				type expectedValues struct {
					optional     bool
					displayName  string
					description  string
					defaultValue *string
				}

				abcd3 := "abcd3"
				abcd4 := "abcd4"
				abcd6 := "abcd6"
				abcd7 := "us-east-1"

				variables := map[string]expectedValues{
					"test_input_1": {
						optional:     false,
						displayName:  "Test Input #1",
						description:  "Description for Test Input #1",
						defaultValue: nil,
					},
					"test_input_2": {
						optional:     false,
						displayName:  "Test Input #2",
						description:  "Description for Test Input #2",
						defaultValue: nil,
					},
					"test_input_3": {
						optional:     true,
						displayName:  "Test Input #3",
						description:  "Description for Test Input #3",
						defaultValue: &abcd3,
					},
					"test_input_4": {
						optional:     true,
						displayName:  "test_input_4",
						description:  "Description for Test Input #4",
						defaultValue: &abcd4,
					},
					"test_input_5": {
						optional:     false,
						displayName:  "Test Input #5",
						description:  "Description for Test Input #5",
						defaultValue: nil,
					},
					"test_input_6": {
						optional:     true,
						displayName:  "test_input_6",
						description:  "Description for Test Input #6",
						defaultValue: &abcd6,
					},
					"test_input_7": {
						optional:     true,
						displayName:  "Test Input #7",
						description:  "Description for Test Input #7",
						defaultValue: &abcd7,
					},
				}

				form, err := r.InputForm()
				Expect(err).NotTo(HaveOccurred())

				for n, v := range variables {

					variable, exists = r.GetVariable(n)
					Expect(exists).To(BeTrue())

					Expect(variable.Optional).To(Equal(v.optional))

					field, err = form.GetInputField(variable.Name)
					Expect(err).ToNot(HaveOccurred())
					Expect(field.DisplayName()).To(Equal(v.displayName))
					Expect(field.Description()).To(Equal(v.description))

					value = field.Value()
					if v.defaultValue == nil {
						Expect(value).To(BeNil())
					} else {
						Expect(*value).To(Equal(*v.defaultValue))
					}
				}
			})

			It("is ordered", func() {

				expectedVariablesInOrder := []string{
					"test_input_5",
					"test_input_1",
					"test_input_3",
					"test_input_2",
					"test_input_7",
					"test_input_4",
					"test_input_6",
				}

				variables := r.GetVariables()
				Expect(len(variables)).To(Equal(len(expectedVariablesInOrder)))

				for i, variable := range variables {
					Expect(variable.Name).To(Equal(expectedVariablesInOrder[i]))
				}
			})

			It("returns a target key from recipe variables", func() {

				form, err = r.InputForm()
				Expect(err).ToNot(HaveOccurred())

				err = form.SetFieldValue("test_input_1", "aa")
				Expect(err).ToNot(HaveOccurred())

				err = form.SetFieldValue("test_input_2", "cookbook")
				Expect(err).ToNot(HaveOccurred())

				Expect(r.GetKeyFieldValues()).To(Equal([]string{"aa", "cookbook"}))
			})

			It("outputs a detailed input data form reference for the basic provider", func() {

				var (
					origStdout, stdOutReader *os.File
				)

				// pipe output to be written to by form output
				origStdout = os.Stdout
				stdOutReader, os.Stdout, err = os.Pipe()
				Expect(err).ToNot(HaveOccurred())

				defer func() {
					stdOutReader.Close()
					os.Stdout = origStdout
				}()

				// channel to signal when getting form input is done
				out := make(chan string)

				go func() {

					var (
						output    bytes.Buffer
						inputForm forms.InputForm
					)

					inputForm, err = r.InputForm()
					Expect(err).ToNot(HaveOccurred())

					tf, err := ux.NewTextForm(
						"Recipe 'Basic' for AWS",
						"RECIPE DATA INPUT REFERENCE",
						inputForm)
					Expect(err).NotTo(HaveOccurred())
					tf.ShowInputReference(ux.DescOnly, 0, 2, 80)

					// close piped output
					os.Stdout.Close()
					_, err = io.Copy(&output, stdOutReader)
					Expect(err).NotTo(HaveOccurred())

					// signal end
					out <- output.String()
				}()

				// wait until signal is received

				output := <-out
				logger.DebugMessage("\n%s\n", output)
				Expect(output).To(Equal(recipeInputDataReferenceOutput))
			})
		})

		Context("persistance", func() {

			BeforeEach(func() {
				outputBuffer.Reset()
			})

			It("marshalls the variables to a json string", func() {

				form, err = r.InputForm()
				Expect(err).ToNot(HaveOccurred())

				err = form.SetFieldValue("test_input_1", "aa")
				Expect(err).ToNot(HaveOccurred())

				err = form.SetFieldValue("test_input_6", "abcd66")
				Expect(err).ToNot(HaveOccurred())

				err = form.SetFieldValue("test_input_5", "abcd5")
				Expect(err).ToNot(HaveOccurred())

				encoder := json.NewEncoder(&outputBuffer)
				err := encoder.Encode(r)
				Expect(err).NotTo(HaveOccurred())

				parsedJSON := make(map[string]interface{})
				err = json.Unmarshal([]byte(outputBuffer.String()), &parsedJSON)
				Expect(err).NotTo(HaveOccurred())

				actualVariables, ok := parsedJSON["variables"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(actualVariables).ToNot(BeNil())
				test_data.ValidateRecipeVariables(actualVariables, test_data.AWSBasicRecipeVariables1AsMap)
			})

			It("unmarshalls the variables from a json string", func() {

				var (
					value *string
				)

				jsonStream := strings.NewReader("{\"variables\":" + test_data.AWSBasicRecipeVariables1 + "}")
				decoder := json.NewDecoder(jsonStream)
				for decoder.More() {
					err = decoder.Decode(r)
					Expect(err).NotTo(HaveOccurred())
				}

				// As required variable test_input_2 is not set
				Expect(r.IsValid()).To(BeFalse())

				form, err = r.InputForm()
				Expect(err).ToNot(HaveOccurred())

				value, err = r.GetValue("test_input_1")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("aa"))

				value, err = r.GetValue("test_input_2")
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(BeNil())

				value, err = r.GetValue("test_input_3")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("abcd3"))

				value, err = r.GetValue("test_input_4")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("abcd4"))

				value, err = r.GetValue("test_input_5")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("abcd5"))

				value, err = r.GetValue("test_input_6")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("abcd66"))

				value, err = r.GetValue("test_input_7")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("us-east-1"))

				err = form.SetFieldValue("test_input_2", "cookbook")
				Expect(err).NotTo(HaveOccurred())
				Expect(r.IsValid()).To(BeTrue())

				value, err = r.GetValue("test_input_2")
				Expect(err).ToNot(HaveOccurred())
				Expect(*value).To(Equal("cookbook"))
			})
		})

		Context("config", func() {

			BeforeEach(func() {
				outputBuffer.Reset()
			})

			It("creates a copy of itself", func() {

				var (
					// value *string
					v1, v2 *string
				)

				jsonStream := strings.NewReader("{\"variables\":" + test_data.AWSBasicRecipeVariables1 + "}")
				decoder := json.NewDecoder(jsonStream)
				for decoder.More() {
					err = decoder.Decode(r)
					Expect(err).NotTo(HaveOccurred())
				}

				copy, err := r.Copy()
				Expect(err).NotTo(HaveOccurred())

				form, err = r.InputForm()
				Expect(err).ToNot(HaveOccurred())

				for _, f := range form.InputFields() {

					v1, err = r.GetValue(f.Name())
					Expect(err).NotTo(HaveOccurred())

					v2, err = copy.GetValue(f.Name())
					Expect(err).NotTo(HaveOccurred())

					if v1 == nil {
						Expect(v2).To(BeNil())
					} else {
						Expect(*v2).To(Equal(*v1))
					}
				}

				// Retrieve form again to ensure form is bound to
				// config and hange value in source config
				form, err = r.InputForm()
				Expect(err).NotTo(HaveOccurred())
				err = form.SetFieldValue("test_input_5", "random value for test_input_5")
				Expect(err).NotTo(HaveOccurred())

				// Validate value set
				v1, err = r.GetValue("test_input_5")
				Expect(err).NotTo(HaveOccurred())
				Expect(*v1).To(Equal("random value for test_input_5"))

				// Validate change does not affect copy
				v2, err = copy.GetValue("test_input_5")
				Expect(err).NotTo(HaveOccurred())
				Expect(*v2).To(Equal("abcd5"))

				// Change value in copied config
				form, err = copy.InputForm()
				Expect(err).NotTo(HaveOccurred())
				err = form.SetFieldValue("test_input_5", "random value for copied test_input_5")
				Expect(err).NotTo(HaveOccurred())

				// Validate value set in copy
				v1, err = copy.GetValue("test_input_5")
				Expect(err).NotTo(HaveOccurred())
				Expect(*v1).To(Equal("random value for copied test_input_5"))

				// Validate source value did not change
				v1, err = r.GetValue("test_input_5")
				Expect(err).NotTo(HaveOccurred())
				Expect(*v1).To(Equal("random value for test_input_5"))
			})
		})
	})
})

const recipeInputDataReferenceOutput = term.BOLD + `Recipe 'Basic' for AWS
======================` + term.NC + `

Basic Test Recipe for AWS

` + term.ITALIC + `RECIPE DATA INPUT REFERENCE` + term.NC + `

* Test Input #5 - Description for Test Input #5
* Test Input #1 - Description for Test Input #1
* Test Input #3 - Description for Test Input #3
* Test Input #2 - Description for Test Input #2
* Test Input #7 - Description for Test Input #7
* test_input_4  - Description for Test Input #4
* test_input_6  - Description for Test Input #6`
