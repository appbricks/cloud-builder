package userspace

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	// indicates if user is active 
	// for the device in context
	Active bool `json:"active"`
}
