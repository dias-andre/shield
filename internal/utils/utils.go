package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetDataPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	appDataDir := filepath.Join(dataHome, "shield")

	if err := os.MkdirAll(appDataDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return filepath.Join(appDataDir, "keys.vault"), nil
}

func GetSocket() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	socketDir := filepath.Join(runtimeDir, "shield")
	return filepath.Join(socketDir, "shield.sock")
}

func GetBackupDir() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	appBackupDir := filepath.Join(dataHome, "shield", ".snapshots")
	return appBackupDir, nil
}

func GetConfigDir() (string, error) {
	dataHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataHome, ".config", "shield"), nil
}

func Clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
