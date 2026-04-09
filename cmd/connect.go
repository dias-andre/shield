package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/dias-andre/shield/internal/core/domain"
	"github.com/dias-andre/shield/internal/utils"

	"github.com/spf13/cobra"
)

var ErrNotPrivateKey = errors.New("The Authentication is not a private key")

var connectCmd = &cobra.Command{
	Use: "connect [name]",
	Short: "Connect to a saved server",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return fmt.Errorf("Failed to get master key: %s", err.Error())
		}
		
		v, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("Failed to get vault: %s", err.Error())
		}
		defer utils.Clear(masterKey)
		defer v.Erase()

		entry, ok := v.Entries[args[0]]
		if !ok {
			return fmt.Errorf("Server '%s' not found.", args[0])
		}
		fmt.Printf("Connecting to '%s'\n", entry.Name)
		// fmt.Print(entry)
		err = connectSSH(entry)
		if err != nil {
			fmt.Println(err.Error())
		}
		// fmt.Print("\033[2A\033[J")
		fmt.Println("Shield closed successfully!")
		return nil
	},
}


func connectSSH(entry domain.SSHEntry) error {
	target := fmt.Sprintf("%s@%s", entry.User, entry.Host)

	var cmd *exec.Cmd

	if entry.AuthType != domain.AuthMethodKey {
		return ErrNotPrivateKey
	}

	tmpFile, err := os.CreateTemp("", "shield-key-*")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(entry.PrivateKey); err != nil {
		return err
	}

	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		return err
	}

	cmd = exec.Command("ssh", "-i", tmpPath, target)
	
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// fmt.Printf("Connecting to server '%s'...\n", entry.Name)

	return cmd.Run()
}
