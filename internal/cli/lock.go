//go:build linux

package cli

import (
	"fmt"

	"github.com/dias-andre/shield/internal/api"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock shield",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := connectRPC(); err != nil {
			return err
		}
		reply := api.UnlockReply{}
		if err := globalClient.Call("VaultServer.Lock", api.EmptyRequest{}, &reply); err != nil {
			return fmt.Errorf("failed to lock shield: %w", err)
		}
		fmt.Println("Shield locked.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
