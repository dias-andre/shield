package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "shield-cli"
	keyName = "master-key"
)

var ErrMasterKeyAlreadyExists = errors.New("A master key already exists!")

func GenerateAndStoreMasterKey() (string, error) {
	if _, err := GetMasterKey(); err == nil {
		return "", ErrMasterKeyAlreadyExists
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", err
	}

	masterKey := base64.StdEncoding.EncodeToString(keyBytes)

	err := keyring.Set(serviceName, keyName, masterKey)

	if err != nil {
		return "", err
	}

	return masterKey, nil
}

func GetMasterKey() (string, error) {
	key, err := keyring.Get(serviceName, keyName)
	if err != nil {
		return "", err
	}
	return key, nil
}
