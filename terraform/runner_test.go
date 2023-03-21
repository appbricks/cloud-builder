package terraform_test

import (
	"fmt"
	"strings"

	"github.com/appbricks/cloud-builder/terraform"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var _ = Describe("Runner", func() {

	var (
		err error

		outputBuffer, errorBuffer strings.Builder

		cli    *utils_mocks.FakeCLI
		runner *terraform.Runner

		testRecipePath,
		testPluginPath string

		output map[string]terraform.Output

		planRequestKey,
		applyRequestKey,
		outputRequestKey string
	)

	BeforeEach(func() {
		cli = utils_mocks.NewFakeCLI(&outputBuffer, &errorBuffer)
	})

	Describe("terraform configuration execution", func() {

		BeforeEach(func() {
			outputBuffer.Reset()
			errorBuffer.Reset()
			cli.Reset()

			testRecipePath = "a/b/c/configpath"
			testPluginPath = "a/b/c/pluginpath"
			runner = terraform.NewRunner(cli,
				testRecipePath,
				testPluginPath,
				map[string]terraform.Input{
					"test_input": {false},
				})

			planRequestKey = cli.AddFakeResponse(
				[]string{
					"-chdir=" + testRecipePath,
					"plan",
					"-input=false",
					"-out=/goutils/test/cli/workingdirectory/tf.plan",
					"-var", "test_input=arg value 1",
				},
				[]string{
					"envvar1=envvar value 1",
					"envvar2=envvar value 2",
				},
				"Plan: 1 to add, 0 to change, 0 to destroy.",
				"",
				nil,
			)

			applyRequestKey = cli.AddFakeResponse(
				[]string{
					"-chdir=" + testRecipePath,
					"apply",
					"/goutils/test/cli/workingdirectory/tf.plan",
				},
				[]string{
					"envvar1=envvar value 1",
					"envvar2=envvar value 2",
				},
				"Apply complete!",
				"",
				nil,
			)

			outputRequestKey = cli.AddFakeResponse(
				[]string{
					"output",
					"-json",
				},
				[]string{
					"envvar1=envvar value 1",
					"envvar2=envvar value 2",
				},
				`{
					"output1": {
							"sensitive": false,
							"type": "string",
							"value": "output value 1"
					},
					"output2": {
							"sensitive": false,
							"type": "number",
							"value": "2"
					}
				}`,
				"",
				nil,
			)
		})

		Context("init", func() {

			It("executes 'terraform init'", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"init",
						"-plugin-dir=" + testPluginPath,
						"-backend-config=endpoint=http://backend",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"Terraform has been successfully initialized!",
					"",
					nil,
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				runner.SetBackend(
					map[string]string{
						"endpoint": "http://backend",
					},
				)
				err = runner.Init()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(HavePrefix("Terraform has been successfully initialized!"))
			})

			It("handles 'terraform init' failure", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"init",
						"-plugin-dir=" + testPluginPath,
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"",
					"Error: Error initializing config",
					fmt.Errorf("Error!"),
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				err = runner.Init()
				Expect(err).To(HaveOccurred())
				Expect(errorBuffer.String()).To(Equal("Error: Error initializing config"))
			})
		})

		Context("plan", func() {

			It("executes 'terraform plan' with given environment and variables and reads output", func() {

				cli.ExpectFakeRequest(planRequestKey)

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				err = runner.Plan(
					map[string]string{
						"test_input": "arg value 1",
					},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(HavePrefix("Plan: 1 to add, 0 to change, 0 to destroy."))
				Expect(errorBuffer.String()).To(Equal(""))
			})
		})

		Context("taint", func() {

			It("taints a list of given resources", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"taint",
						"resource1",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"resource1 tainted:",
					"",
					nil,
				))
				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"taint",
						"resource2",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"resource2 tainted:",
					"",
					nil,
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				err = runner.Taint(
					[]string{"resource1", "resource2"},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(Equal("resource1 tainted:resource2 tainted:"))
				Expect(errorBuffer.String()).To(Equal(""))
			})
		})

		Context("apply", func() {

			It("executes 'terraform apply' with given environment and variables and reads output", func() {

				cli.ExpectFakeRequest(planRequestKey)
				cli.ExpectFakeRequest(applyRequestKey)
				cli.ExpectFakeRequest(outputRequestKey)

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				output, err = runner.Apply(
					map[string]string{
						"test_input": "arg value 1",
					},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(output)).To(Equal(2))
				Expect(output["output1"]).To(Equal(terraform.Output{
					Sensitive: false,
					Type:      "string",
					Value:     "output value 1",
				}))
				Expect(output["output2"]).To(Equal(terraform.Output{
					Sensitive: false,
					Type:      "number",
					Value:     "2",
				}))
				Expect(outputBuffer.String()).To(HavePrefix("Plan: 1 to add, 0 to change, 0 to destroy.Apply complete!"))
				Expect(errorBuffer.String()).To(Equal(""))
			})

			It("handles 'terraform apply' failure", func() {

				cli.ExpectFakeRequest(planRequestKey)
				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"apply",
						"/goutils/test/cli/workingdirectory/tf.plan",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"",
					"Error: oops something went wrong",
					fmt.Errorf("Error!"),
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				output, err = runner.Apply(
					map[string]string{
						"test_input": "arg value 1",
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(output).To(BeNil())
				Expect(errorBuffer.String()).To(Equal("Error: oops something went wrong"))
			})
		})

		Context("output", func() {

			It("executes 'terraform output' and reads the output", func() {

				cli.ExpectFakeRequest(outputRequestKey)

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				output, err = runner.GetOutput()
				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(len(output)).To(Equal(2))
				Expect(output["output1"]).To(Equal(terraform.Output{
					Sensitive: false,
					Type:      "string",
					Value:     "output value 1",
				}))
				Expect(output["output2"]).To(Equal(terraform.Output{
					Sensitive: false,
					Type:      "number",
					Value:     "2",
				}))
			})

			It("handles 'terraform output' failure", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"output",
						"-json",
					},
					[]string{},
					"",
					"Error: getting output",
					fmt.Errorf("Error!"),
				))

				output, err = runner.GetOutput()
				Expect(err).To(HaveOccurred())
				Expect(output).To(BeNil())
				Expect(errorBuffer.String()).To(Equal("Error: getting output"))
			})
		})

		Context("destroy", func() {

			It("executes 'terraform destroy'", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"apply",
						"-destroy",
						"-auto-approve",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"Destroy complete! Resources: 1 destroyed.",
					"",
					nil,
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				err = runner.Destroy()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(HavePrefix("Destroy complete! Resources: 1 destroyed."))
			})

			It("handles 'terraform destroy' failure", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"apply",
						"-destroy",
						"-auto-approve",
					},
					[]string{
						"envvar1=envvar value 1",
						"envvar2=envvar value 2",
					},
					"",
					"Error: Error destroying resources",
					fmt.Errorf("Error!"),
				))

				runner.SetEnv(
					map[string]string{
						"envvar1": "envvar value 1",
						"envvar2": "envvar value 2",
					},
				)
				err = runner.Destroy()
				Expect(err).To(HaveOccurred())
				Expect(errorBuffer.String()).To(Equal("Error: Error destroying resources"))
			})
		})
	})

	Describe("terraform configuration variables", func() {

		BeforeEach(func() {
			outputBuffer.Reset()
			errorBuffer.Reset()
			cli.Reset()

			testRecipePath = "x/y/z/configpath"
			runner = terraform.NewRunner(cli,
				testRecipePath,
				testPluginPath,
				map[string]terraform.Input{
					"test_input_1": {false},
					"test_input_2": {false},
					"test_input_3": {true},
					"test_input_5": {false},
					"test_input_7": {true},
				})
		})

		Context("runtime", func() {

			It("succeeds if required arguments are provided when calling apply", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"plan",
						"-input=false",
						"-out=/goutils/test/cli/workingdirectory/tf.plan",
						"-var", "test_input_1=abcd1",
						"-var", "test_input_2=abcd2",
						"-var", "test_input_3=abcd3",
						"-var", "test_input_5=abcd5",
						"-var", "test_input_7=abcd7",
					},
					[]string{},
					"",
					"",
					nil,
				))
				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"apply",
						"/goutils/test/cli/workingdirectory/tf.plan",
					},
					[]string{},
					"",
					"",
					nil,
				))
				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"output",
						"-json",
					},
					[]string{},
					`{}`,
					"",
					nil,
				))

				output, err = runner.Apply(
					map[string]string{
						"test_input_1": "abcd1", // Required
						"test_input_2": "abcd2", // Required
						"test_input_3": "abcd3", // Optional
						"test_input_5": "abcd5", // Required
						"test_input_7": "abcd7", // Optional
					},
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error if required arguments are not provided when calling apply", func() {

				output, err = runner.Apply(
					map[string]string{
						"test_input_1": "abcd1", // Required
						// "test_input_2": "abcd2", // Required
						"test_input_3": "abcd3", // Optional
						"test_input_5": "abcd5", // Required
						"test_input_7": "abcd7", // Optional
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("the following required arguments were not provided: test_input_2"))
			})

			It("returns an error if an unknown argument provided when calling apply", func() {

				output, err = runner.Apply(
					map[string]string{
						"test_input_1": "abcd1", // Required
						"test_input_2": "abcd2", // Required
						"test_input_3": "abcd3", // Optional
						"test_input_5": "abcd5", // Required
						"test_input_7": "abcd7", // Optional
						"test_input_8": "abcd8", // Unknown
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("the following argument is not known by the templates: test_input_8"))
			})
		})
	})
})
