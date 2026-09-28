// Package migrations inspects vault formats and executes versioned upgrade paths.
package migrations

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
)

type KeySource interface {
	GetKey() ([]byte, error)
}

type KeyStore interface {
	KeySource
	SaveKey([]byte) error
}

type Manager struct {
	mu              sync.Mutex
	vault           *services.VaultService
	keyShare        KeyStore  // key-a, persisted in the OS Secret Service
	legacyMasterKey KeySource // old master-key, used only for v0.1/v0.2 migration
}

type Inspection struct {
	Vault                  services.VaultInfo
	KeyShareChecked        bool
	KeyShareExists         bool
	KeyShareValid          bool
	KeyShareError          string
	LegacyMasterKeyChecked bool
	LegacyMasterKeyExists  bool
	LegacyMasterKeyError   string
}

func NewManager(vault *services.VaultService, keyShare KeyStore, legacyMasterKey KeySource) *Manager {
	return &Manager{vault: vault, keyShare: keyShare, legacyMasterKey: legacyMasterKey}
}

func (m *Manager) Inspect() (Inspection, error) {
	info, err := m.vault.InspectVault()
	if err != nil {
		return Inspection{}, err
	}
	out := Inspection{Vault: info}
	if m.keyShare != nil {
		key, keyErr := m.keyShare.GetKey()
		out.KeyShareChecked = true
		if keyErr != nil {
			out.KeyShareError = keyErr.Error()
		} else {
			out.KeyShareExists = len(key) > 0
			out.KeyShareValid = len(key) == 32
		}
		clear(key)
	}
	if info.Compatibility == services.VaultLegacy && m.legacyMasterKey != nil {
		key, keyErr := m.legacyMasterKey.GetKey()
		out.LegacyMasterKeyChecked = true
		if errors.Is(keyErr, core.ErrMasterKeyNotFound) {
			out.LegacyMasterKeyExists = false
		} else if keyErr != nil {
			out.LegacyMasterKeyError = keyErr.Error()
		} else {
			out.LegacyMasterKeyExists = len(key) > 0
		}
		clear(key)
	}
	return out, nil
}

type Result struct {
	FromVersion string
	ToVersion   string
	BackupPath  string
	Vault       *core.Vault
	Key         []byte
}

func (m *Manager) ensureKeyShare() ([]byte, error) {
	if m.keyShare == nil {
		return nil, fmt.Errorf("OS Secret Service key share is unavailable")
	}
	key, err := m.keyShare.GetKey()
	if err != nil {
		return nil, fmt.Errorf("read Secret Service key share: %w", err)
	}
	if len(key) == 32 {
		return key, nil
	}
	clear(key)
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		clear(key)
		return nil, err
	}
	if err := m.keyShare.SaveKey(key); err != nil {
		clear(key)
		return nil, fmt.Errorf("save Secret Service key share: %w", err)
	}
	return key, nil
}

// InitializeWithPIN creates key-a if absent, then creates a fresh PIN protected vault.
func (m *Manager) InitializeWithPIN(pin []byte) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	info, err := m.vault.InspectVault()
	if err != nil {
		return nil, err
	}
	if info.Compatibility != services.VaultAbsent {
		return nil, fmt.Errorf("cannot initialize vault: %s (version %s): %s", info.Compatibility, info.Version, info.Reason)
	}
	share, err := m.ensureKeyShare()
	if err != nil {
		return nil, err
	}
	defer clear(share)
	vault, key, err := m.vault.CreateVaultWithPIN(pin, share)
	if err != nil {
		return nil, err
	}
	return &Result{ToVersion: fmt.Sprintf("%d.%d.%d", services.VaultFormatMajor, services.VaultFormatMinor, services.VaultFormatPatch), Vault: vault, Key: key}, nil
}

