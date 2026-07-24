// Package services which manages all ports and adapters to work with shield Vault
package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dias-andre/shield/internal/core"
)

type VaultService struct {
	storage core.StoragePort
	crypto  core.EncryptorPort
}

var (
	ErrVaultFileNotExists = errors.New("vault file not found")
	ErrSSHEntryNotFound   = errors.New("ssh entry not found")
	ErrDecryptionFailed   = errors.New("vault decription error")
	ErrBrokenVault        = errors.New("vault file is broken")
)

const (
	ShieldMajor byte = 0
	ShieldMinor byte = 1
	ShieldPatch byte = 0
)

func (s *VaultService) GetVaultOld(key []byte) (*core.Vault, error) {
	var vault core.Vault

	encryptedContent, err := s.storage.Load()
	if err != nil {
		return nil, err
	}

	plaintext, err := s.crypto.Decrypt(encryptedContent, key)
	if err != nil {
		return nil, errors.Join(ErrDecryptionFailed, err)
	}
	err = json.Unmarshal(plaintext, &vault)
	if err != nil {
		return nil, err
	}
	return &vault, nil
}

func (s *VaultService) GetVault(key []byte) (*core.Vault, error) {
	rawVaultStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return nil, fmt.Errorf("invalid storage system")
	}
	rawVault, err := rawVaultStorage.LoadRawVault()
	if err != nil {
		if errors.Is(err, core.ErrInvalidMagic) {
			// try with the old method
			result, err := s.storage.Load()
			if err != nil {
				return nil, errors.Join(ErrBrokenVault, core.ErrInvalidMagic, err)
			}
			fmt.Println("Deprecated vault format detected. Shield will attempt to upgrade it.")
			rawVault, err = s.ParseBytesToRawVault(result)
			if err != nil {
				return nil, fmt.Errorf("failed to upgrade vault format: %v", err)
			}
			err = rawVaultStorage.SaveRawVault(rawVault)
			if err != nil {
				return nil, fmt.Errorf("failed to upgrade vault format: %v", err)
			}
			fmt.Println("Vault successfully upgraded to the latest version.")
		} else {
			return nil, err
		}
	}
	encryptedSize := 12 + len(rawVault.Ciphertext)
	encryptedVault := make([]byte, 0, encryptedSize)
	encryptedVault = append(encryptedVault, rawVault.Nonce[:]...)
	encryptedVault = append(encryptedVault, rawVault.Ciphertext...)
	plaintext, err := s.crypto.Decrypt(encryptedVault, key)
	if err != nil {
		return nil, errors.Join(ErrDecryptionFailed, err)
	}

	var vault core.Vault
	err = json.Unmarshal(plaintext, &vault)
	if err != nil {
		return nil, err
	}
	return &vault, nil
}

func (s *VaultService) ParseBytesToRawVault(vaultInBytes []byte) (*core.RawVault, error) {
	realContentSize := len(vaultInBytes) - 12
	if realContentSize <= 0 {
		return nil, ErrBrokenVault
	}
	var raw core.RawVault
	raw.Version = core.SemVer{
		Major: ShieldMajor,
		Minor: ShieldMinor,
		Patch: ShieldPatch,
	}
	copy(raw.Nonce[:], vaultInBytes[:12])
	raw.Ciphertext = make([]byte, realContentSize)
	copy(raw.Ciphertext, vaultInBytes[12:])
	return &raw, nil
}

func (s *VaultService) AddSSHEntry(entry core.SSHEntry, key []byte) error {
	v, err := s.GetVault(key)
	if err != nil {
		if errors.Is(err, ErrVaultFileNotExists) {
			newVault := core.NewVault()
			v = &newVault
		} else {
			return err
		}
	}
	defer v.Erase()

	v.Entries[entry.Name] = entry

	jsonData, err := json.Marshal(v)
	if err != nil {
		return err
	}

	vaultEncrypted, err := s.crypto.Encrypt(jsonData, key)
	if err != nil {
		return err
	}
	return s.storage.Save(vaultEncrypted)
}

func (s *VaultService) DeleteSSHEntry(entryName string, key []byte) error {
	vault, err := s.GetVault(key)
	if err != nil {
		return err
	}
	defer vault.Erase()

	if entry, ok := vault.Entries[entryName]; ok {
		delete(vault.Entries, entry.Name)
		err := s.SaveVault(vault, key)
		if err != nil {
			return err
		}
		return nil
	}

	return ErrSSHEntryNotFound
}

func (s *VaultService) InitVault() core.Vault {
	return core.NewVault()
}

func (s *VaultService) SaveVault(vault *core.Vault, key []byte) error {
	jsonData, err := json.Marshal(vault)
	if err != nil {
		return nil
	}
	encryptedVault, err := s.crypto.Encrypt(jsonData, key)
	if err != nil {
		return err
	}
	rawVault, err := s.ParseBytesToRawVault(encryptedVault)
	if err != nil {
		return err
	}
	rawVaultStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return errors.New("invalid storage system")
	}
	return rawVaultStorage.SaveRawVault(rawVault)
}

func (s *VaultService) VaultExists() (bool, error) {
	return s.storage.VaultExists()
}

func NewVaultService(encryptor core.EncryptorPort, storage core.StoragePort) VaultService {
	return VaultService{
		storage: storage,
		crypto:  encryptor,
	}
}
