package crypto

import (
	"encoding/base64"
	"crypto/rand"
	"os"

	"github.com/fatih/color"
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "shield-cli"
	keyName = "master-key"
)

func GenerateAndStoreMasterKey() (string, error) {
	if _, err := GetMasterKey(); err == nil {
		color.HiRed("A master key already exists!")
		os.Exit(0)
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
