package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/goutils/logger"
)

// global configuration context
type deviceContext struct {
	// device authentication key
	DeviceIDKey string `json:"deviceIDKey,omitempty"`

	// registered device information
	Device *userspace.Device `json:"device,omitempty"`

	// managed devices - this client will always be 
	// associated with one device which is the registered 
	// primary device. however, the owner user can add 
	// and manage additional secondary devices which can
	// connect via a device's native vpn client with limited
	// functionality.
	ManagedDevices []*userspace.Device `json:"managedDevices,omitempty"`

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
	if dc.Owner != nil && dc.userID != dc.Owner.UserID {
		// ensure owner's private key is rest 
		// if logged in user is different
		dc.Owner.RSAPrivateKey = ""
	}

	encoder := json.NewEncoder(output)
	return encoder.Encode(dc)
}

func (dc *deviceContext) NewDevice() (*userspace.Device, error) {

	var (
		err error

		device *userspace.Device
	)
	
	device = &userspace.Device{}
	if device.Name, err = os.Hostname(); err != nil {
		logger.ErrorMessage("Unable to determine hostname for default device name: %s", err.Error())
	}
	if err = device.UpdateKeys(); err != nil {
		return nil, err
	}
	dc.Device = device
	return dc.Device, err
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

func (dc *deviceContext) NewManagedDevice() (*userspace.Device, error) {

	var (
		err error
	)

	device := &userspace.Device{}
	if err = device.UpdateKeys(); err != nil {
		return nil, err
	}

	dc.ManagedDevices = append(dc.ManagedDevices, device)
	return device, nil
}

func (dc *deviceContext) GetManagedDevice(deviceName string) *userspace.Device {
	for _, device := range dc.ManagedDevices {
		if device.Name == deviceName {
			return device
		}
	}
	return nil
}

func (dc *deviceContext) GetManagedDevices() []*userspace.Device {
	return dc.ManagedDevices
}

func (dc *deviceContext) DeleteManageDevice(deviceID string) {
	for i, device := range dc.ManagedDevices {
		if device.DeviceID == deviceID {
			dc.ManagedDevices = append(dc.ManagedDevices[:i], dc.ManagedDevices[i+1:]...)
			return
		}
	}
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

func (dc *deviceContext) GetGuestUsers() []*userspace.User {

	// create list of ordered users
	users := make([]*userspace.User, len(dc.Users))

	OUTER:
	for _, user := range dc.Users {		
		i := 0
		for _, u := range users {
			if u != nil {
				if strings.Compare(u.Name, user.Name) == 1 {
					copy(users[i+1:], users[i:])
					users[i] = user
					continue OUTER
				}
			} else {
				break
			}
			i++
		}
		users[i] = user
	}
	return users
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

	return &userspace.User{
		UserID: userID,
		Name: name,
	}, nil
}