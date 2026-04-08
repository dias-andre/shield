//go:build linux

package cmd

import (
	"fmt"

	"github.com/dias-andre/shield/internal/core/ports"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use: "lock",
	Short: "Lock shield",
	RunE: func(cmd *cobra.Command, args []string) error {
		lockable, ok := keysystem.(ports.Lockable)

		if !ok {
			return fmt.Errorf("The current keysystem adapter does not support locking")
		}

		err := lockable.Lock()
		if err != nil {
			return fmt.Errorf("Failed to lock shield: %w", err)
		}

		color.Green("🔒 Shield locked successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
