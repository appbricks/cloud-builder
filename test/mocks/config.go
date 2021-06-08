package mocks

import (
	"time"

	"github.com/appbricks/cloud-builder/config"
)

type MockConfig struct {
	authContext config.AuthContext
	deviceContext config.DeviceContext
	context config.Context
}

func NewMockConfig(
	authContext config.AuthContext,
	deviceContext config.DeviceContext,
	context config.Context,
) config.Config {

	return &MockConfig{
		authContext: authContext,
		deviceContext: deviceContext,
		context: context,
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

func (mc *MockConfig) SetPassphrase(passphrase string) {
}

func (mc *MockConfig) SetKeyTimeout(timeout time.Duration) {
}

func (mc *MockConfig) AuthContext() config.AuthContext {
	return mc.authContext
}

func (mc *MockConfig) DeviceContext() config.DeviceContext {
	return mc.deviceContext
}

func (mc *MockConfig) Context() config.Context {
	return mc.context
}
