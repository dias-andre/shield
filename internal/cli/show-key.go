package cli

import (
	"fmt"
	"os"

	"github.com/dias-andre/shield/internal/api"
	"github.com/spf13/cobra"
)

var showKeyCmd = &cobra.Command{
	Use:   "show-key [server name]",
	Short: "Get the raw server key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if rpcErr := connectRPC(); rpcErr != nil {
			return rpcErr
		}

		req := api.FetchKeyRequest{
			EntryName: args[0],
		}
		reply := api.FetchKeyReply{}

		if serverErr := globalClient.Call("VaultServer.FetchKey", &req, &reply); serverErr != nil {
			return fmt.Errorf("unknown error: %w", serverErr)
		}
		if !reply.Success {
			if reply.ErrorCode == 404 {
				return fmt.Errorf("server '%s' not found", args[0])
			}
			return fmt.Errorf("unserialized error: %s", reply.ErrorMsg)
		}

		if _, err := os.Stdout.Write(reply.PrivateKey); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showKeyCmd)
}
