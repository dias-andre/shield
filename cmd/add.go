package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/dias-andre/shield/pkg/vault"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use: "add",
	Short: "Add entries to Vault",
}

var addServer = &cobra.Command{
	Use: "server [name] [user] [host] [authentication]",
	Short: "Add a new SSH server to Vault",
	// Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
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
				Help: "Example: 192.168.0.1 or myawspc",
			}
			err := survey.AskOne(promptName, &name)
			if err != nil {
				return
			}
		}

		if user == "" {
			promptUser := &survey.Input{
				Message: "Type your SSH user",
			}
			err := survey.AskOne(promptUser, &user)
			if err != nil {
				return
			}
		}

		if host == "" {
			promptHost := &survey.Input{
				Message: "Type your SSH host",
				Help: "Example: 192.168.15.1 or myserver.aws.com",
			}
			err := survey.AskOne(promptHost, &host)
			if err != nil {
				return
			}
		}

		if strings.HasPrefix(auth, "file:") {
			authMethod = string(vault.AuthMethodKey)
			auth = strings.ReplaceAll(auth, "file:", "")
		}

		if auth == "" {
			var selectedAuth string
			promptAuth := &survey.Select{
				Message: "What is your authentication method?",
				Options: []string{string(vault.AuthMethodPassword), string(vault.AuthMethodKey), string(vault.NoneAuthMethod)},
				Default: string(vault.AuthMethodPassword),
			}

			err := survey.AskOne(promptAuth, &selectedAuth)
			if err != nil {
				return
			}

			switch selectedAuth {
			case string(vault.AuthMethodPassword):
				err := survey.AskOne(&survey.Password{
					Message: "Type your SSH password",
				}, &auth)
				if err != nil { return }
				authMethod = selectedAuth
			case string(vault.AuthMethodKey):
				err := survey.AskOne(&survey.Input{
					Message: "Path to the private key (.pem or id_rsa):",
					Help: "Example: ~/.ssh/id_rsa or /path/to/your/key/ssh.pem",
				}, &auth)
				if err != nil {
					return
				}
				authMethod = selectedAuth
			default: 
				authMethod = string(vault.NoneAuthMethod)
			}
		}

		entry := vault.SSHEntry{
			Name: name,
			User: user,
			Port: 22,
			Host: host,
			AuthType: vault.AuthMethod(authMethod),
		}

		switch entry.AuthType {
		case vault.AuthMethodPassword:
			entry.Password = auth
		case vault.AuthMethodKey:
			// fmt.Println(auth)
			expandedPath, err := expandPath(auth)
			if err != nil {
				fmt.Printf("Operation failed: %s", err)
				os.Exit(1)
			}
			err = fileExistsValidator(expandedPath)
			if(err != nil) {
				fmt.Printf("File %s not found\n", expandedPath)
				os.Exit(1)
			}
			fileContent, err := os.ReadFile(expandedPath)
			if err != nil {
				fmt.Printf("Failed to Read file %s", auth)
				os.Exit(1)
			}

			entry.PrivateKey = string(fileContent)
			// fmt.Print(entry.PrivateKey)
		}
	},
}

func fileExistsValidator(path string) error {
	fullPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("Failed to Resolve User path")
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("File not found")
	}

	if info.IsDir() {
		return fmt.Errorf("The path is a directory")
	}
	return nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func init() {
	addCmd.AddCommand(addServer)
}
