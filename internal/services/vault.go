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
var ErrSshEntryNotFound = errors.New("Ssh entry not found!")

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

func (s *VaultService) DeleteSshEntry(entryName string, key []byte) error {
	vault, err := s.GetVault(key)
	if err != nil { return err }

	if entry, ok := vault.Entries[entryName]; ok {
		delete(vault.Entries, entry.Name)
		err := s.SaveVault(vault, key)
		if err != nil { return err }
		return nil
	}

	return ErrSshEntryNotFound
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

func (s *VaultService) VaultExists() (bool, error) {
	return s.storage.VaultExists()
}

func NewVaultService(encryptor ports.EncryptorPort, storage ports.StoragePort) VaultService {
	return VaultService{
		storage: storage,
		crypto: encryptor,
	}
}
