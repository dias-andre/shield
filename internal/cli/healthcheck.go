package cli

import (
	"errors"
	"fmt"

	"github.com/dias-andre/shield/internal/core"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var healthCheckCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify vault encryption",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("Verifying vault integrity (it doesn't use your master key)")
		sizeCheckPass := true
		magicBytesPass := true
		permissionsPass := false

		status, err := vaultSystem.CheckVaultHealth()
		if !status {
			if errors.Is(err, core.ErrVaultFileCorrupted) {
				sizeCheckPass = false
			}
			if errors.Is(err, core.ErrInvalidMagic) {
				magicBytesPass = false
			}
			if errors.Is(err, core.ErrInvalidVaultPermissions) {
				permissionsPass = false
			}
		}

		if sizeCheckPass {
			color.Green("✓ Minimum size check\n")
		} else {
			color.Red("✗ Minimum size check\n")
		}

		if magicBytesPass {
			color.Green("✓ Structural Shield format\n")
		} else {
			color.Red("✗ Structural Shield format\n")
		}

		if permissionsPass {
			color.Green("✓ File permissions\n")
		} else {
			color.Red("✗ File permissions\n")
			fmt.Println("Verification failed: The vault permissions are too open.")
			fmt.Println("Expected -rw------- (0600), but found something else.")

			fmt.Println("\nAction required:")
			fmt.Println("The vault needs 0600 permissions.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCheckCmd)
}
