package cmd

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize shield",
	Run: func(cmd *cobra.Command, args []string) {
		color.HiGreen("Thank you for choose shield!\n\n")
		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

		sp.Suffix = "Generating Master Key and Creating Vault\n"
		sp.Start()

		previousKey, _ := keysystem.GetKey()
		if previousKey != nil {
			sp.Suffix = "Stopped!"
			sp.Stop()
			color.HiGreen("A master key already exists!")
			os.Exit(0)
		}

		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			sp.Suffix = "Stopped!"
			sp.Stop()
			fmt.Println(err.Error())
			os.Exit(1)
		}

		err := keysystem.SaveKey(key)
		if err != nil {
			sp.Suffix = "Stopped!"
			sp.Stop()
			color.HiRed("Failed to save key...")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		vault := vaultSystem.InitVault()
		if err := vaultSystem.SaveVault(vault, key); err != nil {
			sp.Suffix = "Stopped!"
			sp.Stop()
			color.HiRed("Failed to save vault...")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		sp.FinalMSG = "Shield Vault created!\n"
		sp.Stop()
	},
}
