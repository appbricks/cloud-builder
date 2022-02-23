package userspace

import "time"

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	KeyTimestamp  time.Duration `json:"keyTimestamp,omitempty"`
	RSAPrivateKey string        `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string        `json:"rsaPublicKey,omitempty"`

	// indicates if user is active 
	// for the device in context
	Active bool `json:"active"`
}
