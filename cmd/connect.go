package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/dias-andre/shield/internal/core/domain"
	"github.com/dias-andre/shield/internal/utils"

	"github.com/spf13/cobra"
)

var ErrNotPrivateKey = errors.New("the authentication method is not a private key")

var connectCmd = &cobra.Command{
	Use:   "connect [name]",
	Short: "Connect to a saved server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		masterKey, err := keysystem.GetKey()
		if err != nil {
			return fmt.Errorf("failed to get master key: %s", err.Error())
		}

		v, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("failed to get vault: %s", err.Error())
		}
		defer utils.Clear(masterKey)
		defer v.Erase()

		entry, ok := v.Entries[args[0]]
		if !ok {
			return fmt.Errorf("server '%s' not found ", args[0])
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

func addKeyToAgent(privateKey []byte) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return fmt.Errorf("ssh-agent is not running. Please, run with eval\"$(ssh-agent -s)\"")
	}

	cmd := exec.Command("ssh-add", "-")
	cmd.Stdin = bytes.NewReader(privateKey)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error: %v, stderr: %s", err, &stderr)
	}
	return nil
}

func wipeAgentKeys() {
	cmd := exec.Command("ssh-add", "-D")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Alert: agent keys not wiped: %s\n", err.Error())
	}
}

func connectSSH(entry domain.SSHEntry) error {
	var cmd *exec.Cmd

	if entry.AuthType != domain.AuthMethodKey {
		return ErrNotPrivateKey
	}

	host, port, err := net.SplitHostPort(entry.Host)
	if err != nil {
		host = entry.Host
		port = "22"
	}

	if err := addKeyToAgent(entry.PrivateKey); err != nil {
		return fmt.Errorf("ssh-agent error: %v", err)
	}
	defer wipeAgentKeys()
	target := fmt.Sprintf("%s@%s", entry.User, host)
	cmd = exec.Command("ssh", "-p", port, target)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// fmt.Printf("Connecting to server '%s'...\n", entry.Name)

	return cmd.Run()
}
