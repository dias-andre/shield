package cmd

import (
	"fmt"

	"github.com/dias-andre/shield/internal/utils"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use: "ls",
	Short: "List all servers in Vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return fmt.Errorf("Failed to get master key: %s\n", err.Error())
		}
		defer utils.Clear(masterKey)

		v, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("Failed to get vault: %s\n", err.Error())
		}
		defer v.Erase()
		fmt.Println("NAME  USER  HOST")
		
		for name, entry := range v.Entries {
			fmt.Printf("%s  %s  %s\n", name, entry.User, entry.Host)
		}
		
		return nil
	},
}
