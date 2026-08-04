package cmd

import (
	"fmt"
	"os"

	"github.com/dias-andre/shield/internal/utils"
	"github.com/spf13/cobra"
)

var showKeyCmd = &cobra.Command{
	Use:   "show-key",
	Short: "Get the raw server key",
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return err
		}
		defer utils.Clear(masterKey)

		vault, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("failed to get vault: %v", err)
		}
		defer vault.Erase()

		entry, ok := vault.Entries[args[0]]
		if !ok {
			return fmt.Errorf("server '%s' not found", args[0])
		}

		_, err = os.Stdout.Write(entry.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to write on stdout: %w", err)
		}
		// fmt.Fprintf(os.Stderr, "DEBUG: Size of decripted key %d bytes\n", len(entry.PrivateKey))

		return nil
	},
}
