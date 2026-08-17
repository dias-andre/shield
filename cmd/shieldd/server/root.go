// Package server implements the daemon-side RPC handlers.
package server

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dias-andre/shield/internal/api"
	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
)

type Session struct {
	mu           sync.RWMutex
	vault        *core.Vault
	masterKey    []byte
	keySystem    core.KeySystemPort
	vaultService services.VaultService
}

func NewSession(ks core.KeySystemPort, vs services.VaultService) *Session {
	return &Session{
		keySystem:    ks,
		vaultService: vs,
	}
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
	}
	return nil
}

func (s *Session) Init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.keySystem.GetKey()
	if err != nil {
		return fmt.Errorf("failed to load master key: %w", err)
	}
	if key == nil {
		return errors.New("master key not found")
	}
	s.masterKey = key
	slog.Info("session master key loaded")

	vaultExists, err := s.vaultService.VaultExists()
	if err != nil {
		return err
	}
	if !vaultExists {
		return errors.New("vault not found")
	}
	vault, err := s.vaultService.GetVault(s.masterKey)
	if err != nil {
		return fmt.Errorf("failed to load vault: %w", err)
	}
	s.vault = vault
	slog.Info("session vault loaded")
	return nil
}

func (s *Session) Destroy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	utils.Clear(s.masterKey)
	s.vault.Erase()
}

func (s *Session) Ping(req *uint32, reply *uint32) error {
	slog.Debug("ping received", "number", *req)
	*reply = *req * 2
	return nil
}

func (s *Session) FetchEntries(_ api.EmptyRequest, reply *api.FetchEntriesReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	reply.Entries = make([]api.ServerEntry, 0, len(s.vault.Entries))
	for name, entry := range s.vault.Entries {
		serverEntry := api.ServerEntry{
			Name: name,
			User: entry.User,
			Host: entry.Host,
		}
		reply.Entries = append(reply.Entries, serverEntry)
	}
	return nil
}

func (s *Session) CreateEntry(req *api.CreateSSHEntryRequest, reply *api.CreateSSHEntryReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	slog.Info("creating SSH entry", "name", req.Name)

	_, alreadyExists := s.vault.Entries[req.Name]
	if alreadyExists {
		reply.Success = false
		reply.ErrorCode = 400
		reply.ErrorMsg = "name already exists"
		return nil
	}

	newEntry := core.SSHEntry{
		Name:     req.Name,
		User:     req.User,
		Host:     req.Host,
		AuthType: req.AuthType,
	}

	if newEntry.AuthType == core.AuthMethodKey {
		newEntry.PrivateKey = make([]byte, len(req.PrivateKey))
		copy(newEntry.PrivateKey, req.PrivateKey)
	}

	s.vault.Entries[newEntry.Name] = newEntry

	if syncErr := s.vaultService.SaveVault(s.vault, s.masterKey); syncErr != nil {
		reply.Success = false
		reply.ErrorCode = 500
		reply.ErrorMsg = fmt.Sprintf("failed to save vault: %v", syncErr)
		delete(s.vault.Entries, req.Name)
		return nil
	}
	reply.Success = true
	slog.Info("SSH entry created", "name", req.Name, "auth", newEntry.AuthType)
	return nil
}

func (s *Session) GetServerEntry(req *api.GetServerEntryRequest, reply *api.GetServerEntryReply) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.vault.Entries[req.Name]
	if exists {
		*reply = api.GetServerEntryReply{
			Entry: api.ServerEntry{
				Name: req.Name,
				User: entry.User,
				Host: entry.Host,
			},
			Success: true,
		}
		return nil
	}
	reply.Success = false
	slog.Warn("entry lookup failed", "name", req.Name)
	return nil
}

func (s *Session) RemoveEntry(req *api.RemoveSSHEntryRequest, reply *api.RemoveSSHEntryReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.vault.Entries[req.Name]
	if exists {
		delete(s.vault.Entries, req.Name)

		if syncErr := s.vaultService.SaveVault(s.vault, s.masterKey); syncErr != nil {
			reply.Success = false
			reply.ErrorMsg = fmt.Sprintf("failed to save vault: %v", syncErr)
			slog.Error("failed to remove entry", "name", req.Name, "error", syncErr)
			s.vault.Entries[entry.Name] = entry
			return nil
		}
		reply.Success = true
		slog.Info("entry removed", "name", req.Name)
		return nil
	}
	reply.Success = false
	reply.ErrorMsg = fmt.Sprintf("server '%s' not found", req.Name)
	slog.Warn("entry removal requested for unknown entry", "name", req.Name)
	return nil
}

func (s *Session) OpenConnection(req *api.GetCredentialsRequest, reply *api.GetCredentialsReply) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.vault.Entries[req.EntryName]
	if !exists {
		reply.Success = false
		reply.ErrorCode = 404
		reply.ErrorMsg = fmt.Sprintf("server '%s' not found", req.EntryName)
		slog.Warn("connection requested for unknown entry", "name", req.EntryName)
		return nil
	}

	*reply = api.GetCredentialsReply{
		Entry: api.ServerEntry{
			Name: entry.Name,
			Host: entry.Host,
			User: entry.User,
		},
		AuthMethod: entry.AuthType,
		Success:    true,
	}
	reply.PrivateKey = make([]byte, len(entry.PrivateKey))
	copy(reply.PrivateKey, entry.PrivateKey)
	slog.Info("entry connection requested", "name", req.EntryName, "auth", entry.AuthType)
	return nil
}

func (s *Session) FetchKey(req *api.FetchKeyRequest, reply *api.FetchKeyReply) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.vault.Entries[req.EntryName]

	if !exists {
		reply.ErrorCode = 404
		reply.ErrorMsg = fmt.Sprintf("server '%s' not found", req.EntryName)
		reply.Success = false
		return nil
	}
	reply.PrivateKey = make([]byte, len(entry.PrivateKey))
	copy(reply.PrivateKey, entry.PrivateKey)
	reply.Success = true
	return nil
}
