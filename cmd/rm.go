package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var forceRm bool

var rmCmd = &cobra.Command{
	Use: "rm [server name]",
	Short: "Remove server",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return fmt.Errorf("Failed to get key: %w", err)
		}
		
		vault, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("Failed to get vault: %w", err)
		}
		
		if _, ok := vault.Entries[args[0]]; !ok {
			return fmt.Errorf("Server '%s' not found!", args[0])
		}
		
		if !forceRm {
			var confirm bool
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Are you sure you want to delete '%s'?", args[0]),
				Default: false,
			}
			err := survey.AskOne(prompt, &confirm)
			if err != nil { return fmt.Errorf("Operation failed: %w", err)}
			
			if !confirm {
				color.Yellow("Operation cancelled.")
				return nil
			}
		}
		
		err = vaultSystem.DeleteSshEntry(args[0], masterKey)
		if err != nil {
			return fmt.Errorf("Failed to delete entry: %w", err)
		}
		
		color.Green("Server '%s' removed successfully!", args[0])
		return nil
		
		// if server, ok := vault.Entries[args[0]]; ok {
		// 	var response string
		// 	prompt := &survey.Select{
		// 		Message: fmt.Sprintf("Are you sure to delete server '%s'?", server.Name),
		// 		Options: []string{"no", "yes"},
		// 		Default: "no",
		// 	}
		// 	err := survey.AskOne(prompt, &response)
		// 	if err != nil {
		// 		return fmt.Errorf("Operation failed: %w", err)
		// 	}
			
		// 	if response != "yes" {
		// 		color.Red("Operation cancelled!")
		// 		return nil
		// 	}
			
		// 	err = vaultSystem.DeleteSshEntry(args[0], masterKey)
		// 	if err != nil { return err }
			
		// 	color.Green("Server removed!")
		// 	return nil
		// }
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&forceRm, "force", "f", false, "Remove server without confirmation")
}