// Copyright (C) 2026 André de Oliveira Dias (diaso.andre@outlook.com)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package cmd is the root cobra package, which manages the shield commands and flags.
package cmd

import (
	"fmt"
	"os"

	"github.com/dias-andre/shield/internal/adapters"
	"github.com/dias-andre/shield/internal/core"
	"github.com/dias-andre/shield/internal/services"
	"github.com/dias-andre/shield/internal/utils"
	"github.com/spf13/cobra"
)

var (
	vaultSystem services.VaultService
	keysystem   core.KeySystemPort
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
	keysystem = adapters.NewKeyringSystem()

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(rmCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
