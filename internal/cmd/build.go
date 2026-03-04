package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"walle/internal/discovery"
	"walle/internal/filter"
)

// buildCollect maps CLI flags to a Collect strategy and an ordered slice of
// Filters. Both scan and fix call this function — it is the single place where
// flag values are translated into the correct discovery behaviour.
//
// Priority order for strategy selection (first match wins):
//  1. baseCommit set → scan committed history between two revisions.
//  2. explicit paths → scan exactly those files.
//  3. all == true    → scan every file under rootPath.
//  4. default        → scan uncommitted working-tree changes.
//
// Filters are always applied in this order:
//  1. ExtensionFilter  — drop files with no tree-sitter grammar.
//  2. GitIgnoreFilter  — drop files matched by .gitignore (skipped when
//     ignoreGitIgnore is true).
func buildCollect(
	rootPath string,
	paths []string,
	all bool,
	baseCommit string,
	targetCommit string,
	scanWhole bool,
	ignoreGitIgnore bool,
) (discovery.Collect, []filter.Filter, error) {
	var collect discovery.Collect
	var err error

	switch {
	case baseCommit != "":
		collect, err = discovery.FromGitHistory(rootPath, baseCommit, targetCommit)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to set up history scan: %w", err)
		}

	case len(paths) > 0:
		collect, err = discovery.FromSpecificFile(rootPath, paths, scanWhole)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to set up file scan: %w", err)
		}

	case all:
		allFiles, err := findAllFiles(rootPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list files: %w", err)
		}
		collect, err = discovery.FromSpecificFile(rootPath, allFiles, true)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to set up full scan: %w", err)
		}

	default:
		collect, err = discovery.FromGitDiff(rootPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to set up diff scan: %w", err)
		}
	}

	filters := []filter.Filter{
		filter.NewExtensionFilter(),
	}
	if !ignoreGitIgnore {
		filters = append(filters, filter.NewGitIgnoreFilter(rootPath))
	}

	return collect, filters, nil
}

// findAllFiles walks root and returns paths for every non-directory entry.
func findAllFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
