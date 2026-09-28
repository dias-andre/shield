package adapters

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	var salt [16]byte
	offset := 7
	if major == 0 && minor >= 3 {
		if len(data) < 35 {
			return nil, core.ErrVaultFileCorrupted
		}
		copy(salt[:], data[7:23])
		offset = 23
	}
	var nonce [12]byte
	copy(nonce[:], data[offset:offset+12])

	rawVault := core.RawVault{
		Version: core.SemVer{
			Major: major,
			Minor: minor,
			Patch: patch,
		},
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: data[offset+12:],
	}

	return &rawVault, nil
}

func (s *FileStorage) SaveRawVault(vault *core.RawVault) error {
	totalSize := 4 + 3 + 12 + len(vault.Ciphertext)
	if vault.Version.Major == 0 && vault.Version.Minor >= 3 {
		totalSize += len(vault.Salt)
	}
	vaultData := make([]byte, 0, totalSize)
	vaultData = append(vaultData, MagicBytes...)
	vaultData = append(vaultData, vault.Version.Major, vault.Version.Minor, vault.Version.Patch)
	if vault.Version.Major == 0 && vault.Version.Minor >= 3 {
		vaultData = append(vaultData, vault.Salt[:]...)
	}
	vaultData = append(vaultData, vault.Nonce[:]...)
	vaultData = append(vaultData, vault.Ciphertext...)

	return atomicWrite(s.vaultPath, vaultData, 0o600)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vault-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// BackupVault preserves the exact current file bytes before a format migration.
func (s *FileStorage) BackupVault() (string, error) {
	data, err := os.ReadFile(s.vaultPath)
	if err != nil {
		return "", err
	}
	backupPath := fmt.Sprintf("%s.pre-migration.%d", s.vaultPath, time.Now().UnixNano())
	f, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(backupPath)
		return "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(backupPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(backupPath)
		return "", err
	}
	return backupPath, nil
}

func (s *FileStorage) RestoreVaultBackup(backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return atomicWrite(s.vaultPath, data, 0o600)
}

func (s *FileStorage) ValidateVault() error {
	info, err := os.Stat(s.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return core.ErrVaultFileNotExists
		}
	}
	if info.Size() <= 19 {
		return core.ErrVaultFileCorrupted
	}

	if info.Mode().Type().Perm() != 0o600 {
		return core.ErrInvalidVaultPermissions
	}

	file, err := os.Open(s.vaultPath)
	if err != nil {
		return err
	}
	defer file.Close()

	magicBytes := make([]byte, 4)
	if _, err := file.Read(magicBytes); err != nil {
		return err
	}
	if !bytes.Equal(magicBytes, MagicBytes) {
		return core.ErrInvalidMagic
	}

	return nil
}

func (s *FileStorage) GetVaultSize() int64 {
	info, err := os.Stat(s.vaultPath)
	if err != nil {
		return 0
	}
	return info.Size()
}
