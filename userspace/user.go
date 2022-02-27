package userspace

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mevansam/goutils/crypto"
	"github.com/mevansam/goutils/logger"
)

type User struct {
	UserID string `json:"userID"`
	Name   string `json:"name"`

	KeyTimestamp  int64  `json:"keyTimestamp,omitempty"`
	RSAPrivateKey string `json:"rsaPrivateKey,omitempty"`
	RSAPublicKey  string `json:"rsaPublicKey,omitempty"`

	key *crypto.RSAKey `json:"-"`

	Certificate string `json:"certificate,omitempty"`

	// indicates if user is active 
	// for the device in context
	Active bool `json:"active"`

	FirstName       string `json:"-"`
	MiddleName      string `json:"-"`
	FamilyName      string `json:"-"`
	PreferredName   string `json:"-"`
}

func (u *User) SetKey(key *crypto.RSAKey) error {

	var (
		err error

		publicKey *crypto.RSAPublicKey

		dataToVerify,
		verifyString []byte
	)

	if len(u.RSAPublicKey) > 0 {
		// validate known public key with provided private 
		// key by encrypting some data with the known public 
		// key and decrypting with the provided private key
		if publicKey, err = crypto.NewPublicKeyFromPEM(u.RSAPublicKey); err != nil {
			return err
		}
		if dataToVerify, err = publicKey.Encrypt([]byte(u.UserID)); err != nil {
			return err
		}
		if verifyString, err = key.Decrypt(dataToVerify); err != nil {
			logger.ErrorMessage(
				"User.SetKey(): Validation of given RSA key with user's known public key failed: %s", 
				err.Error(),
			)
			return err
		}
		if string(verifyString) != u.UserID {
			logger.ErrorMessage(
				"User.SetKey(): Decrypted value was '%s' not expected '%s': %s", 
				string(verifyString), u.UserID, err.Error(),
			)
			return fmt.Errorf("decryption succeeded but verification failed")
		}
	}

	// timestamp in milliseconds
	u.KeyTimestamp = time.Now().UnixMilli()

	if u.RSAPrivateKey, err = key.GetPrivateKeyPEM(); err == nil {
		u.RSAPublicKey, err = key.GetPublicKeyPEM()
	}
	u.key = key
	return err
}

func (u *User) EncryptConfig(configData []byte) (string, error) {

	var (
		err error	

		cipherData []byte
	)
	
	if u.key == nil {
		if u.key, err = crypto.NewRSAKeyFromPEM(u.RSAPrivateKey, nil); err != nil {
			return "", err
		}
	}
	if cipherData, err = u.key.EncryptPack(configData); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(cipherData), nil
}

func (u *User) DecryptConfig(configData string) ([]byte, error) {

	var (
		err error

		decodedData,
		plainData []byte
	)
	
	if u.key == nil {
		if u.key, err = crypto.NewRSAKeyFromPEM(u.RSAPrivateKey, nil); err != nil {
			return nil, err
		}
	}
	if decodedData, err = base64.StdEncoding.DecodeString(configData); err != nil {
		return nil, nil
	}
	if plainData, err = u.key.DecryptUnpack(decodedData); err != nil {
		return nil, err
	}
	
	return plainData, nil
}
