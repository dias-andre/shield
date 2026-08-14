package cli

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/dias-andre/shield/internal/api"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var forceRm bool

var rmCmd = &cobra.Command{
	Use:   "rm [server name]",
	Short: "Remove server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if rpcErr := connectRPC(); rpcErr != nil {
			return rpcErr
		}
		request := api.GetServerEntryRequest{
			Name: args[0],
		}
		reply := api.GetServerEntryReply{}
		if err := globalClient.Call("VaultServer.GetServerEntry", &request, &reply); err != nil {
			return err
		}
		if !reply.Success {
			return fmt.Errorf("server '%s' not found", args[0])
		}

		if !forceRm {
			var confirm bool
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Are you sure you want to delete '%s'?", args[0]),
				Default: false,
			}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				return err
			}
			if !confirm {
				color.Yellow("Operation cancelled.")
				return nil
			}
		}
		deleteReq := api.RemoveSSHEntryRequest{
			Name: args[0],
		}
		deleteReply := api.RemoveSSHEntryReply{}
		if deleteErr := globalClient.Call("VaultServer.RemoveEntry", &deleteReq, &deleteReply); deleteErr != nil {
			return fmt.Errorf("failed to remove entry: %w", deleteErr)
		}
		if !deleteReply.Success {
			return fmt.Errorf("failed to remove entry: %s", deleteReply.ErrorMsg)
		}
		color.Green("Server '%s' removed successfully!", args[0])

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&forceRm, "force", "f", false, "Remove server without confirmation")
}
