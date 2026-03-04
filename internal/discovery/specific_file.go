package discovery

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/go-git/go-git/v5"
)

// FromSpecificFile returns a Collect that reads a fixed set of file paths.
// When scanWhole is true every line is in scope (StatusAdded); otherwise only
// lines that differ from HEAD are in scope (StatusModified + DiffRanges).
//
// Paths that cannot be read (missing, symlinks pointing to directories, etc.)
// are skipped and reported as warnings rather than aborting the entire run.
//
// Reading each file and computing diff ranges are independent operations, so
// all files are processed concurrently.
func FromSpecificFile(rootPath string, filePaths []string, scanWhole bool) (Collect, error) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	return func() ([]File, []Warning, error) {
		type result struct {
			file    File
			warning *Warning
			err     error
			ok      bool
		}

		results := make([]result, len(filePaths))

		var wg sync.WaitGroup
		for i, filePath := range filePaths {
			wg.Add(1)
			go func(i int, filePath string) {
				defer wg.Done()

				// Resolve symlinks so we can detect symlinks-to-directories.
				resolved, err := os.Stat(filePath)
				if err != nil {
					results[i] = result{warning: &Warning{
						Path:    filePath,
						Message: fmt.Sprintf("skipped: %v", err),
					}}
					return
				}
				if resolved.IsDir() {
					results[i] = result{warning: &Warning{
						Path:    filePath,
						Message: "skipped: path is a directory, not a file",
					}}
					return
				}
				if !resolved.Mode().IsRegular() {
					results[i] = result{warning: &Warning{
						Path:    filePath,
						Message: fmt.Sprintf("skipped: not a regular file (mode %s)", resolved.Mode()),
					}}
					return
				}

				content, err := os.ReadFile(filePath)
				if err != nil {
					results[i] = result{warning: &Warning{
						Path:    filePath,
						Message: fmt.Sprintf("skipped: %v", err),
					}}
					return
				}

				file := File{
					Path:    filePath,
					Content: content,
				}

				if scanWhole {
					file.Status = StatusAdded
				} else {
					file.Status = StatusModified
					diffRanges, err := getAddedLineRanges(repo, filePath)
					if err != nil {
						results[i] = result{warning: &Warning{
							Path:    filePath,
							Message: fmt.Sprintf("skipped: failed to compute diff ranges: %v", err),
						}}
						return
					}
					file.DiffRanges = diffRanges
				}

				results[i] = result{file: file, ok: true}
			}(i, filePath)
		}
		wg.Wait()

		var files []File
		var warnings []Warning
		for _, r := range results {
			if r.err != nil {
				return nil, nil, r.err
			}
			if r.warning != nil {
				warnings = append(warnings, *r.warning)
				continue
			}
			if r.ok {
				files = append(files, r.file)
			}
		}

		return files, warnings, nil
	}, nil
}
