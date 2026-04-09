package cmd

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dias-andre/shield/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize shield",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Green("Thank you for choose shield!\n\n")
		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

		sp.Suffix = "Generating Master Key and Creating Vault\n"
		sp.Start()

		key, _ := keysystem.GetKey()
		defer utils.Clear(key)
		if key != nil {
			sp.Suffix = "Stopped!"
			sp.Stop()
			color.Yellow("A master key already exists!")
		} else {
			key = make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				sp.Suffix = "Stopped!"
				sp.Stop()
				return fmt.Errorf("Failed to generate random key: %s", err.Error())
			}

			err := keysystem.SaveKey(key)
			if err != nil {
				sp.Suffix = "Stopped!"
				sp.Stop()
				return fmt.Errorf("Failed to save key: %s", err.Error())
			}
		}

		vaultExists, err := vaultSystem.VaultExists()
		if !vaultExists && err == nil {
			vault := vaultSystem.InitVault()
			defer vault.Erase()
			if err := vaultSystem.SaveVault(vault, key); err != nil {
				sp.Suffix = "Stopped!"
				sp.Stop()
				return fmt.Errorf("Failed to save file: %s", err.Error())
			}
			sp.Stop()
			color.Green("Vault created!")
			return nil
		} else if vaultExists {
			sp.Stop()
			color.Yellow("The vault file already exists!")
			return nil
		}
		
		sp.Stop()
		return fmt.Errorf("Failed to check vault health: %w", err)
	},
}
