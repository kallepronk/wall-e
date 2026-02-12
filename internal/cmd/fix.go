package cmd

import (
	"fmt"
	"os"
	"walle/internal/pipeline"
	"walle/internal/source"

	"github.com/spf13/cobra"
)

var (
	fixAll             bool
	fixPath            string
	fixIgnoreGitIgnore bool
	fixBaseCommit      string
	fixTargetCommit    string
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Trash compact comments",
	Run: func(cmd *cobra.Command, args []string) {
		runFix()
	},
}

func runFix() {
	scanOpts := &source.ScanOptions{
		BaseCommit:   fixBaseCommit,
		TargetCommit: fixTargetCommit,
	}

	if fixPath != "" {
		info, err := os.Stat(fixPath)
		if err != nil {
			fmt.Printf("Error finding path: %v\n", err)
			return
		}

		if info.IsDir() {
			files, err := findAllFiles(fixPath)
			if err != nil {
				fmt.Printf("Error listing files: %v\n", err)
				return
			}
			scanOpts.SpecificFiles = files
			// Respect gitignore when scanning a directory
		} else {
			scanOpts.SpecificFiles = []string{fixPath}
			// Bypass gitignore when scanning a specific file
			scanOpts.IgnoreGitIgnore = true
		}
		scanOpts.Type = source.ScanWhole
	} else if fixAll {
		var err error
		files, err := findAllFiles(".")
		if err != nil {
			fmt.Printf("Error listing files: %v\n", err)
			return
		}
		scanOpts.SpecificFiles = files
		scanOpts.Type = source.ScanWhole
		// Bypass gitignore when using -a flag
		scanOpts.IgnoreGitIgnore = true
	} else {
		scanOpts.Type = source.ScanDiff
		// Default: respect gitignore
	}

	// Override gitignore if flag is set
	if fixIgnoreGitIgnore {
		scanOpts.IgnoreGitIgnore = true
	}

	pipelineOpts := pipeline.Options{
		Verbose: verbose,
	}

	comments, err := pipeline.ScanPipeline(scanOpts, pipelineOpts)
	if err != nil {
		fmt.Printf("Error scanning: %v\n", err)
		return
	}

	if len(comments) == 0 {
		fmt.Println("No comments found.")
		return
	}

	err = pipeline.TrashPipeline(comments)
	if err != nil {
		fmt.Printf("Error in trash pipeline: %v\n", err)
		return
	}
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolVarP(&fixAll, "all", "a", false, "Scan all files in the current directory")
	fixCmd.Flags().StringVarP(&fixPath, "path", "p", "", "Scan a specific file or directory")
	fixCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show comments")
	fixCmd.Flags().BoolVar(&fixIgnoreGitIgnore, "ignore-gitignore", false, "Ignore .gitignore rules")
	fixCmd.Flags().StringVar(&fixBaseCommit, "base", "", "Base commit for comparison")
	fixCmd.Flags().StringVar(&fixTargetCommit, "target", "", "Target commit for comparison")
}
