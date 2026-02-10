package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "walle",
	Short: "A comment cleaner for your codebase",
	Long: `WALL-E scans your codebase for comments and compacts them into oblivion.
			by default it only trashes new comments`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
