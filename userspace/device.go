package userspace

import "github.com/mevansam/goutils/crypto"

type Device struct {
	DeviceID string `json:"deviceID,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`

	Enabled bool `json:"enabled"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	DeviceUsers []*User `json:"-"`
}

func (d *Device) UpdateKeys() error {

	var (
		err error
	)

	// create new device key pair
	d.RSAPrivateKey, d.RSAPublicKey, err = crypto.CreateRSAKeyPair(nil)
	return err
}
