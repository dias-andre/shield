// Package services orchestrates the ports and adapters that back the shield vault.
package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
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

type VaultCompatibility string

const (
	VaultAbsent       VaultCompatibility = "absent"
	VaultCompatible   VaultCompatibility = "compatible"
	VaultLegacy       VaultCompatibility = "legacy_migration_required"
	VaultPINMigration VaultCompatibility = "pin_format_migration_required"
	VaultUnsupported  VaultCompatibility = "unsupported_version"
	VaultInvalid      VaultCompatibility = "invalid"
)

type VaultInfo struct {
	Exists        bool
	Version       string
	Compatibility VaultCompatibility
	Reason        string
}

// InspectVault checks existence, parses the on-disk header and determines whether
// this daemon can open the vault with the current PIN based format.
func (s *VaultService) InspectVault() (VaultInfo, error) {
	exists, err := s.storage.VaultExists()
	if err != nil {
		return VaultInfo{}, fmt.Errorf("check vault existence: %w", err)
	}
	if !exists {
		return VaultInfo{Exists: false, Compatibility: VaultAbsent}, nil
	}
	rawStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return VaultInfo{Exists: true, Compatibility: VaultInvalid, Reason: "storage does not support versioned vaults"}, nil
	}
	raw, err := rawStorage.LoadRawVault()
	if err != nil {
		if errors.Is(err, core.ErrInvalidMagic) {
			legacyBytes, loadErr := s.storage.Load()
			if loadErr == nil && len(legacyBytes) > 12 {
				return VaultInfo{Exists: true, Version: "legacy-unversioned", Compatibility: VaultLegacy, Reason: "unversioned vault requires legacy master-key migration"}, nil
			}
		}
		return VaultInfo{Exists: true, Compatibility: VaultInvalid, Reason: err.Error()}, nil
	}
	version := fmt.Sprintf("%d.%d.%d", raw.Version.Major, raw.Version.Minor, raw.Version.Patch)
	info := VaultInfo{Exists: true, Version: version}
	switch {
	case raw.Version.Major == VaultFormatMajor && raw.Version.Minor == VaultFormatMinor && raw.Version.Patch == VaultFormatPatch:
		info.Compatibility = VaultCompatible
	case raw.Version.Major == VaultFormatMajor && raw.Version.Minor == LegacyPINFormatMinor && raw.Version.Patch == 0:
		info.Compatibility = VaultPINMigration
		info.Reason = "PIN vault predates the Secret Service key share and requires migration"
	case raw.Version.Major == VaultFormatMajor && raw.Version.Minor < VaultFormatMinor:
		info.Compatibility = VaultLegacy
		info.Reason = "vault uses a legacy master-key format and requires migration"
	default:
		info.Compatibility = VaultUnsupported
		info.Reason = "vault format version is not supported by this daemon"
	}
	return info, nil
}

const (
	VaultFormatMajor            = 0
	VaultFormatMinor            = 4
	VaultFormatPatch            = 0
	LegacyPINFormatMinor        = 3
	argonTime            uint32 = 3
	argonMemory          uint32 = 64 * 1024
	argonThreads         uint8  = 2
	argonKeyLen          uint32 = 32
)

func derivePINKey(pin []byte, salt []byte) []byte {
	return argon2.IDKey(pin, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func deriveVaultKey(pin, salt, keyShare []byte) ([]byte, error) {
	if len(keyShare) != 32 {
		return nil, fmt.Errorf("invalid Secret Service key share length")
	}
	pinKey := derivePINKey(pin, salt)
	defer zero(pinKey)
	mac := hmac.New(sha256.New, keyShare)
	_, _ = mac.Write([]byte("shield-vault-master-key-v1\x00"))
	_, _ = mac.Write(pinKey)
	return mac.Sum(nil), nil
}

// GetVaultWithPIN decrypts a version 0.3 vault using a key derived from its stored salt.
func (s *VaultService) GetVaultWithPIN(pin, keyShare []byte) (*core.Vault, []byte, error) {
	rawStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return nil, nil, fmt.Errorf("invalid storage system")
	}
	raw, err := rawStorage.LoadRawVault()
	if err != nil {
		return nil, nil, err
	}
	if raw.Version.Major != VaultFormatMajor || raw.Version.Minor != VaultFormatMinor || raw.Version.Patch != VaultFormatPatch {
		info, _ := s.InspectVault()
		return nil, nil, fmt.Errorf("cannot unlock vault format %s (%s): %s", info.Version, info.Compatibility, info.Reason)
	}
	key, err := deriveVaultKey(pin, raw.Salt[:], keyShare)
	if err != nil {
		return nil, nil, err
	}
	v, err := s.GetVault(key)
	if err != nil {
		zero(key)
		return nil, nil, err
	}
	return v, key, nil
}

// GetVaultWithLegacyPIN opens the intermediate 0.3 PIN-only format for migration.
func (s *VaultService) GetVaultWithLegacyPIN(pin []byte) (*core.Vault, error) {
	rawStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return nil, fmt.Errorf("invalid storage system")
	}
	raw, err := rawStorage.LoadRawVault()
	if err != nil {
		return nil, err
	}
	if raw.Version.Major != VaultFormatMajor || raw.Version.Minor != LegacyPINFormatMinor || raw.Version.Patch != 0 {
		return nil, fmt.Errorf("vault is not the legacy PIN-only format")
	}
	key := derivePINKey(pin, raw.Salt[:])
	defer zero(key)
	return s.GetVault(key)
}

