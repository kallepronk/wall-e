package scanner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func GetScanner(filename string) (LanguageScanner, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".py":
		return &TreeSitterScanner{Language: python.GetLanguage()}, nil
	case ".ts":
		return &TreeSitterScanner{Language: typescript.GetLanguage()}, nil
	case ".tsx":
		return &TreeSitterScanner{Language: typescript.GetLanguage()}, nil
	case ".go":
		return &TreeSitterScanner{Language: golang.GetLanguage()}, nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

}
