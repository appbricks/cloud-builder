package config_test

import (
	"bytes"
	"strings"

	"github.com/appbricks/cloud-builder/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Device Context", func() {

	It("loads and saves the auth context", func() {

		deviceContext := config.NewDeviceContext()
		err := deviceContext.Load(strings.NewReader(deviceContextDocument))
		Expect(err).ToNot(HaveOccurred())

		user, exists := deviceContext.GetPrimaryUser()
		Expect(exists).To(BeTrue())
		Expect(user).To(Equal("username"))

		// change primary user
		deviceContext.SetPrimaryUser("johnd")

		var buffer bytes.Buffer
		err = deviceContext.Save(&buffer)
		Expect(err).ToNot(HaveOccurred())

		deviceContextNew := config.NewDeviceContext()
		err = deviceContextNew.Load(bytes.NewReader(buffer.Bytes()))
		Expect(err).ToNot(HaveOccurred())

		user, exists = deviceContext.GetPrimaryUser()
		Expect(exists).To(BeTrue())
		Expect(user).To(Equal("johnd"))
	})
})

const deviceContextDocument = `
{
	"primary": {
		"name": "username",
		"rsaPrivateKey": "rsa private key",
		"rsaPublicKey": "rsa public key",
		"wgPrivateKey": "wg private key",
		"wgPublickKey": "wg public key"
	},
	"deviceID": "some device id"
}
`
