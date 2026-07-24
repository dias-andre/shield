package adapters

import (
	"bytes"
	"errors"
	"os"

	"github.com/dias-andre/shield/internal/core"
)

var MagicBytes = []byte("SHLD")

type FileStorage struct {
	vaultPath string
}

func (s *FileStorage) Save(v []byte) error {
	return os.WriteFile(s.vaultPath, v, 0o600)
}

func (s *FileStorage) Load() ([]byte, error) {
	v, err := os.ReadFile(s.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return v, core.ErrVaultFileNotExists
		}
		return v, err
	}
	return v, nil
}

func (s *FileStorage) VaultExists() (bool, error) {
	_, err := os.Stat(s.vaultPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (s *FileStorage) LoadRawVault() (*core.RawVault, error) {
	data, err := os.ReadFile(s.vaultPath)
	if err != nil && os.IsNotExist(err) {
		return nil, core.ErrVaultFileNotExists
	}

	if len(data) < 19 {
		return nil, core.ErrVaultFileCorrupted
	}

	if !bytes.Equal(data[:4], MagicBytes) {
		return nil, core.ErrInvalidMagic
	}

	major := data[4]
	minor := data[5]
	patch := data[6]

	var nonce [12]byte
	copy(nonce[:], data[7:19])

	rawVault := core.RawVault{
		Version: core.SemVer{
			Major: major,
			Minor: minor,
			Patch: patch,
		},
		Nonce:      nonce,
		Ciphertext: data[19:],
	}

	return &rawVault, nil
}

func (s *FileStorage) SaveRawVault(vault *core.RawVault) error {
	totalSize := 4 + 3 + 12 + len(vault.Ciphertext)
	vaultData := make([]byte, 0, totalSize)
	vaultData = append(vaultData, MagicBytes...)
	vaultData = append(vaultData, vault.Version.Major, vault.Version.Minor, vault.Version.Patch)
	vaultData = append(vaultData, vault.Nonce[:]...)
	vaultData = append(vaultData, vault.Ciphertext...)

	err := os.WriteFile(s.vaultPath, vaultData, 0o600)
	if err != nil {
		return err
	}

	return nil
}
