package config

import (
	"encoding/json"
	"io"

	"github.com/appbricks/cloud-builder/user"
)

// global configuration context
type deviceContext struct {
	// the primary user owns or has super 
	// user access to this client/device and
	// can access additional capabilities 
	// such as space management
	Primary *user.User `json:"primary,omitempty"`

	// device identifier for this client 
	// associated with the primary user
	DeviceID string `json:"deviceID,omitempty"`

	// guest users of this client
	Guests map[string]*user.User
}

// in: cookbook - the cookbook in context
func NewDeviceContext() *deviceContext {
	return &deviceContext{
		Guests: make(map[string]*user.User),
	}
}

func (ctx *deviceContext) Reset() error {
	ctx.Primary = nil
	ctx.DeviceID = ""
	ctx.Guests = make(map[string]*user.User)	
	return nil
}

func (ctx *deviceContext) Load(input io.Reader) error {
	decoder := json.NewDecoder(input)
	return decoder.Decode(ctx)
}

func (ctx *deviceContext) Save(output io.Writer) error {
	encoder := json.NewEncoder(output)
	return encoder.Encode(ctx)
}

func (ctx *deviceContext) SetPrimaryUser(name string) {
	ctx.Primary = &user.User{
		Name: name,
	}
}

func (ctx *deviceContext) GetPrimaryUser() (string, bool) {
	if ctx.Primary == nil {
		return "", false
	}
	return ctx.Primary.Name, true
}
