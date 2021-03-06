package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/packr/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/mevansam/gocloud/provider"
	"github.com/mevansam/goforms/forms"

	test_data "github.com/appbricks/cloud-builder/test/data"
)

var _ = Describe("Config File", func() {

	var (
		err error

		outputBuffer,
		errorBuffer strings.Builder
		cb *cookbook.Cookbook

		cfgPath string
	)

	BeforeEach(func() {

		err = test_data.EnsureCookbookIsBuilt(workspacePath)
		Expect(err).NotTo(HaveOccurred())

		cookbookDistPath := workspacePath + "/dist"
		box := packr.New(cookbookDistPath, cookbookDistPath)

		cb, err = cookbook.NewCookbook(box, workspacePath, &outputBuffer, &errorBuffer)
		Expect(err).NotTo(HaveOccurred())
		Expect(cb).ToNot(BeNil())

		cfgPath = filepath.Join(os.TempDir(), ".cb/config.yml")
		fmt.Printf("\nTest config file path: %s\n\n", cfgPath)

		os.Remove(cfgPath)
	})

	Context("unencrypted config file", func() {

		It("initializes a config and sets some data", func() {

			var (
				cfg config.Config
			)

			cfg = initConfigFile(cfgPath, cb, "")
			updateContextWithTestData(cfg)

			err = cfg.Save()
			Expect(err).ToNot(HaveOccurred())

			// Load saved configuration and validate
			cfg = initConfigFile(cfgPath, cb, "")
			validateContextTestData(cfg)
		})
	})

	Context("encrypted config file", func() {

		It("initializes config and sets some data", func() {

			var (
				cfg config.Config
			)

			cfg = initConfigFile(cfgPath, cb, "this is a test passphrase")
			updateContextWithTestData(cfg)

			err = cfg.Save()
			Expect(err).ToNot(HaveOccurred())

			// Load saved configuration and validate
			cfg = initConfigFile(cfgPath, cb, "this is a test passphrase")
			validateContextTestData(cfg)
		})

		It("fails to read if passphrase is incorrect", func() {

			var (
				cfg config.Config
			)

			cfg = initConfigFile(cfgPath, cb, "this is a test passphrase")
			updateContextWithTestData(cfg)

			err = cfg.Save()
			Expect(err).ToNot(HaveOccurred())

			cfg, err = config.InitFileConfig(cfgPath, cb,
				// getPassphrase
				func() string {
					return "incorrect password"
				},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			err = cfg.Load()
			// config should fail to load
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cipher: message authentication failed"))
		})
	})

	Context("encrypted config file with saved passphrase", func() {

		It("initializes config and sets some data", func() {

			var (
				cfg config.Config
			)

			cfg = initConfigFile(cfgPath, cb, "this is a test passphrase")
			updateContextWithTestData(cfg)

			cfg.SetKeyTimeout(10 * time.Second)
			err = cfg.Save()
			Expect(err).ToNot(HaveOccurred())

			// get passphrase should not be called and it should be extracted from the key
			cfg, err = config.InitFileConfig(cfgPath, cb,
				// getPassphrase
				func() string {
					Fail("get passphrase called when it should have been retrieved from the saved key")
					return ""
				},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			err = cfg.Load()
			validateContextTestData(cfg)

			time.Sleep(10 * time.Second)
			getPassphraseCalled := false

			// ensure get pass phrase is called as timeout expired
			cfg, err = config.InitFileConfig(cfgPath, cb,
				// getPassphrase
				func() string {
					getPassphraseCalled = true
					return "this is a test passphrase"
				})

			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(getPassphraseCalled).To(BeTrue())

			err = cfg.Load()
			validateContextTestData(cfg)
		})

	})
})

func initConfigFile(
	cfgPath string,
	cb *cookbook.Cookbook,
	passphrase string,
) config.Config {

	var (
		err error
		cfg config.Config
	)

	cfg, err = config.InitFileConfig(cfgPath, cb,
		// getPassphrase
		func() string {
			return passphrase
		},
	)

	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = cfg.Load()
	Expect(err).ToNot(HaveOccurred())

	if !cfg.HasPassphrase() && len(passphrase) > 0 {
		cfg.SetPassphrase(passphrase)
	}
	return cfg
}

func updateContextWithTestData(cfg config.Config) {

	var (
		err error

		cp   provider.CloudProvider
		form forms.InputForm
	)

	authCtx := cfg.AuthContext()
	authCtx.SetToken(&oauth2.Token{
		AccessToken: "access token",
		RefreshToken: "refresh token",
		TokenType: "token type",
	})

	devCtx := cfg.DeviceContext()
	_, err = devCtx.NewOwnerUser("1234", "johnd")
	Expect(err).ToNot(HaveOccurred())

	ctx := cfg.Context()
	Expect(ctx).ToNot(BeNil())
	cp, err = ctx.GetCloudProvider("aws")
	Expect(err).ToNot(HaveOccurred())
	form, err = cp.InputForm()
	Expect(err).ToNot(HaveOccurred())
	err = form.SetFieldValue("access_key", "test access key")
	Expect(err).ToNot(HaveOccurred())
	err = form.SetFieldValue("secret_key", "test secret key")
	Expect(err).ToNot(HaveOccurred())
	ctx.SaveCloudProvider(cp)
}

func validateContextTestData(cfg config.Config) {

	var (
		err error

		cp    provider.CloudProvider
		value *string
	)

	authCtx := cfg.AuthContext()
	token := authCtx.GetToken()
	Expect(token.AccessToken).To(Equal("access token"))
	Expect(token.RefreshToken).To(Equal("refresh token"))
	Expect(token.TokenType).To(Equal("token type"))

	devCtx := cfg.DeviceContext()
	userID, exists := devCtx.GetOwnerUserID()
	Expect(exists).To(BeTrue())
	Expect(userID).To(Equal("1234"))
	userName, exists := devCtx.GetOwnerUserName()
	Expect(exists).To(BeTrue())
	Expect(userName).To(Equal("johnd"))

	ctx := cfg.Context()
	Expect(ctx).ToNot(BeNil())
	cp, err = ctx.GetCloudProvider("aws")
	Expect(err).NotTo(HaveOccurred())
	value, err = cp.GetValue("access_key")
	Expect(err).NotTo(HaveOccurred())
	Expect(*value).To(Equal("test access key"))
	value, err = cp.GetValue("secret_key")
	Expect(err).NotTo(HaveOccurred())
	Expect(*value).To(Equal("test secret key"))
}
