package cmd

import (
	"fmt"
	"os"
	"sync"
	"walle/internal/git"
	"walle/internal/remover"
	"walle/internal/scanner"

	"github.com/spf13/cobra"
)

var (
	fixAll      bool
	fixPath     string
	interactive bool
	force       bool
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Trash compact comments",
	Run: func(cmd *cobra.Command, args []string) {
		runFix()
	},
}

func runFix() {
	tasks := make(map[string][]scanner.Comment)

	var files []string
	var err error

	if fixPath != "" {
		info, statErr := os.Stat(fixPath)
		if statErr != nil {
			fmt.Printf("Error finding path: %v\n", statErr)
			return
		}

		if info.IsDir() {
			files, err = findAllFiles(fixPath)
		} else {
			files = []string{fixPath}
		}
	} else if fixAll {
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

	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			commentScanner, err := scanner.GetScanner(file)
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

			if len(comments) == 0 {
				return
			}

			mu.Lock()
			tasks[file] = comments
			mu.Unlock()
		}(file)
	}
	wg.Wait()

	if len(tasks) == 0 {
		fmt.Println("No comments found.")
		return
	}

	totalComments := 0
	for _, comments := range tasks {
		totalComments += len(comments)
	}
	fmt.Printf("Found %d comments in %d files.\n", totalComments, len(tasks))

	if interactive {
		var cancelled bool
		tasks, cancelled = RunTUI(tasks)
		if cancelled {
			fmt.Println("Operation cancelled.")
			return
		}
		if len(tasks) == 0 {
			fmt.Println("No comments selected for removal.")
			return
		}
	} else if !force {
		fmt.Printf("Prepare to delete %d comments in %d files. Continue? (y/n): ", totalComments, len(tasks))
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("Operation cancelled.")
			return
		}
	}

	removedCount := 0
	for file, comments := range tasks {
		err := remover.RemoveComments(file, comments)
		if err != nil {
			fmt.Printf("⚠️  Error deleting comments in %s: %v\n", file, err)
		} else {
			fmt.Printf("✅ Removed %d comments from %s\n", len(comments), file)
			removedCount += len(comments)
		}
	}

	fmt.Printf("\n🗑️  Trash compacted %d comments total.\n", removedCount)
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolVarP(&fixAll, "all", "a", false, "Scan all files in the current directory")
	fixCmd.Flags().StringVarP(&fixPath, "path", "p", "", "Scan a specific file or directory")
	fixCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Run in interactive mode")
	fixCmd.Flags().BoolVarP(&force, "force", "f", false, "Run without confirmation")
}
