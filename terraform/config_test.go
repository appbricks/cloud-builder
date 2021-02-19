package terraform_test

import (
	"fmt"
	"path/filepath"

	"github.com/appbricks/cloud-builder/terraform"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Test", func() {

	var (
		err error

		testRecipePath string
	)

	BeforeEach(func() {
		testRecipePath, err = filepath.Abs(fmt.Sprintf("%s/../test/fixtures/recipes/basic/aws", sourceDirPath))
		Expect(err).NotTo(HaveOccurred())
	})

	Context("terraform templates", func() {

		It("parses terraform templates and retrieves cloud builder metadata", func() {

			expectedVariablesInOrder := []string{
				"test_input_5",
				"test_input_1",
				"test_input_3",
				"test_input_2",
				"test_input_7",
				"test_input_4",
				"test_input_6",
			}

			reader := terraform.NewConfigReader()
			err = reader.ReadMetadata("basic", "aws", testRecipePath)
			Expect(err).NotTo(HaveOccurred())

			form := reader.InputForm()
			Expect(form).ToNot(BeNil())

			Expect(reader.KeyFields()).To(Equal([]string{"test_input_1", "test_input_2"}))
			Expect(reader.IsBastion()).To(BeTrue())
			Expect(reader.ResourceInstanceList()).To(Equal([]string{"instance1", "instance2", "instance3"}))
			Expect(reader.ResourceInstanceDataList()).To(Equal([]string{"data1", "data2"}))
			Expect(reader.BackendType()).To(Equal("s3"))

			Expect(form.Description()).To(Equal("Basic Test Recipe for AWS"))
			for i, f := range form.InputFields() {
				Expect(f.Name()).To(Equal(expectedVariablesInOrder[i]))
			}
		})
	})
})
