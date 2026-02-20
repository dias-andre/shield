package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dias-andre/shield/pkg/crypto"
	"github.com/dias-andre/shield/pkg/vault"

	"github.com/fatih/color"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use: "setup",
	Short: "Initialize shield",
	Run: func(cmd *cobra.Command, args []string) {
		color.HiGreen("Thank you for choose shield!\n\n")
		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

		sp.Suffix = "Generating Master Key and Creating Vault"
		sp.Start()

		key, err := crypto.GenerateAndStoreMasterKey()
		if err != nil {
			if errors.Is(err, crypto.ErrMasterKeyAlreadyExists) {
				sp.Stop()
				color.Green("A master key already exists!")
			} else {
				sp.Stop()
				color.RedString("A unknown error: %w\n", err)
				os.Exit(1)
			}
		}

		err = vault.InitVault(key)
		if err != nil {
			sp.Stop()
			color.RedString("A unknown error: %w\n", err)
			os.Exit(1)
		}

		sp.FinalMSG = "Shield Vault created!\n"
		sp.Stop()
		fmt.Printf("=> Key: %s\n", key)
	},
}
