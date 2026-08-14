package core

import "errors"

var (
	ErrVaultFileNotExists      = errors.New("vault file does not exist")
	ErrVaultFileCorrupted      = errors.New("vault file corrupted or empty")
	ErrInvalidMagic            = errors.New("invalid vault magic")
	ErrInvalidVaultPermissions = errors.New("invalid vault permissions")
)

type StoragePort interface {
	Save(v []byte) error
	Load() ([]byte, error)
	VaultExists() (bool, error)
	ValidateVault() error
	GetVaultSize() int64
}

type SupportRawVault interface {
	LoadRawVault() (*RawVault, error)
	SaveRawVault(*RawVault) error
}
