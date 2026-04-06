package adapters

import (
	"errors"
	"os"
)

var ErrVaultFileNotExists = errors.New("Vault file not exists!")

type FileStorage struct {
	vaultPath string
}

func(s *FileStorage) Save(v []byte) error {
	return os.WriteFile(s.vaultPath, v, 0600)
}

func(s *FileStorage) Load() ([]byte, error) {
	v, err := os.ReadFile(s.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return v, ErrVaultFileNotExists
		}
		return v, err
	}
	return v, nil
}