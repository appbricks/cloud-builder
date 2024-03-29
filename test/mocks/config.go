package mocks

import (
	"time"

	"github.com/appbricks/cloud-builder/config"
)

type MockConfig struct {
	authContext   config.AuthContext
	deviceContext config.DeviceContext
	targetContext config.TargetContext
}

func NewMockConfig(
	authContext config.AuthContext,
	deviceContext config.DeviceContext,
	targetContext config.TargetContext,
) config.Config {

	return &MockConfig{
		authContext: authContext,
		deviceContext: deviceContext,
		targetContext: targetContext,
	}
}

func (mc *MockConfig) Reset() error {
	return nil
}

func (mc *MockConfig) Load() error {
	return nil
}

func (mc *MockConfig) Save() error {
	return nil
}

func (mc *MockConfig) GetConfigFile() string {
	return "/home/fakeuser/.mycscloud/config.yml"
}

func (mc *MockConfig) GetConfigAsOf() int64 {
	return 0
}

func (mc *MockConfig) SetConfigAsOf(asOf int64) {
}

func (mc *MockConfig) Initialized() bool {
	return true
}

func (mc *MockConfig) SetInitialized() {
}

func (mc *MockConfig) EULAAccepted() bool {
	return true
}

func (mc *MockConfig) SetEULAAccepted() {
}

func (mc *MockConfig) HasPassphrase() bool {
	return false
}

func (mc *MockConfig) GetPassphrase() string {
	return ""
}

func (mc *MockConfig) SetPassphrase(passphrase string) {
}

func (mc *MockConfig) GetKeyTimeout() time.Duration {
	return 0
}

func (mc *MockConfig) SetKeyTimeout(timeout time.Duration) {
}

func (mc *MockConfig) AuthContext() config.AuthContext {
	return mc.authContext
}

func (mc *MockConfig) DeviceContext() config.DeviceContext {
	return mc.deviceContext
}

func (mc *MockConfig) TargetContext() config.TargetContext {
	return mc.targetContext
}

func (mc *MockConfig) SetLoggedInUser(userID, userName string) error {
	if mc.deviceContext != nil {
		mc.deviceContext.SetLoggedInUser(userID, userName)
	}
	return nil
}

func (mc *MockConfig) ContextVars() map[string]string {

	contextVars := make(map[string]string)

	keyID, keyData := mc.authContext.GetPublicKey()
	contextVars["mycs_cloud_public_key_id"] = keyID
	contextVars["mycs_cloud_public_key"] = keyData

	return contextVars
}