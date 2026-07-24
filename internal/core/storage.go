package core

import "errors"

var (
	ErrVaultFileNotExists = errors.New("vault file not exists")
	ErrVaultFileCorrupted = errors.New("vault file corrupted or empty")
	ErrInvalidMagic       = errors.New("invalid vault magic")
)

type StoragePort interface {
	Save(v []byte) error
	Load() ([]byte, error)
	VaultExists() (bool, error)
}

type SupportRawVault interface {
	LoadRawVault() (*RawVault, error)
	SaveRawVault(*RawVault) error
}
