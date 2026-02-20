package cmd

import (
	"fmt"
	"shield/pkg/crypto"

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
		if err != nil {
			color.HiRed("Failed to generate master key...")
			color.HiWhite("Shield only supports GUI environments")
		}

		color.HiGreen("Master key generated!")
		fmt.Printf("=> %s\n", key)
	},
}
