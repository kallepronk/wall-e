package pipeline

import (
	"fmt"
	"strings"
	"sync"
	"walle/internal/comment"
	"walle/internal/source"
)

func ScanPipeline(scanOpts *source.ScanOptions, pipeOpts Options) ([]comment.Comment, error) {
	gitScanner := &source.GitScanner{}

	files, err := gitScanner.GetFiles(*scanOpts)
	if err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	var totalComments []comment.Comment
	filesWithComments := 0
	for _, file := range files {
		wg.Add(1)

		go func(file source.File) {
			defer wg.Done()
			commentScanner, err := comment.GetScanner(file.Path)
			if err != nil {
				return
			}

			comments, err := commentScanner.Scan(file)
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
			totalComments = append(totalComments, comments...)
			filesWithComments++
			fmt.Printf("Found %d comments in %s\n", count, file.Path)
			if pipeOpts.Verbose {
				for _, c := range comments {
					fmt.Printf("\t- Line %d: %s\n", c.Line, strings.ReplaceAll(strings.ReplaceAll(c.Text, "\n", " "), "\r", " "))
				}
			}
			mu.Unlock()
		}(file)
	}
	wg.Wait()
	fmt.Printf("Found %d comments in %d files\n", len(totalComments), filesWithComments)
	return totalComments, nil
}

func TrashPipeline(comments []comment.Comment) error {

	tasks := make(map[string][]comment.Comment)
	for _, cmt := range comments {
		tasks[cmt.FilePath] = append(tasks[cmt.FilePath], cmt)
	}

	removedCount := 0
	for file, comments := range tasks {
		err := comment.RemoveComments(file, comments)
		if err != nil {
			fmt.Printf("⚠️  Error deleting comments in %s: %v\n", file, err)
		} else {
			fmt.Printf("✅ Removed %d comments from %s\n", len(comments), file)
			removedCount += len(comments)
		}
	}

	fmt.Printf("\n🗑️  Trash compacted %d comments total.\n", removedCount)
	return nil
}
