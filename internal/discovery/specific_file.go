package discovery

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

func FromSpecificFile(rootPath string, filePaths []string, scanWhole bool) (Collect, error) {

	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	// throw error if files do not exist or are not files eg directory
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		os.IsNotExist(err)
		if err != nil {
			return nil, fmt.Errorf("%s does not exist", filePath)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory", filePath)
		}
	}

	return func() ([]File, error) {

		var files []File

		for _, filePath := range filePaths {

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
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
					return nil, fmt.Errorf("failed to get diff ranges for %s: %w", filePath, err)
				}
				file.DiffRanges = diffRanges
			}
			files = append(files, file)
		}

		return files, nil
	}, nil
}
