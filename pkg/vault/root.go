package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dias-andre/shield/pkg/crypto"
)

var ErrVaultFileNotExists = errors.New("Vault file not exists!")

func InitVault(masterKey string) error {
	v := NewVault()
	return saveVaultToFile(v, masterKey)
}

func getDataPath() (string, error) {
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

	dataPath, err := getDataPath()
	if err != nil {
		return v, err
	}

	encryptedContent, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return v, ErrVaultFileNotExists
		}
		
		return v, err	
	}

	plaintext, err := crypto.DecryptVault(encryptedContent, masterKey)
	if err != nil { return v, err }

	err = json.Unmarshal(plaintext, &v)
	if err != nil { return v, err }
	return v, nil
}

func AddSshEntry(entry SSHEntry, masterKey string) error {
	v, err := GetVault(masterKey)

	if err != nil {
		if errors.Is(err, ErrVaultFileNotExists) {
			v = NewVault()
		} else {
			return err
		}
	}	

	v.Entries[entry.Name] = entry

	return saveVaultToFile(v, masterKey)
}

func saveVaultToFile(v Vault, masterkey string) error {
	jsonData, err := json.Marshal(v)
	if err != nil { return err }

	vaultEncrypted, err := crypto.EncryptVault(jsonData, masterkey)
	if err != nil { return err }

	path, err := getDataPath()
	if err != nil { return err }

	return os.WriteFile(path, vaultEncrypted, 0600)
}
