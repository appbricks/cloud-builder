package userspace

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	WGPrivateKey string `json:"wgPrivateKey,omitempty"`
	WGPublickKey string `json:"wgPublickKey,omitempty"`

	// indicates if user is active 
	// for the device in context
	Active bool `json:"active"`
}

type SpaceUser struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	IsOwner    bool   `json:"isOwner"`
	AccessType string `json:"accessType"`

	// active devices for this users
	Devices []*Device `json:"devices,omitempty"`
}
