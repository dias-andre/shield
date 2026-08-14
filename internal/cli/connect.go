package cli

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"

	"github.com/dias-andre/shield/internal/core"
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
			return fmt.Errorf("failed to get master key: %w", err)
		}
		defer utils.Clear(masterKey)

		v, err := vaultSystem.GetVault(masterKey)
		if err != nil {
			return fmt.Errorf("failed to get vault: %w", err)
		}
		defer v.Erase()

		entry, ok := v.Entries[args[0]]
		if !ok {
			return fmt.Errorf("server '%s' not found", args[0])
		}
		fmt.Printf("Connecting to '%s'\n", entry.Name)
		if err := connectSSH(entry); err != nil {
			return fmt.Errorf("failed to connect to '%s': %w", entry.Name, err)
		}
		return nil
	},
}

func addKeyToAgent(privateKey []byte) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return errors.New("ssh-agent is not running: run `eval \"$(ssh-agent -s)\"` first")
	}

	cmd := exec.Command("ssh-add", "-")
	cmd.Stdin = bytes.NewReader(privateKey)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add failed: %v: %s", err, stderr.String())
	}
	return nil
}

func wipeAgentKeys() {
	cmd := exec.Command("ssh-add", "-D")
	if err := cmd.Run(); err != nil {
		slog.Warn("failed to wipe keys from ssh-agent", "error", err)
	}
}

func connectSSH(entry core.SSHEntry) error {
	var cmd *exec.Cmd

	if entry.AuthType != core.AuthMethodKey {
		return ErrNotPrivateKey
	}

	host, port, err := net.SplitHostPort(entry.Host)
	if err != nil {
		host = entry.Host
		port = "22"
	}

	if err := addKeyToAgent(entry.PrivateKey); err != nil {
		return fmt.Errorf("ssh-agent error: %w", err)
	}
	defer wipeAgentKeys()
	target := fmt.Sprintf("%s@%s", entry.User, host)
	cmd = exec.Command("ssh", "-p", port, target)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
