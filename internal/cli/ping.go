package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Check if shield service is running",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := connectRPC(); err != nil {
			return err
		}
		var request uint32 = 12
		var reply uint32
		cmd.Println("Checking service status...")
		if err := globalClient.Call("VaultServer.Ping", &request, &reply); err != nil {
			return fmt.Errorf("daemon was reached, but the ping failed: %w", err)
		}
		if reply == (request * 2) {
			cmd.Println("Shield service is active and responding!")
		} else {
			cmd.Println("The service responded, but returned a wrong value.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
