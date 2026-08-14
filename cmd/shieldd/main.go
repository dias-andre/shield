package main

import (
	"errors"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/dias-andre/shield/cmd/shieldd/server"
	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if _, err := os.Stat(utils.SocketPath); err == nil {
		if err := os.Remove(utils.SocketPath); err != nil {
			slog.Error("failed to remove stale socket file", "path", utils.SocketPath, "error", err)
			os.Exit(1)
		}
		slog.Info("removed stale socket file", "path", utils.SocketPath)
	}

	keysystem, err := adapters.NewKeyringSystem()
	if err != nil {
		slog.Error("failed to initialize keyring system", "error", err)
		os.Exit(1)
	}
	vaultPath, err := utils.GetDataPath()
	if err != nil {
		slog.Error("failed to resolve vault path", "error", err)
		os.Exit(1)
	}
	storage := adapters.NewFileSystemStorage(vaultPath)
	encryptor := adapters.NewAESEncryptor()
	service := services.NewVaultService(encryptor, storage)

	host := server.NewSession(keysystem, service)
	if err := host.Init(); err != nil {
		slog.Error("failed to initialize host", "error", err)
		os.Exit(1)
	}
	defer host.Destroy()

	if err := rpc.RegisterName("VaultServer", host); err != nil {
		slog.Error("failed to register RPC host", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("unix", utils.SocketPath)
	if err != nil {
		slog.Error("failed to listen on socket", "path", utils.SocketPath, "error", err)
		os.Exit(1)
	}
	if err := os.Chmod(utils.SocketPath, 0o600); err != nil {
		slog.Error("failed to set socket permissions", "error", err)
		os.Exit(1)
	}
	slog.Info("daemon started", "socket", utils.SocketPath)

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