// ResetWithPIN explicitly replaces any existing vault with an empty hybrid vault.
// The previous vault and any existing key-a are backed up before key replacement.
func (m *Manager) ResetWithPIN(pin []byte) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keyShare == nil {
		return nil, fmt.Errorf("OS Secret Service key share is unavailable")
	}
	info, err := m.vault.InspectVault()
	if err != nil {
		return nil, err
	}
	var backupPath string
	if info.Exists {
		backupPath, err = m.vault.BackupVault()
		if err != nil {
			return nil, fmt.Errorf("preserve existing vault before reset: %w", err)
		}
	}
	oldShare, err := m.keyShare.GetKey()
	if err != nil {
		return nil, fmt.Errorf("inspect existing Secret Service key-a: %w", err)
	}
	if len(oldShare) > 0 && backupPath != "" {
		f, createErr := os.OpenFile(backupPath+".key-a", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if createErr != nil {
			clear(oldShare)
			return nil, fmt.Errorf("preserve existing key-a: %w", createErr)
		}
		if _, err = f.Write(oldShare); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			clear(oldShare)
			_ = os.Remove(backupPath + ".key-a")
			return nil, fmt.Errorf("preserve existing key-a: %w", err)
		}
	}
	defer clear(oldShare)
	newShare := make([]byte, 32)
	if _, err := rand.Read(newShare); err != nil {
		return nil, err
	}
	defer clear(newShare)
	if err := m.keyShare.SaveKey(newShare); err != nil {
		m.restoreResetState(oldShare, backupPath)
		return nil, fmt.Errorf("save new Secret Service key-a: %w", err)
	}
	vault := m.vault.InitVault()
	key, err := m.vault.SaveVaultWithPIN(&vault, pin, newShare)
	if err != nil {
		m.restoreResetState(oldShare, backupPath)
		return nil, fmt.Errorf("create replacement vault: %w", err)
	}
	verified, verifyKey, err := m.vault.GetVaultWithPIN(pin, newShare)
	clear(verifyKey)
	if err != nil {
		clear(key)
		m.restoreResetState(oldShare, backupPath)
		return nil, fmt.Errorf("verify replacement vault: %w", err)
	}
	return &Result{FromVersion: info.Version, ToVersion: fmt.Sprintf("%d.%d.%d", services.VaultFormatMajor, services.VaultFormatMinor, services.VaultFormatPatch), BackupPath: backupPath, Vault: verified, Key: key}, nil
}

func (m *Manager) restoreResetState(oldShare []byte, backupPath string) {
	if len(oldShare) > 0 {
		_ = m.keyShare.SaveKey(oldShare)
	}
	if backupPath != "" {
		_ = m.vault.RestoreVaultBackup(backupPath)
	}
}

// UnlockWithPIN opens the current format or migrates recognized legacy formats.
func (m *Manager) UnlockWithPIN(pin []byte) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	info, err := m.vault.InspectVault()
	if err != nil {
		return nil, err
	}
	if info.Compatibility == services.VaultAbsent {
		return nil, fmt.Errorf("no vault exists; initialize it first")
	}
	if info.Compatibility == services.VaultUnsupported || info.Compatibility == services.VaultInvalid {
		return nil, fmt.Errorf("cannot open vault %s (%s): %s", info.Version, info.Compatibility, info.Reason)
	}
	if info.Compatibility == services.VaultCompatible {
		share, err := m.ensureExistingKeyShare()
		if err != nil {
			return nil, err
		}
		defer clear(share)
		vault, key, err := m.vault.GetVaultWithPIN(pin, share)
		if err != nil {
			return nil, err
		}
		return &Result{FromVersion: info.Version, ToVersion: info.Version, Vault: vault, Key: key}, nil
	}
	var vault *core.Vault
	switch info.Compatibility {
	case services.VaultPINMigration:
		vault, err = m.vault.GetVaultWithLegacyPIN(pin)
	case services.VaultLegacy:
		vault, err = m.openLegacyMasterKeyVault()
	default:
		return nil, fmt.Errorf("no migration path registered for vault %s (%s)", info.Version, info.Compatibility)
	}
	if err != nil {
		return nil, fmt.Errorf("could not open legacy vault; migration left the source intact: %w", err)
	}
	share, err := m.ensureKeyShare()
	if err != nil {
		return nil, err
	}
	defer clear(share)
	migrated, err := m.vault.MigrateVault(vault, pin, share, info.Version)
	if err != nil {
		return nil, err
	}
	return &Result{FromVersion: migrated.FromVersion, ToVersion: migrated.ToVersion, BackupPath: migrated.BackupPath, Vault: migrated.Vault, Key: migrated.Key}, nil
}

func (m *Manager) ensureExistingKeyShare() ([]byte, error) {
	if m.keyShare == nil {
		return nil, fmt.Errorf("OS Secret Service key share is unavailable")
	}
	key, err := m.keyShare.GetKey()
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		clear(key)
		return nil, fmt.Errorf("required Secret Service key share `key-a` is missing or invalid")
	}
	return key, nil
}

func (m *Manager) openLegacyMasterKeyVault() (*core.Vault, error) {
	if m.legacyMasterKey == nil {
		return nil, fmt.Errorf("legacy master-key source is unavailable")
	}
	key, err := m.legacyMasterKey.GetKey()
	if err != nil {
		return nil, fmt.Errorf("load legacy master key: %w", err)
	}
	defer clear(key)
	if len(key) == 0 {
		return nil, fmt.Errorf("legacy master key not found in Secret Service")
	}
	return m.vault.GetVault(key)
}
