package adapters

import (
	"errors"

	"github.com/99designs/keyring"
	"github.com/dias-andre/shield/internal/core"
)

type PartedKeyring struct {
	ring keyring.Keyring
}

func NewPartedKeyring() (core.KeySystemPort, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:             "shield-cli",
		AllowedBackends:         []keyring.BackendType{keyring.SecretServiceBackend},
		LibSecretCollectionName: "login",
	})
	if err != nil {
		return nil, err
	}
	return &PartedKeyring{
		ring: ring,
	}, nil
}

func (s *PartedKeyring) GetKey() ([]byte, error) {
	key, err := s.ring.Get("key-a")
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return key.Data, nil
}

func (s *PartedKeyring) SaveKey(key []byte) error {
	if err := s.ring.Set(keyring.Item{
		Key:  "key-a",
		Data: key,
	}); err != nil {
		return err
	}
	return nil
}
