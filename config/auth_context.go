package config

import (
	"encoding/json"
	"io"

	"golang.org/x/oauth2"
)

type authContext struct {
	Token *oauth2.Token `json:"token,omitempty"`
	
	// additional token data returned by derivative 
	// oauth flows (i.e. cognito id_token)
	Extra map[string]interface{} `json:"tokenExtra,omitempty"`

	MyCSPublicKeyID string `json:"mycsPublicKeyID,omitempty"`
	MyCSPublicKey   string `json:"mycsPublicKey,omitempty"`
}

func NewAuthContext() *authContext {
	return &authContext{
		Token: &oauth2.Token{},
		Extra: make(map[string]interface{}),
	}
}

func (ac *authContext) Reset() error {
	ac.Token = &oauth2.Token{}
	ac.Extra = make(map[string]interface{})
	return nil
}

func (ac *authContext) Load(input io.Reader) error {
	decoder := json.NewDecoder(input)
	if err := decoder.Decode(ac); err != nil {
		return err
	}

	// add saved extra token data to token
	ac.Token = ac.Token.WithExtra(ac.Extra)
	return nil
}

func (ac *authContext) Save(output io.Writer) error {
	encoder := json.NewEncoder(output)
	// extract extra fields to persist
	ac.Extra["id_token"] = ac.Token.Extra("id_token")
	return encoder.Encode(ac)
}

func (ac *authContext) SetToken(token *oauth2.Token) {
	ac.Token = token
}

func (ac *authContext) GetToken() *oauth2.Token {
	return ac.Token
}

func (ac *authContext) SetPublicKey(keyID, key string) {
	ac.MyCSPublicKeyID = keyID
	ac.MyCSPublicKey = key
}

func (ac *authContext) GetPublicKey() (string, string) {
	return ac.MyCSPublicKeyID, ac.MyCSPublicKey
}

func (ac *authContext) IsLoggedIn() bool {
	return ac.Token != nil && ac.Token.Valid()
}
