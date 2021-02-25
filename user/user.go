package user

type User struct {
	Name string `json:"name"`

	RSAPublicKey string `json:"rsaPublicKey,omitempty"`
	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`

	WGPublickKey string `json:"wgPublickKey,omitempty"`
	WGPrivateKey string `json:"wgPrivateKey,omitempty"`
}
