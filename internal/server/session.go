// Package server implements the daemon-side RPC handlers.
package server

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
)

type Session struct {
	mu              sync.RWMutex
	backupTrigger   chan struct{}
	vault           *core.Vault
	masterKey       []byte
	keySystem       core.KeySystemPort
	partedKeySystem core.KeySystemPort
	vaultService    services.VaultService
	backup          core.BackupPort
}

func NewSession(ks core.KeySystemPort, ps core.KeySystemPort, vs services.VaultService, bp core.BackupPort) *Session {
	return &Session{
		backupTrigger:   make(chan struct{}, 1),
		keySystem:       ks,
		partedKeySystem: ps,
		vaultService:    vs,
		backup:          bp,
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

func (s *Session) Init() error {
	s.mu.Lock()

	key, err := s.keySystem.GetKey()
	if err != nil {
		return fmt.Errorf("failed to load master key: %w", err)
	}
	if key == nil {
		slog.Warn("master key not found")
		slog.Info("generating master key")
		if err := s.genKey(); err != nil {
			return err
		}
	}
	s.masterKey = make([]byte, len(key))
	copy(s.masterKey, key)
	slog.Info("session master key loaded")

	vaultExists, err := s.vaultService.VaultExists()
	if err != nil {
		return err
	}
	if !vaultExists {
		slog.Warn("vault not found")
		slog.Info("initializing new vault")
		if err := s.initNewVault(); err != nil {
			return err
		}
	}
	vault, err := s.vaultService.GetVault(s.masterKey)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}
	s.vault = vault
	s.mu.Unlock()
	slog.Info("session vault loaded")

	go s.backupWorker()
	slog.Info("backup thread spawned")
	return nil
}

func (s *Session) genKey() error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	if err := s.keySystem.SaveKey(key); err != nil {
		return err
	}
	s.masterKey = make([]byte, len(key))
	copy(s.masterKey, key)
	return nil
}

func (s *Session) initNewVault() error {
	vault := s.vaultService.InitVault()
	if err := s.vaultService.SaveVault(&vault, s.masterKey); err != nil {
		return err
	}
	s.vault = &vault
	return nil
}

func (s *Session) Destroy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	utils.Clear(s.masterKey)
	s.vault.Erase()
}

func (s *Session) Setup() error {
	slog.Info("starting shield setup")
	key, err := s.keySystem.GetKey()
	if err != nil {
		return err
	}
	if key != nil {
		slog.Info("master key already exists")
	} else {
		slog.Info("generating new master key")
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return err
		}
		if err := s.keySystem.SaveKey(key); err != nil {
			return err
		}
		slog.Info("master key saved to keyring")
	}

	vaultExists, err := s.vaultService.VaultExists()
	if err != nil {
		return err
	}
	if !vaultExists {
		slog.Info("initializing new vault")
		vault := s.vaultService.InitVault()
		if err := s.vaultService.SaveVault(&vault, key); err != nil {
			return err
		}
		slog.Info("vault initialized")
		return nil
	}

	return nil
}
