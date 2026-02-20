package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "shield",
	Short: "Tool for managing encrypted server keys",
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

