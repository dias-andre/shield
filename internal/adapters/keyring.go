package adapters

import (
	"errors"

	"github.com/99designs/keyring"
	"github.com/dias-andre/shield/internal/core"
)

type KeyringAdapter struct {
	ring keyring.Keyring
}

func NewKeyringAdapter() (core.KeySystemPort, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "shield-cli",
	})
	if err != nil {
		return nil, err
	}

	return &KeyringAdapter{
		ring: ring,
	}, nil
}

func (k *KeyringAdapter) GetKey() ([]byte, error) {
	data, err := k.ring.Get("master-key")
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return data.Data, nil
}

func (k *KeyringAdapter) SaveKey(key []byte) error {
	if err := k.ring.Set(keyring.Item{
		Key:  "master-key",
		Data: key,
	}); err != nil {
		return err
	}

	return nil
}
