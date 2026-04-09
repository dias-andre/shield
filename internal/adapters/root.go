// Copyright (C) 2026 André de Oliveira Dias (diaso.andre@outlook.com)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

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