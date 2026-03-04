package pipeline

import (
	"fmt"
	"sync"
	"walle/internal/comment"
	"walle/internal/discovery"
)

// ScanPipeline runs the full scan:
//  1. Collect files using the configured strategy.
//  2. Apply each filter in order.
//  3. Scan each surviving file for comments concurrently.
//
// It returns the full list of found comments. Printing and formatting are the
// responsibility of the caller (cmd layer).
func ScanPipeline(cfg RunConfig) ([]comment.Comment, error) {
	files, err := cfg.Collect()
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %w", err)
	}

	for _, f := range cfg.Filters {
		files = f.Apply(files)
	}

	return scanFiles(files)
}

// scanFiles fans out comment scanning across all files concurrently.
// Each file is parsed and scanned by an independent goroutine.
func scanFiles(files []discovery.File) ([]comment.Comment, error) {
	type result struct {
		comments []comment.Comment
		err      error
	}

	results := make([]result, len(files))

	var wg sync.WaitGroup
	for i, file := range files {
		wg.Add(1)
		go func(i int, file discovery.File) {
			defer wg.Done()
			scanner, err := comment.GetScanner(file.Path)
			if err != nil {
				// Unsupported file type slipped through the extension filter;
				// treat as a non-fatal skip.
				return
			}
			comments, err := scanner.Scan(file)
			if err != nil {
				results[i] = result{err: fmt.Errorf("failed to scan %s: %w", file.Path, err)}
				return
			}
			results[i] = result{comments: comments}
		}(i, file)
	}
	wg.Wait()

	var all []comment.Comment
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.comments...)
	}

	return all, nil
}

// TrashPipeline removes the given comments from disk. Files are processed
// concurrently since each file's edits are independent.
//
// All per-file errors are collected and returned as a combined error so the
// caller can report exactly which files failed.
func TrashPipeline(comments []comment.Comment) error {
	tasks := groupByFile(comments)

	type result struct {
		file string
		err  error
	}

	results := make(chan result, len(tasks))

	var wg sync.WaitGroup
	for file, fileComments := range tasks {
		wg.Add(1)
		go func(file string, fileComments []comment.Comment) {
			defer wg.Done()
			err := comment.RemoveComments(file, fileComments)
			results <- result{file: file, err: err}
		}(file, fileComments)
	}
	wg.Wait()
	close(results)

	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("failed to remove comments from %s: %w", r.file, r.err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	// Combine all errors into one so the caller has full visibility.
	combined := errs[0]
	for _, e := range errs[1:] {
		combined = fmt.Errorf("%w; %w", combined, e)
	}
	return combined
}

// groupByFile partitions a flat comment list into a map keyed by file path.
func groupByFile(comments []comment.Comment) map[string][]comment.Comment {
	tasks := make(map[string][]comment.Comment)
	for _, c := range comments {
		tasks[c.FilePath] = append(tasks[c.FilePath], c)
	}
	return tasks
}
