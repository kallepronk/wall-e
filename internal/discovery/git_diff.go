package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-git/go-git/v5"
)

// FromGitDiff returns a Collect that discovers files with uncommitted changes
// in the working tree. Each eligible file is read and its diff ranges computed
// concurrently — one goroutine per file — since both file I/O and LCS
// calculation are independent across files.
//
// git reports paths relative to the repository root, so every path is joined
// with the root before reading. This keeps the scan correct when walle runs
// from a subdirectory of the repo.
//
// Files that cannot be read or diffed are skipped and reported as warnings
// rather than aborting the entire run.
func FromGitDiff(rootPath string) (Collect, error) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("no git repository found (is this directory in a git repo?): %w", err)
	}

	return func() ([]File, []Warning, error) {
		repoRoot, err := getRepoRoot(repo)
		if err != nil {
			return nil, nil, err
		}

		worktree, err := repo.Worktree()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get worktree: %w", err)
		}

		status, err := worktree.Status()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get worktree status: %w", err)
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

			absPath := filepath.Join(repoRoot, filepath.FromSlash(filePath))
			switch {
			case s.Worktree == git.Untracked:
				candidates = append(candidates, candidate{absPath, StatusUntracked})
			case s.Staging == git.Added:
				candidates = append(candidates, candidate{absPath, StatusAdded})
			case s.Staging == git.Modified || s.Worktree == git.Modified:
				candidates = append(candidates, candidate{absPath, StatusModified})
			}
		}

		type result struct {
			file    File
			warning *Warning
		}

		// Fan-out: read + diff each file concurrently. File I/O and LCS
		// computation are both independent across files, so this scales
		// horizontally with the number of changed files.
		results := make([]result, len(candidates))

		var wg sync.WaitGroup
		for i, c := range candidates {
			wg.Add(1)
			go func(i int, c candidate) {
				defer wg.Done()

				content, err := os.ReadFile(c.path)
				if err != nil {
					results[i] = result{warning: &Warning{
						Path:    c.path,
						Message: fmt.Sprintf("skipped: %v", err),
					}}
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
						results[i] = result{warning: &Warning{
							Path:    c.path,
							Message: fmt.Sprintf("skipped: failed to compute diff ranges: %v", err),
						}}
						return
					}
					file.DiffRanges = diffRanges
				}

				results[i] = result{file: file}
			}(i, c)
		}
		wg.Wait()

		var files []File
		var warnings []Warning
		for _, r := range results {
			if r.warning != nil {
				warnings = append(warnings, *r.warning)
				continue
			}
			files = append(files, r.file)
		}

		return files, warnings, nil
	}, nil
}
