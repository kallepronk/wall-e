package discovery

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

func FromGitHistory(rootPath string, baseCommit string, targetCommit string) (Collect, error) {

	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	return func() ([]File, error) {

		err := ValidateCommitOrder(baseCommit, targetCommit)
		if err != nil {
			return nil, err
		}

		var baseTree *object.Tree
		if baseCommit == "" {
			head, err := repo.Head()
			if err != nil {
				return nil, fmt.Errorf("failed to get HEAD: %w", err)
			}
			baseCommitObj, err := repo.CommitObject(head.Hash())
			if err != nil {
				return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
			}
			baseTree, err = baseCommitObj.Tree()
			if err != nil {
				return nil, fmt.Errorf("failed to get base tree: %w", err)
			}
		} else {
			baseHash, err := repo.ResolveRevision(plumbing.Revision(baseCommit))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve base commit %s: %w", baseCommit, err)
			}
			baseCommitObj, err := repo.CommitObject(*baseHash)
			if err != nil {
				return nil, fmt.Errorf("failed to get base commit: %w", err)
			}
			baseTree, err = baseCommitObj.Tree()
			if err != nil {
				return nil, fmt.Errorf("failed to get base tree: %w", err)
			}
		}

		var targetTree *object.Tree
		if targetCommit == "" {
			head, err := repo.Head()
			if err != nil {
				return nil, fmt.Errorf("failed to get HEAD: %w", err)
			}
			targetCommitObj, err := repo.CommitObject(head.Hash())
			if err != nil {
				return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
			}
			targetTree, err = targetCommitObj.Tree()
			if err != nil {
				return nil, fmt.Errorf("failed to get target tree: %w", err)
			}
		} else {
			targetHash, err := repo.ResolveRevision(plumbing.Revision(targetCommit))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve target commit %s: %w", targetCommit, err)
			}
			targetCommitObj, err := repo.CommitObject(*targetHash)
			if err != nil {
				return nil, fmt.Errorf("failed to get target commit: %w", err)
			}
			targetTree, err = targetCommitObj.Tree()
			if err != nil {
				return nil, fmt.Errorf("failed to get target tree: %w", err)
			}
		}

		changes, err := baseTree.Diff(targetTree)
		if err != nil {
			return nil, fmt.Errorf("failed to compute diff: %w", err)
		}

		var files []File

		for _, change := range changes {
			action, err := change.Action()
			if err != nil {
				return nil, fmt.Errorf("failed to get change action: %w", err)
			}

			if action == merkletrie.Delete {
				continue
			}

			_, toFile, err := change.Files()
			if err != nil {
				return nil, fmt.Errorf("failed to get change files: %w", err)
			}

			if toFile == nil {
				continue
			}

			file, err := processTreeFile(toFile, action, baseTree)
			if err != nil {
				return nil, err
			}

			if file != nil {
				files = append(files, *file)
			}
		}

		return files, nil
	}, nil
}

func processTreeFile(toFile *object.File, action merkletrie.Action, baseTree *object.Tree) (*File, error) {

	file := &File{
		Path: toFile.Name,
	}

	switch action {
	case merkletrie.Insert:
		file.Status = StatusAdded
	case merkletrie.Modify:
		file.Status = StatusModified
	default:
		return nil, nil
	}

	content, err := toFile.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", toFile.Name, err)
	}
	file.Content = []byte(content)

	var oldContent string
	if baseTree != nil {
		baseFile, err := baseTree.File(toFile.Name)
		if err == nil {
			oldContent, _ = baseFile.Contents()
		}
	}
	file.DiffRanges = calculateAddedRanges(oldContent, content)

	return file, nil
}

func ValidateCommitOrder(baseCommit, targetCommit string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return errors.New("no git repository found")
	}

	baseHash, err := repo.ResolveRevision(plumbing.Revision(baseCommit))
	if err != nil {
		return fmt.Errorf("failed to resolve base commit %s: %w", baseCommit, err)
	}

	targetHash, err := repo.ResolveRevision(plumbing.Revision(targetCommit))
	if err != nil {
		return fmt.Errorf("failed to resolve target commit %s: %w", targetCommit, err)
	}

	if *baseHash == *targetHash {
		return nil
	}

	targetCommitObj, err := repo.CommitObject(*targetHash)
	if err != nil {
		return fmt.Errorf("failed to get target commit: %w", err)
	}

	baseCommitObj, err := repo.CommitObject(*baseHash)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}

	isAncestor, err := baseCommitObj.IsAncestor(targetCommitObj)
	if err != nil {
		return fmt.Errorf("failed to check commit ancestry: %w", err)
	}

	if !isAncestor {
		return fmt.Errorf("target commit is earlier than base commit - target must be later than or equal to base")
	}

	return nil
}
