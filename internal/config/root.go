// Package config which stores ao shield configuration parsing
package config

import (
	"os"
	"path/filepath"

	"github.com/dias-andre/shield/internal/utils"
	"github.com/pelletier/go-toml/v2"
)

type VaultConfig struct {
	StorageDir string `toml:"storage_dir"`
}

type BackupConfig struct {
	Enabled     bool   `toml:"enabled"`
	Dir         string `toml:"dir"`
	MaxKeep     int    `toml:"max_keep"`
	ThreadLimit int    `toml:"thread_limit"`
}

type Config struct {
	Vault  VaultConfig  `toml:"vault"`
	Backup BackupConfig `toml:"backup"`
}

func DefaultConfig() (*Config, error) {
	dataPath, err := utils.GetDataPath()
	if err != nil {
		return nil, err
	}
	backupPath, err := utils.GetBackupDir()
	if err != nil {
		return nil, err
	}

	return &Config{
		Vault: VaultConfig{
			StorageDir: dataPath,
		},
		Backup: BackupConfig{
			Enabled:     true,
			Dir:         backupPath,
			MaxKeep:     10,
			ThreadLimit: 1,
		},
	}, nil
}

func LoadConfig(configDir string) (*Config, error) {
	cfg, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			return nil, err
		}
		data, err := toml.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
