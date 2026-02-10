package git

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

func GetChangedFiles() ([]string, error) {

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})

	if err != nil {
		return nil, errors.New("no git repository found (are you in a git dir?)")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var changedFiles []string

	for filePath, fileStatus := range status {

		// Skip deleted files
		if fileStatus.Staging == git.Deleted || fileStatus.Worktree == git.Deleted {
			continue
		}

		hasChanges := false

		if fileStatus.Staging == git.Modified || fileStatus.Staging == git.Added {
			hasChanges = true
		}

		if fileStatus.Worktree == git.Modified || fileStatus.Worktree == git.Untracked {
			hasChanges = true
		}

		if hasChanges {
			changedFiles = append(changedFiles, filePath)
		}

	}
	return changedFiles, nil
}
