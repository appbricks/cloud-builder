package userspace

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	WGPrivateKey string `json:"wgPrivateKey,omitempty"`
	WGPublickKey string `json:"wgPublickKey,omitempty"`

	// active devices for this users
	Devices []*Device `json:"devices,omitempty"`

	// indicates if user is active 
	// for the device in context
	Active bool `json:"active"`
}
