// Copyright (C) 2026 André de Oliveira Dias (diaso.andre@outlook.com)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package adapters is the core of shield adapters
package adapters

import "github.com/dias-andre/shield/internal/core"

const (
	shieldServiceName = "shield-cli"
	shieldKeyName     = "master-key"
)

func NewFileSystemStorage(path string) core.StoragePort {
	var storage FileStorage
	storage.vaultPath = path
	return &storage
}

func NewAESEncryptor() core.EncryptorPort {
	var encryptor AESEncrypter
	return &encryptor
}

// func NewKeyringSystem() ports.KeySystemPort {
// 	return &KeyringSystem{
// 		serviceName: "shield-cli",
// 		keyName: "master-key",
// 	}
// }
