// Package services orchestrates the ports and adapters that back the shield vault.
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dias-andre/shield/internal/core"
)

type VaultService struct {
	storage core.StoragePort
	crypto  core.EncryptorPort
}

var (
	ErrDecryptionFailed = errors.New("vault decryption error")
	ErrBrokenVault      = errors.New("vault file is broken")
)

const (
	ShieldMajor byte = 0
	ShieldMinor byte = 1
	ShieldPatch byte = 0
)

func (s *VaultService) GetVault(key []byte) (*core.Vault, error) {
	rawVaultStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return nil, fmt.Errorf("invalid storage system")
	}
	rawVault, err := rawVaultStorage.LoadRawVault()
	if err != nil {
		if errors.Is(err, core.ErrInvalidMagic) {
			result, err := s.storage.Load()
			if err != nil {
				return nil, errors.Join(ErrBrokenVault, core.ErrInvalidMagic, err)
			}
			slog.Info("deprecated vault format detected, upgrading")
			rawVault, err = s.ParseBytesToRawVault(result)
			if err != nil {
				return nil, fmt.Errorf("failed to upgrade vault format: %w", err)
			}
			if err := rawVaultStorage.SaveRawVault(rawVault); err != nil {
				return nil, fmt.Errorf("failed to upgrade vault format: %w", err)
			}
			slog.Info("vault upgraded to the latest format")
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

func (s *VaultService) InitVault() core.Vault {
	return core.NewVault()
}

func (s *VaultService) SaveVault(vault *core.Vault, key []byte) error {
	jsonData, err := json.Marshal(vault)
	if err != nil {
		return err
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

func (s *VaultService) CheckVaultHealth() (bool, error) {
	vaultValidation := s.storage.ValidateVault()
	if vaultValidation != nil {
		return false, vaultValidation
	}
	if vaultSize := s.storage.GetVaultSize(); vaultSize <= s.crypto.GetMinimumVaultSize() {
		return false, fmt.Errorf("invalid vault size")
	}
	return true, nil
}
