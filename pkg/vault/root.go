package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"shield/pkg/crypto"
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

	if err := os.MkdirAll(appDataDir, 0700); err != nil {
		return "", fmt.Errorf("Failed to create data directory: %w", err)
	}

	return filepath.Join(appDataDir, "keys.vault"), nil
}

func GetVault(masterKey string) (Vault, error) {
	var v Vault

	dataPath, err := GetDataPath()
	if err != nil {
		return v, err
	}

	encryptedContent, err := os.ReadFile(dataPath)
	if err != nil {
		return v, err
	}

	plaintext, err := crypto.DecryptVault(encryptedContent, masterKey)
	err = json.Unmarshal(plaintext, &v)
	return v, nil
}

func SaveVault(encryptedPayload []byte) error {
	path, err := GetDataPath()
	if err != nil {
		return err
	}

	return os.WriteFile(path, encryptedPayload, 0600)
}
