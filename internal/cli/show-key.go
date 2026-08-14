package cli

import (
	"fmt"
	"os"

	"github.com/dias-andre/shield/internal/utils"
	"github.com/spf13/cobra"
)

var showKeyCmd = &cobra.Command{
	Use:   "show-key [server name]",
	Short: "Get the raw server key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return err
		}
		defer utils.Clear(masterKey)

		vault, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("failed to get vault: %w", err)
		}
		defer vault.Erase()

		entry, ok := vault.Entries[args[0]]
		if !ok {
			return fmt.Errorf("server '%s' not found", args[0])
		}

		if _, err := os.Stdout.Write(entry.PrivateKey); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showKeyCmd)
}
