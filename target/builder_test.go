package target_test

import (
	"strings"

	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/goforms/forms"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cookbook_mocks "github.com/appbricks/cloud-builder/test/mocks"
	backend_mocks "github.com/mevansam/gocloud/test/mocks"
	provider_mocks "github.com/mevansam/gocloud/test/mocks"
	utils_mocks "github.com/mevansam/goutils/test/mocks"
)

var _ = Describe("Builder", func() {

	var (
		err error

		outputBuffer, errorBuffer strings.Builder

		cli      *utils_mocks.FakeCLI
		recipe   *cookbook_mocks.FakeRecipe
		provider *provider_mocks.FakeCloudProvider
		backend  *backend_mocks.FakeCloudBackend

		testRecipePath string

		builder *target.Builder
	)

	BeforeEach(func() {
		cli = utils_mocks.NewFakeCLI(&outputBuffer, &errorBuffer)
		recipe = cookbook_mocks.NewFakeRecipe(cli)

		provider = provider_mocks.NewFakeCloudProvider()
		backend = backend_mocks.NewFakeCloudBackend()
	})

	Describe("target deployment", func() {

		BeforeEach(func() {
			outputBuffer.Reset()
			errorBuffer.Reset()
			cli.Reset()
			recipe.Reset()
			provider.Reset()
			backend.Reset()

			testRecipePath = "a/b/c/testrecipepath"
			recipe.SetRecipePath(testRecipePath)
		})

		Context("launch", func() {

			BeforeEach(func() {

				var (
					inputForm forms.InputForm
				)

				provider.AddInputField(
					"envvar1_input",
					"display name for envvar1",
					"description name for envvar1",
					"",
					[]string{"envvar1"},
				)
				provider.AddInputField(
					"envvar2_input",
					"display name for envvar2",
					"description name for envvar2",
					"provider value 2",
					[]string{"envvar2"},
				)
				inputForm, _ = provider.InputForm()
				_ = inputForm.SetFieldValue("envvar1_input", "provider value 1")

				backend.AddInputField(
					"endpoint",
					"display name for backend endpoint",
					"description name for backend endpoint",
					"",
					[]string{},
				)
				backend.AddInputField(
					"user",
					"display name for backend user",
					"description name for backend user",
					"admin",
					[]string{},
				)
				inputForm, _ = backend.InputForm()
				_ = inputForm.SetFieldValue("endpoint", "http://backend")

				recipe.AddInputField(
					"test_input_1",
					"display name for test input 1",
					"description name for test input 1",
					"",
					[]string{},
				)
				recipe.AddInputField(
					"test_input_2",
					"display name for test input 2",
					"description name for test input 2",
					"arg value 2",
					[]string{},
				)
				recipe.AddInputField(
					"test_input_3",
					"display name for test input 3",
					"description name for test input 3",
					"default arg value 3",
					[]string{},
				)
				recipe.AddInputField(
					"test_input_4",
					"display name for test input 4",
					"description name for test input 4",
					"default arg value 4",
					[]string{},
				)
				inputForm, _ = recipe.InputForm()
				_ = inputForm.SetFieldValue("test_input_1", "arg value 1")
				_ = inputForm.SetFieldValue("test_input_4", "arg value 4")

				builder, err = target.NewBuilder(
					"test/key",
					recipe,
					provider,
					backend,
					map[string]string{
						"test_input_3": "arg value 3",
					},
					// cli buffers already set so ignored
					nil, nil,
				)
				Expect(err).NotTo(HaveOccurred())
			})

			It("initializes a target", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"init",
						"-plugin-dir=/fake/providerpath",
						"-backend-config=endpoint=http://backend",
						"-backend-config=user=admin",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
					},
					"Terraform has been successfully initialized!",
					"",
					nil,
				))

				err = builder.Initialize()
				Expect(err).NotTo(HaveOccurred())
				Expect(cli.IsExpectedRequestStackEmpty()).To(BeTrue())
			})

			It("show target's launch plan", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"plan",
						"-input=false",
						"-out=/goutils/test/cli/workingdirectory/tf.plan",
						"-var", "test_input_1=arg value 1",
						"-var", "test_input_2=arg value 2",
						"-var", "test_input_4=arg value 4",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
					},
					"Plan: 1 to add, 0 to change, 0 to destroy.",
					"",
					nil,
				))

				err = builder.ShowLaunchPlan()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(Equal("Plan: 1 to add, 0 to change, 0 to destroy."))
			})

			It("taints the target's instance resources", func() {

				for _, resource := range []string{"instance1", "instance2", "instance3"} {
					cli.ExpectFakeRequest(cli.AddFakeResponse(
						[]string{
							"taint",
							resource,
						},
						[]string{
							"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
							"TF_VAR_test_input_3=arg value 3",
							"envvar1=provider value 1",
							"envvar2=provider value 2",
						},
						resource+" tainted:",
						"",
						nil,
					))
				}

				err = builder.SetRebuildInstances()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(Equal("instance1 tainted:instance2 tainted:instance3 tainted:"))
			})

			It("taints the target's instance data resources", func() {

				for _, resource := range []string{"data1", "data2"} {
					cli.ExpectFakeRequest(cli.AddFakeResponse(
						[]string{
							"taint",
							resource,
						},
						[]string{
							"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
							"TF_VAR_test_input_3=arg value 3",
							"envvar1=provider value 1",
							"envvar2=provider value 2",
						},
						resource+" tainted:",
						"",
						nil,
					))
				}

				err = builder.SetRebuildInstanceData()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(Equal("data1 tainted:data2 tainted:"))
			})

			It("launches a target", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"plan",
						"-input=false",
						"-out=/goutils/test/cli/workingdirectory/tf.plan",
						"-var", "test_input_1=arg value 1",
						"-var", "test_input_2=arg value 2",
						"-var", "test_input_4=arg value 4",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
					},
					"Plan: 1 to add, 0 to change, 0 to destroy.",
					"",
					nil,
				))

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"-chdir=" + testRecipePath,
						"apply",
						"/goutils/test/cli/workingdirectory/tf.plan",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
					},
					"Apply complete!",
					"",
					nil,
				))

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"output",
						"-json",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
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
				))

				err = builder.Launch()
				Expect(err).NotTo(HaveOccurred())
				Expect(cli.IsExpectedRequestStackEmpty()).To(BeTrue())
			})

			It("deletes a target", func() {

				cli.ExpectFakeRequest(cli.AddFakeResponse(
					[]string{
						"destroy",
						"-auto-approve",
					},
					[]string{
						"TF_DATA_DIR=/goutils/test/cli/workingdirectory/.terraform",
						"TF_VAR_test_input_3=arg value 3",
						"envvar1=provider value 1",
						"envvar2=provider value 2",
					},
					"Destroy complete! Resources: 1 destroyed.",
					"",
					nil,
				))

				err = builder.Delete()
				Expect(err).NotTo(HaveOccurred())
				Expect(outputBuffer.String()).To(HavePrefix("Destroy complete! Resources: 1 destroyed."))
			})
		})
	})
})
