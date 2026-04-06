package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dias-andre/shield/internal/core/domain"

	"github.com/spf13/cobra"
)

var ErrNotPrivateKey = errors.New("The Authentication is not a private key")

var connectCmd = &cobra.Command{
	Use: "connect [name]",
	Short: "Connect to a saved server",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sp.Prefix = "Requesting master key...\n"
		sp.Start()

		masterKey, err := keysystem.GetKey()
		if err != nil {
			sp.FinalMSG = "Failed to get master key!\n"
			sp.Stop()
			os.Exit(1)
		}

		sp.FinalMSG = "Master key obtained!\n"
		sp.Stop()
		v, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			fmt.Println("Failed to get Vault")
			os.Exit(1)
		}

		entry, ok := v.Entries[args[0]]
		if !ok {
			fmt.Printf("Server '%s' not found\n", args[0])
			os.Exit(0)
		}
		fmt.Printf("Connecting to '%s'\n", entry.Name)
		// fmt.Print(entry)
		err = connectSSH(entry)
		if err != nil {
			fmt.Println(err.Error())
		}
		// fmt.Print("\033[2A\033[J")
		fmt.Println("Shield closed successfully!")
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

	if _, err := tmpFile.WriteString(entry.PrivateKey); err != nil {
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
