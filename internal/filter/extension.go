package filter

import (
	"path/filepath"
	"strings"
	"walle/internal/discovery"
	"walle/internal/languages"
)

// ExtensionFilter removes files whose extension has no tree-sitter grammar.
// Files without a supported grammar cannot be scanned for comments.
type ExtensionFilter struct{}

func NewExtensionFilter() *ExtensionFilter {
	return &ExtensionFilter{}
}

func (f *ExtensionFilter) Apply(files []discovery.File) []discovery.File {
	kept := files[:0]
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		if languages.IsSupportedExtension(ext) {
			kept = append(kept, file)
		}
	}
	return kept
}
