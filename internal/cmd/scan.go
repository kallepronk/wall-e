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
	scanAll  bool
	scanPath string
	verbose  bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Find comments without deleting them",
	Run: func(cmd *cobra.Command, args []string) {
		runScan()
	},
}

func runScan() {
	scanOpts := &source.ScanOptions{}

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
		} else {
			scanOpts.SpecificFiles = []string{scanPath}
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
	} else {
		scanOpts.Type = source.ScanDiff
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
}