// CreateVaultWithPIN creates a new version 0.3 vault and returns its derived key.
func (s *VaultService) CreateVaultWithPIN(pin, keyShare []byte) (*core.Vault, []byte, error) {
	info, err := s.InspectVault()
	if err != nil {
		return nil, nil, err
	}
	if info.Compatibility != VaultAbsent {
		return nil, nil, fmt.Errorf("cannot create vault: %s (version %s): %s", info.Compatibility, info.Version, info.Reason)
	}
	vault := s.InitVault()
	key, err := s.SaveVaultWithPIN(&vault, pin, keyShare)
	if err != nil {
		return nil, nil, err
	}
	return &vault, key, nil
}

// SaveVaultWithPIN encrypts a vault with a fresh salt and the current PIN format.
func (s *VaultService) SaveVaultWithPIN(vault *core.Vault, pin, keyShare []byte) ([]byte, error) {
	var salt [16]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return nil, err
	}
	key, err := deriveVaultKey(pin, salt[:], keyShare)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(&vault)
	if err != nil {
		zero(key)
		return nil, err
	}
	encrypted, err := s.crypto.Encrypt(data, key)
	if err != nil {
		zero(key)
		return nil, err
	}
	raw := &core.RawVault{Version: core.SemVer{Major: VaultFormatMajor, Minor: VaultFormatMinor, Patch: VaultFormatPatch}, Salt: salt}
	copy(raw.Nonce[:], encrypted[:12])
	raw.Ciphertext = append([]byte(nil), encrypted[12:]...)
	rawStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		zero(key)
		return nil, fmt.Errorf("invalid storage system")
	}
	if err := rawStorage.SaveRawVault(raw); err != nil {
		zero(key)
		return nil, err
	}
	return key, nil
}

type MigrationResult struct {
	FromVersion string
	ToVersion   string
	BackupPath  string
	Vault       *core.Vault
	Key         []byte
}

// MigrateLegacyVault converts a legacy-key encrypted vault to the PIN format.
// The original file is preserved before it is replaced.
func (s *VaultService) MigrateVault(vault *core.Vault, pin, keyShare []byte, fromVersion string) (*MigrationResult, error) {
	info, err := s.InspectVault()
	if err != nil {
		return nil, err
	}
	if info.Compatibility != VaultLegacy && info.Compatibility != VaultPINMigration {
		return nil, fmt.Errorf("vault is not a migratable format: %s (%s)", info.Version, info.Compatibility)
	}
	backupStorage, ok := s.storage.(core.SupportVaultBackup)
	if !ok {
		return nil, fmt.Errorf("storage does not support safe migration backups")
	}
	backupPath, err := backupStorage.BackupVault()
	if err != nil {
		return nil, fmt.Errorf("preserve original vault: %w", err)
	}
	newKey, err := s.SaveVaultWithPIN(vault, pin, keyShare)
	if err != nil {
		return nil, fmt.Errorf("write migrated vault: %w (original preserved at %s)", err, backupPath)
	}
	// Re-open the just-written file before reporting success.
	verified, verifyKey, err := s.GetVaultWithPIN(pin, keyShare)
	zero(verifyKey)
	if err != nil {
		if restore, ok := s.storage.(core.SupportVaultRestore); ok {
			_ = restore.RestoreVaultBackup(backupPath)
		}
		return nil, fmt.Errorf("verify migrated vault: %w (original preserved at %s)", err, backupPath)
	}
	if fromVersion == "" {
		fromVersion = info.Version
	}
	return &MigrationResult{FromVersion: fromVersion, ToVersion: fmt.Sprintf("%d.%d.%d", VaultFormatMajor, VaultFormatMinor, VaultFormatPatch), BackupPath: backupPath, Vault: verified, Key: newKey}, nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

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
			slog.Info("deprecated unversioned vault format detected")
			rawVault, err = s.ParseBytesToRawVault(result)
			if err != nil {
				return nil, fmt.Errorf("failed to parse legacy vault format: %w", err)
			}
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
	rawVaultStorage, ok := s.storage.(core.SupportRawVault)
	if !ok {
		return errors.New("invalid storage system")
	}
	current, currentErr := rawVaultStorage.LoadRawVault()
	if currentErr != nil {
		return currentErr
	}
	rawVault := &core.RawVault{Version: current.Version, Salt: current.Salt}
	copy(rawVault.Nonce[:], encryptedVault[:12])
	rawVault.Ciphertext = append([]byte(nil), encryptedVault[12:]...)
	return rawVaultStorage.SaveRawVault(rawVault)
}

func (s *VaultService) VaultExists() (bool, error) {
	return s.storage.VaultExists()
}

func (s *VaultService) BackupVault() (string, error) {
	backup, ok := s.storage.(core.SupportVaultBackup)
	if !ok {
		return "", fmt.Errorf("storage does not support vault backups")
	}
	return backup.BackupVault()
}

func (s *VaultService) RestoreVaultBackup(path string) error {
	restore, ok := s.storage.(core.SupportVaultRestore)
	if !ok {
		return fmt.Errorf("storage does not support vault backup restoration")
	}
	return restore.RestoreVaultBackup(path)
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

func (s *VaultService) CopyVault(dst *core.Vault, src *core.Vault) {
	*dst = *src
	if src.Entries != nil {
		dst.Entries = make(map[string]core.SSHEntry, len(src.Entries))
		for key, srcEntry := range src.Entries {
			dstEntry := srcEntry
			if dstEntry.AuthType == core.AuthMethodKey && srcEntry.PrivateKey != nil {
				dstEntry.PrivateKey = make([]byte, len(srcEntry.PrivateKey))
				copy(dstEntry.PrivateKey, srcEntry.PrivateKey)
			}
			dst.Entries[key] = dstEntry

		}
	}
}
