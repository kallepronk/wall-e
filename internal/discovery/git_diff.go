package discovery

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-git/go-git/v5"
)

// FromGitDiff returns a Collect that discovers files with uncommitted changes
// in the working tree. Each eligible file is read and its diff ranges computed
// concurrently — one goroutine per file — since both file I/O and LCS
// calculation are independent across files.
func FromGitDiff(rootPath string) (Collect, error) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("no git repository found (is this directory in a git repo?): %w", err)
	}

	return func() ([]File, error) {
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %w", err)
		}

		status, err := worktree.Status()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree status: %w", err)
		}

		// Collect candidates before spawning goroutines.
		type candidate struct {
			path       string
			fileStatus FileStatus
		}

		var candidates []candidate
		for filePath, s := range status {
			if s.Staging == git.Deleted || s.Worktree == git.Deleted {
				continue
			}

			switch {
			case s.Worktree == git.Untracked:
				candidates = append(candidates, candidate{filePath, StatusUntracked})
			case s.Staging == git.Added:
				candidates = append(candidates, candidate{filePath, StatusAdded})
			case s.Staging == git.Modified || s.Worktree == git.Modified:
				candidates = append(candidates, candidate{filePath, StatusModified})
			}
		}

		// Fan-out: read + diff each file concurrently. File I/O and LCS
		// computation are both independent across files, so this scales
		// horizontally with the number of changed files.
		files := make([]File, len(candidates))
		errs := make([]error, len(candidates))

		var wg sync.WaitGroup
		for i, c := range candidates {
			wg.Add(1)
			go func(i int, c candidate) {
				defer wg.Done()

				content, err := os.ReadFile(c.path)
				if err != nil {
					errs[i] = fmt.Errorf("failed to read file %s: %w", c.path, err)
					return
				}

				file := File{
					Path:    c.path,
					Content: content,
					Status:  c.fileStatus,
				}

				if c.fileStatus == StatusModified {
					diffRanges, err := getAddedLineRanges(repo, c.path)
					if err != nil {
						errs[i] = fmt.Errorf("failed to get diff ranges for %s: %w", c.path, err)
						return
					}
					file.DiffRanges = diffRanges
				}

				files[i] = file
			}(i, c)
		}
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return nil, err
			}
		}

		// Strip zero-value slots (candidates that errored and wrote nothing).
		var result []File
		for _, f := range files {
			if f.Path != "" {
				result = append(result, f)
			}
		}

		return result, nil
	}, nil
}
