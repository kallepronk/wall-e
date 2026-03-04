package filter

import (
	"os"
	"path/filepath"
	"walle/internal/discovery"

	gitignore "github.com/sabhiram/go-gitignore"
)

// GitIgnoreFilter removes files that match .gitignore rules at the given root.
// If no .gitignore file exists the filter passes all files through unchanged.
type GitIgnoreFilter struct {
	rootPath string
}

func NewGitIgnoreFilter(rootPath string) *GitIgnoreFilter {
	return &GitIgnoreFilter{rootPath: rootPath}
}

func (f *GitIgnoreFilter) Apply(files []discovery.File) []discovery.File {
	gi := loadGitIgnore(f.rootPath)
	if gi == nil {
		return files
	}

	kept := files[:0]
	for _, file := range files {
		absPath, err := filepath.Abs(file.Path)
		if err != nil {
			kept = append(kept, file)
			continue
		}
		relPath, err := filepath.Rel(f.rootPath, absPath)
		if err != nil || gi.MatchesPath(relPath) {
			continue
		}
		kept = append(kept, file)
	}
	return kept
}

func loadGitIgnore(rootPath string) *gitignore.GitIgnore {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		return nil
	}
	gi, err := gitignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil
	}
	return gi
}
