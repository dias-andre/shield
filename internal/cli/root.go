// Copyright (C) 2026 André de Oliveira Dias (diaso.andre@outlook.com)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package cli implements the shield CLI commands and flags.
package cli

import (
	"fmt"
	"net/rpc"
	"os"

	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
	"github.com/spf13/cobra"
)

var (
	vaultSystem  services.VaultService
	keysystem    core.KeySystemPort
	globalClient *rpc.Client
)

var rootCmd = &cobra.Command{
	Use:           "shield",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       "0.1.0",
	Short:         "Tool for managing encrypted SSH keys",
}

func init() {
	datapath, err := utils.GetDataPath()
	if err != nil {
		fmt.Println("Failed to load configuration")
		os.Exit(1)
	}
	encryptor := adapters.NewAESEncryptor()
	repo := adapters.NewFileSystemStorage(datapath)
	vaultSystem = services.NewVaultService(encryptor, repo)
	keysystem, err = adapters.NewKeyringSystem()
	if err != nil {
		fmt.Printf("Failed to initialize keyring system: %v\n", err)
		os.Exit(1)
	}
}

func connectRPC() error {
	if globalClient != nil {
		return nil
	}
	client, err := rpc.Dial("unix", utils.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to shield daemon: %w (is shieldd running?)", err)
	}
	globalClient = client
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		if globalClient != nil {
			_ = globalClient.Close()
		}
		os.Exit(1)
	}
	if globalClient != nil {
		_ = globalClient.Close()
	}
}
