package adapters

import (
	"encoding/base64"

	"github.com/zalando/go-keyring"
)

type KeyringSystem struct {
	serviceName string
	keyName string
}

func (s *KeyringSystem) SaveKey(key []byte) error {
	stringKey := base64.StdEncoding.EncodeToString(key)
	
	return keyring.Set(s.serviceName, s.keyName, stringKey)
}

func (s *KeyringSystem) GetKey() ([]byte, error) {
	key, err := keyring.Get(s.serviceName, s.keyName)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(key)
}