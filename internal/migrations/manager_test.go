package migrations

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
	"golang.org/x/crypto/argon2"
)

type memoryKeyStore struct{ key []byte }

func (m *memoryKeyStore) GetKey() ([]byte, error)  { return append([]byte(nil), m.key...), nil }
func (m *memoryKeyStore) SaveKey(key []byte) error { m.key = append([]byte(nil), key...); return nil }

func newTestManager(t *testing.T, legacyKeys KeySource) (*Manager, *memoryKeyStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "keys.vault")
	storage := adapters.NewFileSystemStorage(path)
	encryptor := adapters.NewAESEncryptor()
	vaultService := services.NewVaultService(encryptor, storage)
	keyShare := &memoryKeyStore{}
	return NewManager(&vaultService, keyShare, legacyKeys), keyShare, path
}

func TestInitializeInspectAndUnlockHybridVault(t *testing.T) {
	manager, keyShare, _ := newTestManager(t, nil)
	before, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if before.Vault.Compatibility != services.VaultAbsent || !before.KeyShareChecked || before.KeyShareExists {
		t.Fatalf("unexpected pre-setup inspection: %+v", before)
	}

	created, err := manager.InitializeWithPIN([]byte("correct horse battery"))
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	clear(created.Key)
	if len(keyShare.key) != 32 {
		t.Fatalf("expected a 32-byte key share, got %d bytes", len(keyShare.key))
	}

	inspection, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Vault.Compatibility != services.VaultCompatible || !inspection.KeyShareExists || !inspection.KeyShareValid {
		t.Fatalf("unexpected post-setup inspection: %+v", inspection)
	}

	opened, err := manager.UnlockWithPIN([]byte("correct horse battery"))
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	clear(opened.Key)
	if _, err := manager.UnlockWithPIN([]byte("wrong PIN")); err == nil {
		t.Fatal("unlock with an incorrect PIN unexpectedly succeeded")
	}
}

