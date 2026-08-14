package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dias-andre/shield/internal/api"
	"github.com/dias-andre/shield/internal/core"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add entries to Vault",
}

var addServer = &cobra.Command{
	Use:   "server [name] [user] [host] [authentication]",
	Short: "Add a new SSH server to Vault",
	// Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		rpcErr := connectRPC()
		if rpcErr != nil {
			return rpcErr
		}
		var name, user, host, auth, authMethod string

		if len(args) > 0 {
			name = args[0]
		}
		if len(args) > 1 {
			user = args[1]
		}
		if len(args) > 2 {
			host = args[2]
		}
		if len(args) > 3 {
			auth = args[3]
		}

		if name == "" {
			promptName := &survey.Input{
				Message: "Create a name for your server",
				Help:    "Example: 192.168.0.1 or myawspc",
			}
			err := survey.AskOne(promptName, &name)
			if err != nil {
				return fmt.Errorf("failed to read prompt: %w", err)
			}
		}

		if user == "" {
			promptUser := &survey.Input{
				Message: "Type your SSH user",
			}
			err := survey.AskOne(promptUser, &user)
			if err != nil {
				return fmt.Errorf("failed to read prompt: %w", err)
			}
		}

		if host == "" {
			promptHost := &survey.Input{
				Message: "Type your SSH host",
				Help:    "Example: 192.168.15.1 or myserver.aws.com",
			}
			err := survey.AskOne(promptHost, &host)
			if err != nil {
				return fmt.Errorf("failed to read prompt: %w", err)
			}
		}

		if strings.HasPrefix(auth, "file:") {
			authMethod = string(core.AuthMethodKey)
			auth = strings.ReplaceAll(auth, "file:", "")
		}

		if auth == "" {
			var selectedAuth string
			promptAuth := &survey.Select{
				Message: "What is your authentication method?",
				Options: []string{string(core.AuthMethodKey), string(core.NoneAuthMethod)},
				Default: string(core.NoneAuthMethod),
			}

			err := survey.AskOne(promptAuth, &selectedAuth)
			if err != nil {
				return fmt.Errorf("failed to read prompt: %w", err)
			}

			if selectedAuth == string(core.AuthMethodKey) {
				err := survey.AskOne(&survey.Input{
					Message: "Path to the private key (.pem or id_rsa):",
					Help:    "Example: ~/.ssh/id_rsa or /path/to/your/key/ssh.pem",
				}, &auth)
				if err != nil {
					return fmt.Errorf("failed to read prompt: %w", err)
				}
				authMethod = selectedAuth

			} else {
				authMethod = string(core.NoneAuthMethod)
			}
		}

		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sp.Suffix = "Storing your SSH credentials\n"
		sp.Start()
		defer sp.Stop()

		request := api.CreateSSHEntryRequest{
			Name:     name,
			User:     user,
			Host:     host,
			AuthType: core.AuthMethod(authMethod),
		}
		reply := api.CreateSSHEntryReply{}

		if request.AuthType == core.AuthMethodKey {
			expandedPath, err := resolvePath(auth)
			if err != nil {
				return fmt.Errorf("failed to resolve key path: %w", err)
			}
			if err := fileExistsValidator(expandedPath); err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			request.KeyLocation = expandedPath
		}
		if err := globalClient.Call("VaultServer.CreateEntry", &request, &reply); err != nil {
			sp.FinalMSG = "Operation failed!\n"
			return fmt.Errorf("failed to call daemon: %w", err)
		}
		if !reply.Success {
			sp.FinalMSG = "Operation failed!\n"
			return fmt.Errorf("failed to save SSH entry: %s", reply.ErrorMsg)
		}
		sp.FinalMSG = "SSH credentials saved!\n"
		return nil
	},
}

func fileExistsValidator(path string) error {
	fullPath, err := resolvePath(path)
	if err != nil {
		return fmt.Errorf("failed to resolve user path: %w", err)
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}

	if info.IsDir() {
		return fmt.Errorf("the path is a directory")
	}
	return nil
}

func resolvePath(path string) (string, error) {
	path = strings.TrimSpace(path)

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, path[2:])
	}
	return filepath.Abs(path)
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addServer)
}
