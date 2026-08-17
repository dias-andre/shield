package cli

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"

	"github.com/dias-andre/shield/internal/api"
	"github.com/dias-andre/shield/internal/core"

	"github.com/spf13/cobra"
)

var ErrNotPrivateKey = errors.New("the authentication method is not a private key")

var connectCmd = &cobra.Command{
	Use:   "connect [name]",
	Short: "Connect to a saved server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if rpcErr := connectRPC(); rpcErr != nil {
			return rpcErr
		}
		request := api.GetCredentialsRequest{
			EntryName: args[0],
		}
		reply := api.GetCredentialsReply{}
		if err := globalClient.Call("VaultServer.OpenConnection", &request, &reply); err != nil {
			return err
		}
		if !reply.Success {
			if reply.ErrorCode == 404 {
				return fmt.Errorf("server '%s' not found", args[0])
			} else {
				return errors.New(reply.ErrorMsg)
			}
		}

		// SSH CONNECTION
		var command *exec.Cmd

		host, port, netErr := net.SplitHostPort(reply.Entry.Host)
		if netErr != nil {
			host = reply.Entry.Host
			port = "22"
		}

		if reply.AuthMethod == core.AuthMethodKey {
			if len(reply.PrivateKey) <= 0 {
				return errors.New("invalid private key")
			}
			if agentErr := addKeyToAgent(reply.PrivateKey); agentErr != nil {
				return fmt.Errorf("ssh-agent error: %w", agentErr)
			}
			defer wipeAgentKeys()
		}

		target := fmt.Sprintf("%s@%s", reply.Entry.User, host)
		command = exec.Command("ssh", "-p", port, target)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		fmt.Printf("Connecting to '%s'\n", reply.Entry.Name)
		return command.Run()
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

func init() {
	rootCmd.AddCommand(connectCmd)
}
