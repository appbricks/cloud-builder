package config

import (
	"io"
	"time"

	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/gocloud/provider"
)

// provides an interface for managing application configuration
type Config interface {
	Reset() error

	Load() error
	Save() error

	GetConfigFile() string

	GetConfigAsOf() int64
	SetConfigAsOf(asOf int64)

	EULAAccepted() bool
	SetEULAAccepted()

	Initialized() bool
	SetInitialized()

	HasPassphrase() bool
	GetPassphrase() string
	SetPassphrase(passphrase string)

	GetKeyTimeout() time.Duration
	SetKeyTimeout(timeout time.Duration)
	AuthContext() AuthContext
	DeviceContext() DeviceContext
	TargetContext() TargetContext

	SetLoggedInUser(userID, userName string) error

	ContextVars() map[string]string
}

// provides an interface for saving and retrieving an oauth token
type AuthContext interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	SetToken(token *oauth2.Token)
	GetToken() *oauth2.Token

	SetPublicKey(keyID, keyData string)
	GetPublicKey() (string, string)

	IsLoggedIn() bool
}

// provides an interface for saving and retrieving an oauth token
type DeviceContext interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	NewDevice() (*userspace.Device, error)
	SetDeviceID(deviceIDKey, deviceID, name string) *userspace.Device
	GetDevice() *userspace.Device
	GetDeviceIDKey() string
	GetDeviceID() (string, bool)
	GetDeviceName() (string, bool)

	NewManagedDevice() (*userspace.Device, error)
	GetManagedDevice(deviceName string) *userspace.Device
	GetManagedDevices() []*userspace.Device
	DeleteManageDevice(deviceID string)

	NewOwnerUser(userID, name string) (*userspace.User, error)
	GetOwner() *userspace.User
	GetOwnerUserID() (string, bool)
	GetOwnerUserName() (string, bool)
	IsAuthorizedUser(name string) bool

	NewGuestUser(userID, name string) (*userspace.User, error)
	AddGuestUser(user *userspace.User)
	GetGuestUsers() []*userspace.User
	GetGuestUser(name string) (*userspace.User, bool)
	ResetGuestUsers() map[string]*userspace.User

	SetLoggedInUser(userID, userName string)
	GetLoggedInUserID() string
	GetLoggedInUserName() string
	GetLoggedInUser() (*userspace.User, error)
}

// provides an interface for managing the configuration context
type TargetContext interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	Cookbook() *cookbook.Cookbook
	GetCookbookRecipe(recipe, iaas string) (cookbook.Recipe, error)
	SaveCookbookRecipe(recipe cookbook.Recipe)

	CloudProviderTemplates() []provider.CloudProvider
	GetCloudProvider(iaas string) (provider.CloudProvider, error)
	SaveCloudProvider(provider provider.CloudProvider)

	NewTarget(recipeKey, recipeIaas string) (*target.Target, error)
	TargetSet() *target.TargetSet
	HasTarget(name string) bool
	GetTarget(name string) (*target.Target, error)
	SaveTarget(key string, target *target.Target)
	DeleteTarget(key string)

	IsDirty() bool
}
