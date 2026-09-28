package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/dias-andre/shield/internal/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func readPIN(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	pin, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if len(pin) == 0 {
		return nil, fmt.Errorf("PIN cannot be empty")
	}
	return pin, nil
}

var setupForce bool

func unlock(create bool) error {
	reset := create && setupForce
	if create {
		fmt.Println("Starting Shield vault setup…")
	} else {
		fmt.Println("Connecting to the Shield daemon…")
	}
	if err := connectRPC(); err != nil {
		return err
	}
	fmt.Println("Checking vault format and Secret Service key availability…")
	info := api.InspectVaultReply{}
	if err := globalClient.Call("VaultServer.InspectVault", &api.EmptyRequest{}, &info); err != nil {
		return fmt.Errorf("failed to inspect vault: %w", err)
	}
	if create {
		if !reset && info.Compatibility != "absent" {
			if info.Exists {
				return fmt.Errorf("setup refused: vault already exists (version %s, status %s): %s; use `shield unlock` or migrate the vault", info.Version, info.Compatibility, info.Reason)
			}
			return fmt.Errorf("setup refused: vault state is %s: %s", info.Compatibility, info.Reason)
		}
		if !info.KeyShareChecked {
			return fmt.Errorf("cannot inspect Secret Service; start the daemon in your desktop session and try again")
		}
		if info.KeyShareError != "" {
			return fmt.Errorf("cannot inspect Secret Service key-a: %s", info.KeyShareError)
		}
		if reset && info.Exists {
			fmt.Printf("Force setup will replace vault version %s. The daemon will preserve a recovery copy first.\n", info.Version)
			if info.Compatibility == "legacy_migration_required" && info.LegacyMasterKeyChecked && !info.LegacyMasterKeyExists {
				fmt.Println("The old master key is unavailable; this backup is preserved, but it cannot be decrypted unless that key is recovered.")
			}
		}
		if info.KeyShareExists {
			if info.KeyShareValid {
				if reset {
					fmt.Println("Found a valid Secret Service key share (key-a); force setup will back it up and create a new one.")
				} else {
					fmt.Println("Found a valid Secret Service key share (key-a); setup will reuse it.")
				}
			} else {
				fmt.Println("Found an invalid Secret Service key-a; the daemon will replace it while creating the new vault.")
			}
		} else if info.KeyShareChecked && info.KeyShareError == "" {
			fmt.Println("No Secret Service key share found; the daemon will create key-a during setup.")
		} else if info.KeyShareError != "" {
			fmt.Printf("Could not inspect Secret Service key-a: %s\n", info.KeyShareError)
		}
	} else if !info.Exists {
		return fmt.Errorf("no vault exists; run `shield setup` first")
	} else {
		fmt.Printf("Found vault version %s (%s).\n", info.Version, info.Compatibility)
		if !info.KeyShareChecked {
			return fmt.Errorf("cannot access the Secret Service key share; the daemon has no keyring adapter")
		}
		if info.KeyShareError != "" {
			return fmt.Errorf("cannot access the Secret Service key share: %s", info.KeyShareError)
		}
		if info.Compatibility == "legacy_migration_required" {
			if !info.LegacyMasterKeyChecked {
				return fmt.Errorf("cannot inspect the legacy master key; the daemon has no legacy keyring adapter")
			}
			if info.LegacyMasterKeyExists {
				fmt.Println("Legacy master key found in Secret Service; it can be used for migration.")
			} else if info.LegacyMasterKeyError != "" {
				fmt.Printf("Could not inspect legacy master key: %s\n", info.LegacyMasterKeyError)
				return fmt.Errorf("cannot migrate this vault without reading the legacy master key")
			} else {
				fmt.Println("Legacy master key was not found in Secret Service.")
				return fmt.Errorf("cannot migrate this vault: the legacy master key is unavailable")
			}
			fmt.Println("Unlock will migrate this vault to the hybrid PIN + Secret Service format.")
		} else if info.Compatibility == "pin_format_migration_required" {
			fmt.Println("Unlock will add the Secret Service key share and migrate this vault to the hybrid format.")
		} else if info.Compatibility == "compatible" && !info.KeyShareValid {
			return fmt.Errorf("required Secret Service key share key-a is missing; the vault cannot be unlocked with this keyring")
		}
	}
	if reset {
		fmt.Fprint(os.Stderr, "This replaces the active vault with an empty one; the old files will be preserved as backups. Type RESET to continue: ")
		answer, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("failed to read reset confirmation: %w", readErr)
		}
		if strings.TrimSpace(answer) != "RESET" {
			return fmt.Errorf("force setup cancelled")
		}
	}
	pin, err := readPIN("PIN: ")
	if err != nil {
		return fmt.Errorf("failed to read PIN: %w", err)
	}
	defer clear(pin)
	if create || info.Compatibility == "legacy_migration_required" || info.Compatibility == "pin_format_migration_required" {
		confirmation, err := readPIN("Confirm PIN: ")
		if err != nil {
			clear(confirmation)
			return fmt.Errorf("failed to read PIN confirmation: %w", err)
		}
		matches := bytes.Equal(pin, confirmation)
		clear(confirmation)
		if !matches {
			return fmt.Errorf("PINs do not match")
		}
	}
	req := api.UnlockRequest{PIN: pin, Create: create, Reset: reset}
	reply := api.UnlockReply{}
	if reset {
		fmt.Println("The daemon is preserving the old vault, replacing key-a, and creating a fresh vault…")
	} else if create {
		fmt.Println("The daemon is creating key-a and encrypting the new vault…")
	} else if info.Compatibility == "legacy_migration_required" || info.Compatibility == "pin_format_migration_required" {
		fmt.Println("The daemon is migrating and validating your vault; the original will be backed up…")
	} else {
		fmt.Println("The daemon is deriving the vault key and unlocking…")
	}
	if err := globalClient.Call("VaultServer.Unlock", &req, &reply); err != nil {
		return err
	}
	if !reply.Success {
		return fmt.Errorf("unlock failed: %s", reply.ErrorMsg)
	}
	if reset {
		fmt.Println("A fresh vault has been created and unlocked.")
		if reply.BackupPath != "" {
			fmt.Printf("Previous vault backup: %s (key-a backup, if present: %s.key-a)\n", reply.BackupPath, reply.BackupPath)
		}
	} else if create {
		fmt.Println("Vault initialized and unlocked. A Secret Service key share protects it together with your PIN.")
	} else if info.Compatibility == "legacy_migration_required" || info.Compatibility == "pin_format_migration_required" {
		fmt.Printf("Vault migrated from version %s to the hybrid format and unlocked.\n", info.Version)
	} else {
		fmt.Println("Vault unlocked.")
	}
	return nil
}

var setupCmd = &cobra.Command{Use: "setup", Short: "Create a PIN protected vault", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
	fmt.Println("Thank you for choosing shield!")
	return unlock(true)
}}
var unlockCmd = &cobra.Command{Use: "unlock", Short: "Unlock the vault", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error { return unlock(false) }}

func init() {
	setupCmd.Flags().BoolVar(&setupForce, "force", false, "Replace an existing vault after confirmation, preserving a backup")
	rootCmd.AddCommand(setupCmd, unlockCmd)
}
