package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use: "ping",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := connectRPC(); err != nil {
			return err
		}
		var request uint32 = 12
		var reply uint32
		fmt.Printf("Calling shield daemon ping with number: %d\n", request)
		if err := globalClient.Call("VaultServer.Ping", &request, &reply); err != nil {
			return fmt.Errorf("failed to call VaultServer.Ping: %w", err)
		}
		fmt.Printf("The shield daemon returned: %d\n", reply)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
