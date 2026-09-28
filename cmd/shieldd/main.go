package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/config"
	"github.com/dias-andre/shield/internal/migrations"
	"github.com/dias-andre/shield/internal/server"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
)

func getHostSession() *server.Session {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	configDir, err := utils.GetConfigDir()
	if err != nil {
		slog.Error("failed to resolve configuration file", "error", err)
		os.Exit(1)
	}
	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		slog.Error("failed to parse configuration file, check config.toml", "directory", configDir, "error", err)
	}
	slog.Info("configuration loaded")
	vaultPath, err := utils.GetDataPath()
	if err != nil {
		slog.Error("failed to resolve vault path", "error", err)
		os.Exit(1)
	}
	backupPath, err := utils.GetBackupDir()
	if err != nil {
		slog.Error("failed to resolve backup directory", "error", err)
		os.Exit(1)
	}

	storage := adapters.NewFileSystemStorage(vaultPath)
	encryptor := adapters.NewAESEncryptor()
	backup := adapters.NewLocalFileBackup(backupPath, encryptor, uint8(cfg.Backup.MaxKeep))
	service := services.NewVaultService(encryptor, storage)
	keyShare, keyShareErr := adapters.NewPartedKeyring()
	if keyShareErr != nil {
		slog.Warn("Secret Service key share unavailable; setup/unlock requiring key-a will fail", "error", keyShareErr)
	}
	legacyMasterKey, legacyKeyErr := adapters.NewKeyringSystem()
	if legacyKeyErr != nil {
		slog.Warn("legacy master key unavailable; legacy vault migration will not be available", "error", legacyKeyErr)
	}
	migrationManager := migrations.NewManager(&service, keyShare, legacyMasterKey)

	host := server.NewSession(server.SessionConfig{
		BackupSystem:     backup,
		VaultService:     service,
		MigrationManager: migrationManager,
		Config:           cfg,
	})
	return host
}

func main() {
	socketPath := utils.GetSocket()
	if fsErr := os.MkdirAll(socketPath, 0o700); fsErr != nil {
		slog.Error("failed to prepare socket", "path", socketPath, "err", fsErr)
		os.Exit(1)
	}

	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			slog.Error("failed to remove stale socket file", "path", socketPath, "error", err)
			os.Exit(1)
		}
		slog.Info("removed stale socket file", "path", socketPath)
	}
	host := getHostSession()
	ctx := context.Background()
	if err := host.Init(ctx); err != nil {
		slog.Error("failed to initialize host", "error", err)
		os.Exit(1)
	}

	if err := rpc.RegisterName("VaultServer", host); err != nil {
		slog.Error("failed to register RPC host", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		slog.Error("failed to listen on socket", "path", socketPath, "error", err)
		os.Exit(1)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		slog.Error("failed to set socket permissions", "error", err)
		os.Exit(1)
	}
	slog.Info("daemon started", "socket", socketPath)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				slog.Error("failed to accept connection", "error", err)
				continue
			}
			if !isAuthorized(conn) {
				if err := conn.Close(); err != nil {
					slog.Error("failed to close unauthorized connection", "error", err)
				}
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	if err := listener.Close(); err != nil {
		slog.Error("failed to close listener", "error", err)
	}
}
