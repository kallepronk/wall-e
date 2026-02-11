package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"walle/internal/comment"
	"walle/internal/git"

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
	var files []string
	var err error

	if scanPath != "" {
		info, err := os.Stat(scanPath)
		if err != nil {
			fmt.Printf("Error finding path: %v\n", err)
			return
		}

		if info.IsDir() {
			files, err = findAllFiles(scanPath)
		} else {
			files = []string{scanPath}
		}
	} else if scanAll {
		files, err = findAllFiles(".")
	} else {
		files, err = git.GetChangedFiles()
	}

	if err != nil {
		fmt.Printf("Error listing files: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No files to scan.")
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	totalComments := 0
	filesWithComments := 0
	for _, file := range files {
		wg.Add(1)

		go func(file string) {
			defer wg.Done()
			commentScanner, err := comment.GetScanner(file)
			if err != nil {
				return
			}

			content, err := os.ReadFile(file)
			if err != nil {
				mu.Lock()
				fmt.Printf("⚠️  Skipping %s: %v\n", file, err)
				mu.Unlock()
				return
			}

			comments, err := commentScanner.Scan(content)
			if err != nil {
				mu.Lock()
				fmt.Printf("⚠️  Parse error scanning %s: %v\n", file, err)
				mu.Unlock()
				return
			}

			count := len(comments)
			if count == 0 {
				return
			}

			mu.Lock()
			totalComments += count
			filesWithComments++
			fmt.Printf("Found %d comments in %s\n", count, file)
			if verbose {
				for _, c := range comments {
					fmt.Printf("\t- Line %d: %s\n", c.Line, strings.ReplaceAll(strings.ReplaceAll(c.Text, "\n", " "), "\r", " "))
				}
			}
			mu.Unlock()
		}(file)
	}
	wg.Wait()
	fmt.Printf("Found %d comments in %d files\n", totalComments, filesWithComments)
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
