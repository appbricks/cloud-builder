package config

import (
	"encoding/json"
	"io"

	"golang.org/x/oauth2"
)

type authContext struct {
	token *oauth2.Token
}

func NewAuthContext() *authContext {
	return &authContext{}
}

func (ac *authContext) Reset() error {
	ac.token = &oauth2.Token{}
	return nil
}

func (ac *authContext) Load(input io.Reader) error {
	decoder := json.NewDecoder(input)
	ac.token = &oauth2.Token{}
	return decoder.Decode(ac.token)
}

func (ac *authContext) Save(output io.Writer) error {
	encoder := json.NewEncoder(output)
	return encoder.Encode(ac.token)
}

func (ac *authContext) SetToken(token *oauth2.Token) {
	ac.token = token
}

func (ac *authContext) GetToken() *oauth2.Token {
	return ac.token
}
