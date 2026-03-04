package discovery

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// FromGitHistory returns a Collect that discovers files changed between two
// commits. baseCommit must be an ancestor of targetCommit. Both are resolved
// as git revisions (SHA, tag, branch name, etc.).
func FromGitHistory(rootPath, baseCommit, targetCommit string) (Collect, error) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no git repository found (is this directory in a git repo?)")
	}

	return func() ([]File, []Warning, error) {
		if err := ValidateCommitOrder(repo, baseCommit, targetCommit); err != nil {
			return nil, nil, err
		}

		baseTree, err := resolveTree(repo, baseCommit)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve base tree: %w", err)
		}

		targetTree, err := resolveTree(repo, targetCommit)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve target tree: %w", err)
		}

		changes, err := baseTree.Diff(targetTree)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compute diff: %w", err)
		}

		var files []File
		for _, change := range changes {
			action, err := change.Action()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get change action: %w", err)
			}

			if action == merkletrie.Delete {
				continue
			}

			_, toFile, err := change.Files()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get change files: %w", err)
			}

			if toFile == nil {
				continue
			}

			file, err := fileFromTreeChange(toFile, action, baseTree)
			if err != nil {
				return nil, nil, err
			}

			if file != nil {
				files = append(files, *file)
			}
		}

		return files, nil, nil
	}, nil
}

// fileFromTreeChange builds a File from a tree-sitter change entry.
// It always computes diff ranges (StatusAdded files get an empty base).
func fileFromTreeChange(toFile *object.File, action merkletrie.Action, baseTree *object.Tree) (*File, error) {
	file := &File{Path: toFile.Name}

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
		if baseFile, err := baseTree.File(toFile.Name); err == nil {
			oldContent, _ = baseFile.Contents()
		}
	}
	file.DiffRanges = calculateAddedRanges(oldContent, content)

	return file, nil
}

// resolveTree returns the git tree for a given revision string.
// If revision is empty it falls back to HEAD.
func resolveTree(repo *git.Repository, revision string) (*object.Tree, error) {
	var hash *plumbing.Hash

	if revision == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		h := head.Hash()
		hash = &h
	} else {
		h, err := repo.ResolveRevision(plumbing.Revision(revision))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve revision %s: %w", revision, err)
		}
		hash = h
	}

	commitObj, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	tree, err := commitObj.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	return tree, nil
}

// ValidateCommitOrder checks that baseCommit is an ancestor of targetCommit.
// An opened *git.Repository is passed in to avoid re-opening from disk.
func ValidateCommitOrder(repo *git.Repository, baseCommit, targetCommit string) error {
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
		return fmt.Errorf("target commit %s is not a descendant of base commit %s", targetCommit, baseCommit)
	}

	return nil
}
