package config

import (
	"io"
	"time"

	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/provider"
)

// provides an interface for managing application configuration
type Config interface {
	Reset() error

	Load() error
	Save() error

	EULAAccepted() bool
	SetEULAAccepted()

	Initialized() bool
	SetInitialized()

	HasPassphrase() bool
	SetPassphrase(passphrase string)

	SetKeyTimeout(timeout time.Duration)
	AuthContext() AuthContext
	DeviceContext() DeviceContext
	Context() Context
}

// provides an interface for saving and retrieving an oauth token
type AuthContext interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	SetToken(token *oauth2.Token)
	GetToken() *oauth2.Token
}

// provides an interface for saving and retrieving an oauth token
type DeviceContext interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	SetPrimaryUser(name string)
	GetPrimaryUser() (string, bool)
}

// provides an interface for managing the configuration context
type Context interface {
	Reset() error

	Load(input io.Reader) error
	Save(output io.Writer) error

	Cookbook() *cookbook.Cookbook
	GetCookbookRecipe(recipe, iaas string) (cookbook.Recipe, error)
	SaveCookbookRecipe(recipe cookbook.Recipe)

	CloudProviderTemplates() []provider.CloudProvider
	GetCloudProvider(iaas string) (provider.CloudProvider, error)
	SaveCloudProvider(provider provider.CloudProvider)

	NewTarget(recipeName, recipeIaas string) (*target.Target, error)
	TargetSet() *target.TargetSet
	HasTarget(name string) bool
	GetTarget(name string) (*target.Target, error)
	SaveTarget(key string, target *target.Target)
}
