package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dias-andre/shield/internal/core/domain"
	"github.com/dias-andre/shield/internal/utils"

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
				return fmt.Errorf("Operation failed: %s", err.Error())
			}
		}

		if user == "" {
			promptUser := &survey.Input{
				Message: "Type your SSH user",
			}
			err := survey.AskOne(promptUser, &user)
			if err != nil {
				return fmt.Errorf("Operation failed: %s", err.Error())
			}
		}

		if host == "" {
			promptHost := &survey.Input{
				Message: "Type your SSH host",
				Help:    "Example: 192.168.15.1 or myserver.aws.com",
			}
			err := survey.AskOne(promptHost, &host)
			if err != nil {
				return fmt.Errorf("Operation failed: %s", err.Error())
			}
		}

		if strings.HasPrefix(auth, "file:") {
			authMethod = string(domain.AuthMethodKey)
			auth = strings.ReplaceAll(auth, "file:", "")
		}

		if auth == "" {
			var selectedAuth string
			promptAuth := &survey.Select{
				Message: "What is your authentication method?",
				Options: []string{string(domain.AuthMethodKey), string(domain.NoneAuthMethod)},
				Default: string(domain.NoneAuthMethod),
			}

			err := survey.AskOne(promptAuth, &selectedAuth)
			if err != nil {
				return fmt.Errorf("Operation failed: %s", err.Error())
			}
	
			if(selectedAuth == string(domain.AuthMethodKey)) {
				err := survey.AskOne(&survey.Input{
					Message: "Path to the private key (.pem or id_rsa):",
					Help:    "Example: ~/.ssh/id_rsa or /path/to/your/key/ssh.pem",
				}, &auth)
				
				if err != nil {
					return fmt.Errorf("Operation failed: %s", err.Error())
				}
				authMethod = selectedAuth

			} else {
				authMethod = string(domain.NoneAuthMethod)
			}	
		}

		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sp.Suffix = "Storing your SSH Credentials\n"
		sp.Start()

		entry := domain.SSHEntry{
			Name:     name,
			User:     user,
			Port:     22,
			Host:     host,
			AuthType: domain.AuthMethod(authMethod),
		}

		if (entry.AuthType == domain.AuthMethodKey) {
			// fmt.Println(auth)
			expandedPath, err := expandPath(auth)
			if err != nil {
				return fmt.Errorf("Operation failed: %s", err.Error())
			}
			err = fileExistsValidator(expandedPath)
			if err != nil {
				return fmt.Errorf("Failed to read file: %s", err.Error())
			}
			fileContent, err := os.ReadFile(expandedPath)
			if err != nil {
				sp.Stop()
				return fmt.Errorf("Failed to read file: %s", err.Error())
			}

			entry.PrivateKey = fileContent
			// fmt.Print(entry.PrivateKey)
		}

		masterKey, err := keysystem.GetKey()
		if err != nil {
			sp.Stop()
			return fmt.Errorf("Failed to get master key: %s", err.Error())
		}
		defer utils.Clear(masterKey)

		err =  vaultSystem.AddSshEntry(entry, masterKey)
		if err != nil {
			sp.Stop()
			return fmt.Errorf("Failed to save credentials: %s", err.Error())
		}

		sp.FinalMSG = "SSH Credentials saved!\n"
		sp.Stop()
		return nil
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
