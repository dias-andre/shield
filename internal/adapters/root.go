package adapters

import "github.com/dias-andre/shield/internal/core/ports"

const (
	shieldServiceName = "shield-cli"
	shieldKeyName = "master-key"
)

func NewFileSystemStorage(path string) ports.StoragePort {
	var storage FileStorage
	storage.vaultPath = path
	return &storage
}

func NewAESEncryptor() ports.EncryptorPort {
	var encryptor AES_Encrypter
	return &encryptor
}

// func NewKeyringSystem() ports.KeySystemPort {
// 	return &KeyringSystem{
// 		serviceName: "shield-cli",
// 		keyName: "master-key",
// 	}
// }