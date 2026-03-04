package cmd

import (
	"fmt"
	"os"
	"walle/internal/pipeline"

	"github.com/spf13/cobra"
)

var (
	scanAll             bool
	scanIgnoreGitIgnore bool
	scanBaseCommit      string
	scanTargetCommit    string
	scanVerbose         bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [file...]",
	Short: "Find comments without modifying files",
	Long: `Scan reports all comments found in the target files.

By default, only files with uncommitted working-tree changes are scanned and
only the changed lines within those files are inspected. Pass explicit file
paths as arguments, --all, or --base/--target to change which files are in scope.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runScan(args)
	},
}

func runScan(paths []string) {
	rootPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		return
	}

	collect, filters, err := buildCollect(
		rootPath,
		paths,
		scanAll,
		scanBaseCommit,
		scanTargetCommit,
		false, // scan uses diff ranges, not whole-file
		scanIgnoreGitIgnore,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	comments, err := pipeline.ScanPipeline(pipeline.RunConfig{
		Collect: collect,
		Filters: filters,
	})
	if err != nil {
		fmt.Printf("Error scanning: %v\n", err)
		return
	}

	if len(comments) == 0 {
		fmt.Println("No comments found.")
		return
	}

	// Group comments by file, preserving order of first appearance.
	fileOrder, byFile := printComments(comments, scanVerbose)

	fmt.Printf("\nTotal: %d comment(s) in %d file(s)\n", len(comments), len(fileOrder))
	_ = byFile
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVarP(&scanAll, "all", "a", false, "Scan all files under the current directory")
	scanCmd.Flags().BoolVarP(&scanVerbose, "verbose", "v", false, "Print each comment")
	scanCmd.Flags().BoolVar(&scanIgnoreGitIgnore, "ignore-gitignore", false, "Do not apply .gitignore rules")
	scanCmd.Flags().StringVar(&scanBaseCommit, "base", "", "Base commit (enables history scan)")
	scanCmd.Flags().StringVar(&scanTargetCommit, "target", "", "Target commit (defaults to HEAD when --base is set)")
}
