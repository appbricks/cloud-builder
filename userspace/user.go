package userspace

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	WGPrivateKey string `json:"wgPrivateKey,omitempty"`
	WGPublickKey string `json:"wgPublickKey,omitempty"`

	Active bool `json:"active"`
}
