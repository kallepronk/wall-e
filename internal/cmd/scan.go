package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"walle/internal/pipeline"
	"walle/internal/source"

	"github.com/spf13/cobra"
)

var (
	scanAll             bool
	scanPath            string
	verbose             bool
	scanIgnoreGitIgnore bool
	scanBaseCommit      string
	scanTargetCommit    string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Find comments without deleting them",
	Run: func(cmd *cobra.Command, args []string) {
		runScan()
	},
}

func runScan() {
	// Validate target commit is not earlier than base commit
	if scanBaseCommit != "" && scanTargetCommit != "" {
		if err := source.ValidateCommitOrder(scanBaseCommit, scanTargetCommit); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	scanOpts := &source.ScanOptions{
		BaseCommit:   scanBaseCommit,
		TargetCommit: scanTargetCommit,
	}

	if scanPath != "" {
		info, err := os.Stat(scanPath)
		if err != nil {
			fmt.Printf("Error finding path: %v\n", err)
			return
		}

		if info.IsDir() {
			files, err := findAllFiles(scanPath)
			if err != nil {
				fmt.Printf("Error listing files: %v\n", err)
				return
			}
			scanOpts.SpecificFiles = files
			// Respect gitignore when scanning a directory
		} else {
			scanOpts.SpecificFiles = []string{scanPath}
			// Bypass gitignore when scanning a specific file
			scanOpts.IgnoreGitIgnore = true
		}
		scanOpts.Type = source.ScanWhole
	} else if scanAll {
		var err error
		files, err := findAllFiles(".")
		if err != nil {
			fmt.Printf("Error listing files: %v\n", err)
			return
		}
		scanOpts.SpecificFiles = files
		scanOpts.Type = source.ScanWhole
		// Respect gitignore by default when using -a flag
	} else {
		scanOpts.Type = source.ScanDiff
		// Default: respect gitignore
	}

	// Override gitignore if flag is set
	if scanIgnoreGitIgnore {
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
	}
}

func findAllFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVarP(&scanAll, "all", "a", false, "Scan all files")
	scanCmd.Flags().StringVarP(&scanPath, "path", "p", "", "Scan a specific file")
	scanCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show comments")
	scanCmd.Flags().BoolVar(&scanIgnoreGitIgnore, "ignore-gitignore", false, "Ignore .gitignore rules")
	scanCmd.Flags().StringVar(&scanBaseCommit, "base", "", "Base commit for comparison")
	scanCmd.Flags().StringVar(&scanTargetCommit, "target", "", "Target commit for comparison")
}
