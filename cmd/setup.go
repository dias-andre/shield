package cmd

import (
	"errors"
	"fmt"
	"github.com/dias-andre/shield/pkg/crypto"

	"github.com/fatih/color"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use: "setup",
	Short: "Initialize shield",
	Run: func(cmd *cobra.Command, args []string) {
		color.HiGreen("Thank you for choose shield!\n\n")

		fmt.Println("Generating master key...")
		key, err := crypto.GenerateAndStoreMasterKey()
		if errors.Is(err, crypto.ErrMasterKeyAlreadyExists) {
			fmt.Println("A master key already exists!")
			return
		}

		color.HiGreen("Master key generated!")
		fmt.Printf("=> %s\n", key)
	},
}
