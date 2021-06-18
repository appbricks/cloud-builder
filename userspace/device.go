package userspace

type Device struct {
	DeviceID string `json:"deviceID,omitempty"`
	Name     string `json:"name,omitempty"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	Address string   `json:"address,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}
