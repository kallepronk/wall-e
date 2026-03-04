package discovery

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

func FromGitDiff(rootPath string) (Collect, error) {

	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	return func() ([]File, error) {

		worktree, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %w", err)
		}

		status, err := worktree.Status()
		if err != nil {
			return nil, fmt.Errorf("failed to get status: %w", err)
		}

		var files []File

		for filePath, fileStatus := range status {
			if fileStatus.Staging == git.Deleted || fileStatus.Worktree == git.Deleted {
				continue
			}

			var file File
			file.Path = filePath

			if fileStatus.Worktree == git.Untracked {
				file.Status = StatusUntracked

				content, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
				}
				file.Content = content
				files = append(files, file)
				continue
			}

			if fileStatus.Staging == git.Added {
				file.Status = StatusAdded
			} else if fileStatus.Staging == git.Modified || fileStatus.Worktree == git.Modified {
				file.Status = StatusModified
			} else {
				continue
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			file.Content = content

			diffRanges, err := getAddedLineRanges(repo, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get diff ranges for %s: %w", filePath, err)
			}
			file.DiffRanges = diffRanges

			files = append(files, file)
		}

		return files, nil
	}, nil
}
