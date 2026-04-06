package services

import (
	"encoding/json"
	"errors"

	"github.com/dias-andre/shield/internal/core/domain"
	"github.com/dias-andre/shield/internal/core/ports"
)

type VaultService struct {
	storage ports.StoragePort
	crypto ports.EncryptorPort
}

var ErrVaultFileNotExists = errors.New("Vault file not exists!")

func (s *VaultService) GetVault(key []byte) (domain.Vault, error) {
	var v domain.Vault

	encryptedContent, err := s.storage.Load()
	if err != nil {
		return v, err
	}

	plaintext, err := s.crypto.Decrypt(encryptedContent, key)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(plaintext, &v)
	if err != nil { return v, err }
	return v, nil
}

func (s *VaultService) AddSshEntry(entry domain.SSHEntry, key []byte) error {

	v, err := s.GetVault(key)
	if err != nil {
		if errors.Is(err, ErrVaultFileNotExists) {
			v = domain.NewVault()
		} else {
			return err
		}
	}

	v.Entries[entry.Name] = entry

	jsonData, err := json.Marshal(v)
	if err != nil { return err }

	vaultEncrypted, err := s.crypto.Encrypt(jsonData, key)
	if err != nil { return err }
	return s.storage.Save(vaultEncrypted)
}

func (s *VaultService) InitVault() domain.Vault {
	return domain.NewVault()
}

func (s *VaultService) SaveVault(vault domain.Vault, key []byte) error {
	jsonData, err := json.Marshal(vault)
	if err != nil { return nil }

	encryptedVault, err := s.crypto.Encrypt(jsonData, key)
	if err != nil { return err }

	return s.storage.Save(encryptedVault)
}

func NewVaultService(encryptor ports.EncryptorPort, storage ports.StoragePort) VaultService {
	return VaultService{
		storage: storage,
		crypto: encryptor,
	}
}