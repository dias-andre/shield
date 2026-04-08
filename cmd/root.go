package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/core/ports"
	"github.com/dias-andre/shield/internal/services"
	"github.com/spf13/cobra"
)

var vaultSystem services.VaultService
var keysystem ports.KeySystemPort

var rootCmd = &cobra.Command{
	Use: "shield",
	SilenceErrors: true,
	SilenceUsage: true,
	Short: "Tool for managing encrypted server keys",
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

func init() {
	datapath, err := getDataPath()
	if err != nil {
		fmt.Println("Failed to load configuration")
		os.Exit(1)
	}
	
	encryptor := adapters.NewAESEncryptor()
	repo := adapters.NewFileSystemStorage(datapath)
	vaultSystem = services.NewVaultService(encryptor, repo)
	keysystem = adapters.NewKeyringSystem()

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(connectCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

