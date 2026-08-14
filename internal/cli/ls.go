package cli

import (
	"fmt"

	"github.com/dias-andre/shield/internal/api"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all servers in Vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		rpcErr := connectRPC()
		if rpcErr != nil {
			return rpcErr
		}
		fetchEntriesRequest := api.FetchEntriesReply{}
		err := globalClient.Call("VaultServer.FetchEntries", new(api.EmptyRequest), &fetchEntriesRequest)
		if err != nil {
			return err
		}
		fmt.Println("NAME  USER  HOST")

		for _, entry := range fetchEntriesRequest.Entries {
			fmt.Printf("%s  %s  %s\n", entry.Name, entry.User, entry.Host)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
