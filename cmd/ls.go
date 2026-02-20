package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dias-andre/shield/pkg/crypto"
	"github.com/dias-andre/shield/pkg/vault"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use: "ls",
	Short: "List all servers in Vault",
	Run: func(cmd *cobra.Command, args []string) {
		masterKey, err := crypto.GetMasterKey()
		if err != nil {
			color.RedString("Failed to get master key: %s\n", err.Error())
			os.Exit(1)
		}

		v, err := vault.GetVault(masterKey)
		if err != nil {
			os.Exit(1)
		}
		fmt.Println("NAME  USER  HOST  AUTH")
		
		for name, entry := range v.Entries {
			fmt.Printf("%s  %s  %s  %s\n", name, entry.User, entry.Host, strings.ToUpper(string(entry.AuthType)))
		}
	},
}
