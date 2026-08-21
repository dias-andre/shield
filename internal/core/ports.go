package core

import (
	"errors"
)

var (
	ErrVaultFileNotExists      = errors.New("vault file does not exist")
	ErrVaultFileCorrupted      = errors.New("vault file corrupted or empty")
	ErrInvalidMagic            = errors.New("invalid vault magic")
	ErrInvalidVaultPermissions = errors.New("invalid vault permissions")
)

type EncryptorPort interface {
	Encrypt(vault []byte, key []byte) ([]byte, error)
	Decrypt(data []byte, key []byte) ([]byte, error)
	GetMinimumVaultSize() int64
}

type KeySystemPort interface {
	GetKey() ([]byte, error)
	SaveKey([]byte) error
}

type Lockable interface {
	Lock() error
}

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

type BackupPort interface {
	CreateBackup(*Vault, []byte) error
}
