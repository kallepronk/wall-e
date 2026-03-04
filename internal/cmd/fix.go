package cmd

import (
	"fmt"
	"os"
	"walle/internal/pipeline"

	"github.com/spf13/cobra"
)

var (
	fixAll             bool
	fixIgnoreGitIgnore bool
	fixBaseCommit      string
	fixVerbose         bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [file...]",
	Short: "Remove comments from files",
	Long: `Fix scans for comments and removes them from disk.

By default, only files with uncommitted working-tree changes are scanned and
only comments on changed lines are removed. Pass explicit file paths as
arguments, --all, or --base to change which files are in scope.

When explicit file paths are given every comment in those files is removed,
not just comments on changed lines.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runFix(args)
	},
}

func runFix(paths []string) {
	rootPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		return
	}

	// When the caller names specific files we remove ALL comments in them
	// (scanWhole = true). For every other mode we respect diff ranges.
	scanWhole := len(paths) > 0 || fixAll

	collect, filters, err := buildCollect(
		rootPath,
		paths,
		fixAll,
		fixBaseCommit,
		"", // fix always targets HEAD
		scanWhole,
		fixIgnoreGitIgnore,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	result, err := pipeline.ScanPipeline(pipeline.RunConfig{
		Collect: collect,
		Filters: filters,
	})
	if err != nil {
		fmt.Printf("Error scanning: %v\n", err)
		return
	}

	for _, w := range result.Warnings {
		fmt.Printf("Warning: %s\n", w)
	}

	if len(result.Comments) == 0 {
		fmt.Println("No comments found.")
		return
	}

	// Report what was found before removal.
	fileOrder, _ := printComments(result.Comments, fixVerbose)

	if err := pipeline.TrashPipeline(result.Comments); err != nil {
		fmt.Printf("Error removing comments: %v\n", err)
		return
	}

	fmt.Printf("\nRemoved %d comment(s) from %d file(s)\n", len(result.Comments), len(fileOrder))
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolVarP(&fixAll, "all", "a", false, "Fix all files under the current directory")
	fixCmd.Flags().StringVarP(&fixBaseCommit, "base", "b", "", "Base commit (enables history scan; target is always HEAD)")
	fixCmd.Flags().BoolVarP(&fixVerbose, "verbose", "v", false, "Print each comment before removing it")
	fixCmd.Flags().BoolVar(&fixIgnoreGitIgnore, "ignore-gitignore", false, "Do not apply .gitignore rules")
}
