package cmd

import (
	"fmt"
	"os"
	"strings"
	"walle/internal/languages"

	"github.com/spf13/cobra"
)

func buildLongDescription() string {
	langs := languages.GetSupportedLanguageNames()
	return fmt.Sprintf(`WALL-E scans your codebase for comments and compacts them into oblivion.
By default it only trashes new comments.

Supported languages:
  %s`, strings.Join(langs, ", "))
}

var rootCmd = &cobra.Command{
	Use:   "walle",
	Short: "A comment cleaner for your codebase",
	Long:  buildLongDescription(),
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
