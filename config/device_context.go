package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/goutils/crypto"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// global configuration context
type deviceContext struct {
	// device authentication key
	DeviceIDKey string `json:"deviceIDKey,omitempty"`

	// device key pair
	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	// registered device information
	Device *userspace.Device `json:"device,omitempty"`

	// the owner user owns or has super 
	// user access to this client/device and
	// can access additional capabilities 
	// such as space management
	Owner *userspace.User `json:"owner,omitempty"`

	// authorized guest users
	Users map[string]*userspace.User `json:"users,omitempty"`

	// logged in user
	userID   string
	userName string	
}

// in: cookbook - the cookbook in context
func NewDeviceContext() *deviceContext {
	return &deviceContext{
		Users: make(map[string]*userspace.User),
	}
}

func (dc *deviceContext) Reset() error {
	dc.DeviceIDKey = ""
	dc.RSAPrivateKey = ""
	dc.RSAPublicKey = ""
	dc.Device = nil
	dc.Owner = nil
	dc.Users = make(map[string]*userspace.User)
	return nil
}

func (dc *deviceContext) Load(input io.Reader) error {	
	decoder := json.NewDecoder(input)	
	return decoder.Decode(dc)
}

func (dc *deviceContext) Save(output io.Writer) error {
	encoder := json.NewEncoder(output)
	return encoder.Encode(dc)
}

func (dc *deviceContext) NewDevice() (*userspace.Device, error) {
	dc.Device = &userspace.Device{}
	return dc.UpdateDeviceKeys()
}

func (dc *deviceContext) UpdateDeviceKeys() (*userspace.Device, error) {

	var (
		err error
	)

	// create new device key pair
	if dc.Device.RSAPrivateKey, dc.Device.RSAPublicKey, err = crypto.CreateRSAKeyPair(nil); err != nil {
		return nil, err
	}	
	return dc.Device, nil
}

func (dc *deviceContext) SetDeviceID(deviceIDKey, deviceID, name string) *userspace.Device {
	dc.DeviceIDKey = deviceIDKey
	dc.Device.DeviceID = deviceID
	dc.Device.Name = name
	return dc.Device
}

func (dc *deviceContext) GetDevice() *userspace.Device {
	return dc.Device
}

func (dc *deviceContext) GetDeviceIDKey() string {
	return dc.DeviceIDKey
}

func (dc *deviceContext) GetDeviceID() (string, bool) {
	if dc.Device == nil {
		return "", false
	}
	return dc.Device.DeviceID, true
}

func (dc *deviceContext) GetDeviceName() (string, bool) {
	if dc.Device == nil {
		return "", false
	}
	return dc.Device.Name, true
}

func (dc *deviceContext) NewOwnerUser(userID, name string) (*userspace.User, error) {

	var (
		err error
	)

	dc.Owner, err = newUser(userID, name)
	dc.Owner.Active = true
	return dc.Owner, err
}

func (dc *deviceContext) GetOwner() *userspace.User {
	return dc.Owner
}

func (dc *deviceContext) GetOwnerUserID() (string, bool) {
	if dc.Owner == nil {
		return "", false
	}
	return dc.Owner.UserID, true
}

func (dc *deviceContext) GetOwnerUserName() (string, bool) {
	if dc.Owner == nil {
		return "", false
	}
	return dc.Owner.Name, true
}

func (dc *deviceContext) NewGuestUser(userID, name string) (*userspace.User, error) {

	var (
		err  error
		user *userspace.User
	)

	if user, err = newUser(userID, name); err != nil {
		return nil, err
	}
	user.Active = false
	dc.Users[name] = user
	return user, nil
}

func (dc *deviceContext) AddGuestUser(user *userspace.User) {
	dc.Users[user.Name] = user
}

func (dc *deviceContext) GetGuestUser(name string) (*userspace.User, bool) {
	user, exists := dc.Users[name]
	return user, exists
}

func (dc *deviceContext) ResetGuestUsers() map[string]*userspace.User {
	guests := dc.Users
	dc.Users = make(map[string]*userspace.User)
	return guests
}

func (dc *deviceContext) IsAuthorizedUser(name string) bool {

	if name == dc.Owner.Name {
		return true
	}
	user, exists := dc.Users[name]
	return exists && user.Active
}

func (dc *deviceContext) SetLoggedInUser(userID, userName string) {
	dc.userID = userID
	dc.userName = userName
}

func (dc *deviceContext) GetLoggedInUserID() string {
	return dc.userID
}

func (dc *deviceContext) GetLoggedInUserName() string {
	return dc.userName
}

func (dc *deviceContext) GetLoggedInUser() (*userspace.User, error) {

	var (
		exists bool
		user   *userspace.User
	)

	if dc.userID == dc.Owner.UserID {
		user = dc.Owner
	} else if user, exists = dc.Users[dc.userName]; !exists {
		return nil, fmt.Errorf("logged in user \"%s\" does not exist in device context", dc.userName)
	}
	return user, nil
}

func newUser(userID, name string) (*userspace.User, error) {

	var (
		err error

		key wgtypes.Key
	)

	if key, err = wgtypes.GeneratePrivateKey(); err != nil {
		return nil, err
	}
	return &userspace.User{
		UserID: userID,
		Name: name,
		WGPrivateKey: key.String(),
		WGPublickKey: key.PublicKey().String(),
	}, nil
}