// Package server implements the daemon-side RPC handlers.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/dias-andre/shield/internal/config"
	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/migrations"
	"github.com/dias-andre/shield/internal/services"
)

type VaultState int

const (
	VaultStateLocked VaultState = iota
	VaultStateUnlocked
)

type Session struct {
	mu         sync.RWMutex
	createMu   sync.Mutex
	state      VaultState
	initState  core.VaultInitState
	initRecord *core.InitializationRecord

	backupTrigger chan struct{}
	lockTimer     *time.Timer

	vault     *core.Vault
	masterKey []byte

	vaultService     services.VaultService
	migrationManager *migrations.Manager
	backup           core.BackupPort
	initStore        *services.InitializationStore
	config           *config.Config
}

type SessionConfig struct {
	BackupSystem     core.BackupPort
	VaultService     services.VaultService
	MigrationManager *migrations.Manager
	Config           *config.Config
}

func NewSession(cfg SessionConfig) *Session {
	return &Session{
		backupTrigger: make(chan struct{}, 1),

		vaultService:     cfg.VaultService,
		migrationManager: cfg.MigrationManager,
		backup:           cfg.BackupSystem,
		config:           cfg.Config,
		state:            VaultStateLocked,
		initStore:        services.NewInitializationStore(filepath.Dir(cfg.Config.Vault.StorageDir)),
	}
}

func (s *Session) backupWorker() {
	for range s.backupTrigger {
		s.mu.RLock()
		vaultCopy := core.NewVault()
		s.vaultService.CopyVault(&vaultCopy, s.vault)
		s.mu.RUnlock()

		if err := s.backup.CreateBackup(&vaultCopy, s.masterKey); err != nil {
			slog.Error("failed to create backup", "error", err)
		}
	}
}

func (s *Session) requestAsyncBackup() {
	select {
	case s.backupTrigger <- struct{}{}:
	default:
	}
}

func (s *Session) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = ctx
	s.initState = core.InitDeferredLazy
	s.recordInit(core.InitDeferredLazy, "vault remains locked until PIN unlock")
	return nil
}

func (s *Session) UnlockWithPIN(pin []byte, create, reset bool) (string, error) {
	defer clear(pin)
	if create {
		s.createMu.Lock()
		defer s.createMu.Unlock()
	}
	s.mu.RLock()
	if s.state == VaultStateUnlocked && !create && !reset {
		s.mu.RUnlock()
		return "", nil
	}
	s.mu.RUnlock()
	if s.migrationManager == nil {
		return "", fmt.Errorf("vault key manager is not configured")
	}
	var result *migrations.Result
	var err error
	if reset {
		result, err = s.migrationManager.ResetWithPIN(pin)
	} else if create {
		result, err = s.migrationManager.InitializeWithPIN(pin)
	} else {
		result, err = s.migrationManager.UnlockWithPIN(pin)
	}
	if err != nil {
		return "", err
	}
	vault, key := result.Vault, result.Key
	if result.BackupPath != "" {
		slog.Info("vault migration completed", "from_version", result.FromVersion, "to_version", result.ToVersion, "backup", result.BackupPath)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockTimer != nil {
		s.lockTimer.Stop()
	}
	clear(s.masterKey)
	if s.vault != nil {
		s.vault.Erase()
	}
	s.masterKey, s.vault = key, vault
	s.state = VaultStateUnlocked
	minutes := s.config.Daemon.UnlockCacheMinutes
	if minutes <= 0 {
		minutes = 15
	}
	s.lockTimer = time.AfterFunc(time.Duration(minutes)*time.Minute, s.lockVault)
	return result.BackupPath, nil
}

func (s *Session) lockVault() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockTimer != nil {
		s.lockTimer.Stop()
		s.lockTimer = nil
	}
	clear(s.masterKey)
	s.masterKey = nil
	if s.vault != nil {
		s.vault.Erase()
		s.vault = nil
	}
	s.state = VaultStateLocked
}

func (s *Session) LockVault() { s.lockVault() }

func (s *Session) requireUnlocked() error {
	if s.state != VaultStateUnlocked || s.vault == nil {
		return fmt.Errorf("vault is locked; run `shield unlock`")
	}
	return nil
}

func (s *Session) recordInit(state core.VaultInitState, reason string) {
	record := &core.InitializationRecord{
		State:     state,
		ErrorMsg:  reason,
		Timestamp: time.Now(),
	}
	if err := s.initStore.Save(record); err != nil {
		slog.Error("failed to save init record", "error", err)
	}
}
