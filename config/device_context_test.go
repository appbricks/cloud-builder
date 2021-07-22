package config_test

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/appbricks/cloud-builder/config"
	"github.com/mevansam/goutils/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Device Context", func() {

	It("loads and saves the auth context", func() {

		var (
			exists bool
		)

		deviceContext := config.NewDeviceContext()
		err := deviceContext.Load(strings.NewReader(deviceContextDocument))
		Expect(err).ToNot(HaveOccurred())
		Expect(deviceContext.DeviceIDKey).To(Equal("device id key"))

		deviceID, exists := deviceContext.GetDeviceID()
		Expect(exists).To(BeTrue())
		Expect(deviceID).To(Equal("device id"))
		deviceName, exists := deviceContext.GetDeviceName()
		Expect(exists).To(BeTrue())
		Expect(deviceName).To(Equal("test device"))
		Expect(deviceContext.Device.Address).To(Equal("device.address"))
		Expect(deviceContext.Device.DNS).To(Equal([]string{"dns1", "dns2"}))

		userID, exists := deviceContext.GetOwnerUserID()
		Expect(exists).To(BeTrue())
		Expect(userID).To(Equal("user id"))
		userName, exists := deviceContext.GetOwnerUserName()
		Expect(exists).To(BeTrue())
		Expect(userName).To(Equal("user"))
		Expect(deviceContext.Owner.RSAPrivateKey).To(Equal("rsa private key"))
		Expect(deviceContext.Owner.RSAPublicKey).To(Equal("rsa public key"))
		Expect(deviceContext.Owner.WGPrivateKey).To(Equal("wg private key"))
		Expect(deviceContext.Owner.WGPublickKey).To(Equal("wg public key"))

		user1, exists := deviceContext.GetGuestUser("user1")
		Expect(exists).To(BeTrue())
		Expect(user1.UserID).To(Equal("user1 id"))
		Expect(user1.Name).To(Equal("user1"))
		user2, exists := deviceContext.GetGuestUser("user2")
		Expect(exists).To(BeTrue())
		Expect(user2.UserID).To(Equal("user2 id"))
		Expect(user2.Name).To(Equal("user2"))

		// reset device context, update it and save it

		device, err := deviceContext.NewDevice()
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.HasPrefix(device.RSAPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\n")).To(BeTrue())
		Expect(strings.HasPrefix(device.RSAPublicKey, "-----BEGIN PUBLIC KEY-----\n")).To(BeTrue())
		deviceContext.SetDeviceID("zyxw", "1234", "New Test Device")
		Expect(deviceContext.GetDeviceIDKey()).To(Equal("zyxw"))
		Expect(device.DeviceID).To(Equal("1234"))
		Expect(device.Name).To(Equal("New Test Device"))

		user, err := deviceContext.NewOwnerUser("4321", "johnd")
		Expect(err).ToNot(HaveOccurred())
		Expect(user.UserID).To(Equal("4321"))
		Expect(user.Name).To(Equal("johnd"))
		Expect(len(user.WGPrivateKey) > 0).To(BeTrue())
		Expect(len(user.WGPublickKey) > 0).To(BeTrue())

		user1, err = deviceContext.NewGuestUser("1111", "guest1")
		Expect(err).ToNot(HaveOccurred())
		Expect(user1.UserID).To(Equal("1111"))
		Expect(user1.Name).To(Equal("guest1"))
		Expect(len(user1.WGPrivateKey) > 0).To(BeTrue())
		Expect(len(user1.WGPublickKey) > 0).To(BeTrue())
		_, exists = deviceContext.GetGuestUser("guest1")
		Expect(exists).To(BeTrue())
		user2, err = deviceContext.NewGuestUser("2222", "guest2")
		Expect(err).ToNot(HaveOccurred())
		Expect(user2.UserID).To(Equal("2222"))
		Expect(user2.Name).To(Equal("guest2"))
		Expect(len(user2.WGPrivateKey) > 0).To(BeTrue())
		Expect(len(user2.WGPublickKey) > 0).To(BeTrue())
		_, exists = deviceContext.GetGuestUser("guest2")
		Expect(exists).To(BeTrue())

		Expect(deviceContext.IsAuthorizedUser("johnd"))
		Expect(deviceContext.IsAuthorizedUser("guest1"))
		Expect(deviceContext.IsAuthorizedUser("guest2"))

		var (
			buffer    bytes.Buffer
			savedJSON interface{}
		)
		err = deviceContext.Save(&buffer)
		Expect(err).ToNot(HaveOccurred())

		err = json.Unmarshal(buffer.Bytes(), &savedJSON)
		Expect(err).ToNot(HaveOccurred())

		Expect(utils.MustGetValueAtPath("deviceIDKey", savedJSON)).To(Equal("zyxw"))
		Expect(utils.MustGetValueAtPath("device/deviceID", savedJSON)).To(Equal("1234"))
		Expect(utils.MustGetValueAtPath("device/name", savedJSON)).To(Equal("New Test Device"))
		Expect(utils.MustGetValueAtPath("device/rsaPrivateKey", savedJSON)).To(Equal(device.RSAPrivateKey))
		Expect(utils.MustGetValueAtPath("device/rsaPublicKey", savedJSON)).To(Equal(device.RSAPublicKey))
		Expect(utils.MustGetValueAtPath("owner/userID", savedJSON)).To(Equal(user.UserID))
		Expect(utils.MustGetValueAtPath("owner/name", savedJSON)).To(Equal(user.Name))
		Expect(utils.MustGetValueAtPath("owner/wgPrivateKey", savedJSON)).To(Equal(user.WGPrivateKey))
		Expect(utils.MustGetValueAtPath("owner/wgPublickKey", savedJSON)).To(Equal(user.WGPublickKey))
		Expect(utils.MustGetValueAtPath("users/guest1/userID", savedJSON)).To(Equal(user1.UserID))
		Expect(utils.MustGetValueAtPath("users/guest1/name", savedJSON)).To(Equal(user1.Name))
		Expect(utils.MustGetValueAtPath("users/guest1/wgPrivateKey", savedJSON)).To(Equal(user1.WGPrivateKey))
		Expect(utils.MustGetValueAtPath("users/guest1/wgPublickKey", savedJSON)).To(Equal(user1.WGPublickKey))
		Expect(utils.MustGetValueAtPath("users/guest2/userID", savedJSON)).To(Equal(user2.UserID))
		Expect(utils.MustGetValueAtPath("users/guest2/name", savedJSON)).To(Equal(user2.Name))
		Expect(utils.MustGetValueAtPath("users/guest2/wgPrivateKey", savedJSON)).To(Equal(user2.WGPrivateKey))
		Expect(utils.MustGetValueAtPath("users/guest2/wgPublickKey", savedJSON)).To(Equal(user2.WGPublickKey))
	})
})

const deviceContextDocument = `
{
	"deviceIDKey": "device id key",
	"device": {
		"deviceID": "device id",
		"name": "test device",
		"rsaPublicKey": "device rsa public key",
		"rsaPrivateKey": "device rsa private key",
		"address": "device.address",
		"dns": [ "dns1", "dns2" ]
	},
	"owner": {
		"userID": "user id",
		"name": "user",
		"rsaPrivateKey": "rsa private key",
		"rsaPublicKey": "rsa public key",
		"wgPrivateKey": "wg private key",
		"wgPublickKey": "wg public key"
	},
	"users": {
		"user1": {
			"userID": "user1 id",
			"name": "user1",
			"rsaPrivateKey": "rsa private key #1",
			"rsaPublicKey": "rsa public key #1",
			"wgPrivateKey": "wg private key #1",
			"wgPublickKey": "wg public key #1"
		},
		"user2": {
			"userID": "user2 id",
			"name": "user2",
			"rsaPrivateKey": "rsa private key #1",
			"rsaPublicKey": "rsa public key #1",
			"wgPrivateKey": "wg private key #1",
			"wgPublickKey": "wg public key #1"
		}
	}
}
`
