package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

const SocketPath = "/tmp/shield.sock"

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

func Clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
