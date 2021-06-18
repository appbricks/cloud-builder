package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"strings"
)

// creates a private key, public key pair
func CreateRSAKeyPair() (string, string, error) {
	
	var (
		err error
		key *rsa.PrivateKey

		privateKey, publicKey []byte
		privateKeyPEM, publicKeyPEM strings.Builder
	)

	// create rsa key pair
	if key, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
		return "", "", err
	}
	// pem encoded private key
	if privateKey, err = x509.MarshalPKCS8PrivateKey(key); err  != nil {
		return "", "", err
	}
	if err := pem.Encode(
		&privateKeyPEM, 
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKey,
		},
	); err != nil {
		return "", "", err
	}
	// pem encoded public key
	if publicKey, err = asn1.Marshal(key.PublicKey); err != nil {
		return "", "", err
	}
	if err := pem.Encode(
		&publicKeyPEM, 
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKey,
		},
	); err != nil {
		return "", "", err
	}

	return privateKeyPEM.String(), publicKeyPEM.String(), err
}