func TestUnlockMigratesLegacyMasterKeyVaultAndPreservesBackup(t *testing.T) {
	legacyMasterKey := &memoryKeyStore{key: bytes.Repeat([]byte{0x5a}, 32)}
	manager, keyShare, path := newTestManager(t, legacyMasterKey)

	vault := core.NewVault()
	vault.Entries["test"] = core.SSHEntry{Name: "test", User: "alice", Host: "example.test", AuthType: core.NoneAuthMethod}
	plain, err := json.Marshal(&vault)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := adapters.NewAESEncryptor().Encrypt(plain, legacyMasterKey.key)
	if err != nil {
		t.Fatal(err)
	}
	var nonce [12]byte
	copy(nonce[:], encrypted[:12])
	raw := &core.RawVault{Version: core.SemVer{Major: 0, Minor: 2, Patch: 0}, Nonce: nonce, Ciphertext: append([]byte(nil), encrypted[12:]...)}
	storage := adapters.NewFileSystemStorage(path).(core.SupportRawVault)
	if err := storage.SaveRawVault(raw); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	inspection, err := manager.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Vault.Compatibility != services.VaultLegacy || !inspection.LegacyMasterKeyExists {
		t.Fatalf("unexpected legacy inspection: %+v", inspection)
	}

	result, err := manager.UnlockWithPIN([]byte("a sufficiently long PIN"))
	if err != nil {
		t.Fatalf("migrate and unlock: %v", err)
	}
	defer clear(result.Key)
	if result.FromVersion != "0.2.0" || result.ToVersion != "0.4.0" || result.BackupPath == "" {
		t.Fatalf("unexpected migration result: %+v", result)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("read migration backup: %v", err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatal("migration backup does not contain the original vault")
	}
	if len(keyShare.key) != 32 {
		t.Fatal("migration did not create the Secret Service key share")
	}
	if result.Vault.Entries["test"].Host != "example.test" {
		t.Fatal("entry was not preserved by migration")
	}

	opened, err := manager.UnlockWithPIN([]byte("a sufficiently long PIN"))
	if err != nil {
		t.Fatalf("unlock migrated vault: %v", err)
	}
	defer clear(opened.Key)
	if opened.Vault.Entries["test"].User != "alice" {
		t.Fatal("migrated vault data did not survive reopening")
	}
}

func TestUnlockMigratesIntermediatePINVault(t *testing.T) {
	manager, keyShare, path := newTestManager(t, nil)
	pin := []byte("intermediate PIN phrase")
	salt := bytes.Repeat([]byte{0x31}, 16)
	legacyKey := argon2.IDKey(pin, salt, 3, 64*1024, 2, 32)
	vault := core.NewVault()
	vault.Entries["test"] = core.SSHEntry{Name: "test", Host: "legacy.test", User: "bob"}
	plain, err := json.Marshal(&vault)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := adapters.NewAESEncryptor().Encrypt(plain, legacyKey)
	clear(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	var storedSalt [16]byte
	copy(storedSalt[:], salt)
	var nonce [12]byte
	copy(nonce[:], encrypted[:12])
	storage := adapters.NewFileSystemStorage(path).(core.SupportRawVault)
	if err := storage.SaveRawVault(&core.RawVault{Version: core.SemVer{Major: 0, Minor: 3, Patch: 0}, Salt: storedSalt, Nonce: nonce, Ciphertext: append([]byte(nil), encrypted[12:]...)}); err != nil {
		t.Fatal(err)
	}

	result, err := manager.UnlockWithPIN(pin)
	if err != nil {
		t.Fatalf("migrate PIN-only vault: %v", err)
	}
	defer clear(result.Key)
	if result.FromVersion != "0.3.0" || result.ToVersion != "0.4.0" {
		t.Fatalf("unexpected migration result: %+v", result)
	}
	if len(keyShare.key) != 32 {
		t.Fatal("migration did not create key-a")
	}
	if result.Vault.Entries["test"].Host != "legacy.test" {
		t.Fatal("entry was not preserved by migration")
	}
}

func TestResetPreservesVaultAndKeyShareBackups(t *testing.T) {
	manager, keyShare, path := newTestManager(t, nil)
	created, err := manager.InitializeWithPIN([]byte("original PIN phrase"))
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	clear(created.Key)
	originalVault, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	originalShare := append([]byte(nil), keyShare.key...)

	reset, err := manager.ResetWithPIN([]byte("replacement PIN phrase"))
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	defer clear(reset.Key)
	if reset.BackupPath == "" {
		t.Fatal("reset did not report the original-vault backup path")
	}
	backupVault, err := os.ReadFile(reset.BackupPath)
	if err != nil {
		t.Fatalf("read vault backup: %v", err)
	}
	if !bytes.Equal(backupVault, originalVault) {
		t.Fatal("reset backup differs from original vault")
	}
	backupShare, err := os.ReadFile(reset.BackupPath + ".key-a")
	if err != nil {
		t.Fatalf("read key-a backup: %v", err)
	}
	if !bytes.Equal(backupShare, originalShare) {
		t.Fatal("key-a backup differs from original share")
	}
	if bytes.Equal(keyShare.key, originalShare) {
		t.Fatal("reset reused the original Secret Service share")
	}
	if len(reset.Vault.Entries) != 0 {
		t.Fatal("reset vault is not empty")
	}
	opened, err := manager.UnlockWithPIN([]byte("replacement PIN phrase"))
	if err != nil {
		t.Fatalf("unlock reset vault: %v", err)
	}
	clear(opened.Key)
}

func TestResetReplacesUnrecoverableLegacyVault(t *testing.T) {
	manager, _, path := newTestManager(t, nil)
	storage := adapters.NewFileSystemStorage(path).(core.SupportRawVault)
	legacyBytes := []byte("legacy encrypted bytes")
	var nonce [12]byte
	copy(nonce[:], bytes.Repeat([]byte{0x77}, 12))
	if err := storage.SaveRawVault(&core.RawVault{
		Version:    core.SemVer{Major: 0, Minor: 1, Patch: 0},
		Nonce:      nonce,
		Ciphertext: legacyBytes,
	}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result, err := manager.ResetWithPIN([]byte("new empty vault PIN"))
	if err != nil {
		t.Fatalf("reset legacy vault without its old master key: %v", err)
	}
	defer clear(result.Key)
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("read preserved legacy vault: %v", err)
	}
	if !bytes.Equal(backup, original) {
		t.Fatal("reset did not preserve the unrecoverable legacy vault")
	}
	if result.ToVersion != "0.4.0" || len(result.Vault.Entries) != 0 {
		t.Fatalf("unexpected reset result: %+v", result)
	}
}

func clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
