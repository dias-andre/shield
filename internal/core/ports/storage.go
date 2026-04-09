package ports

type StoragePort interface {
	Save(v []byte) error
	Load() ([]byte, error)
	VaultExists() (bool, error)
}