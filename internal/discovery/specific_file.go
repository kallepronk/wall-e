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
// Reading each file and computing diff ranges are independent operations, so
// all files are processed concurrently.
func FromSpecificFile(rootPath string, filePaths []string, scanWhole bool) (Collect, error) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("%s does not exist", filePath)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory, not a file", filePath)
		}
	}

	return func() ([]File, error) {
		// Fan-out: read + optionally diff each file concurrently.
		files := make([]File, len(filePaths))
		errs := make([]error, len(filePaths))

		var wg sync.WaitGroup
		for i, filePath := range filePaths {
			wg.Add(1)
			go func(i int, filePath string) {
				defer wg.Done()

				content, err := os.ReadFile(filePath)
				if err != nil {
					errs[i] = fmt.Errorf("failed to read file %s: %w", filePath, err)
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
						errs[i] = fmt.Errorf("failed to get diff ranges for %s: %w", filePath, err)
						return
					}
					file.DiffRanges = diffRanges
				}

				files[i] = file
			}(i, filePath)
		}
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return nil, err
			}
		}

		return files, nil
	}, nil
}
