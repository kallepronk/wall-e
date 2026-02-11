package comment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type Scanner interface {
	Scan(content []byte) ([]Comment, error)
}

func GetScanner(filename string) (Scanner, error) {
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

type TreeSitterScanner struct {
	Language *sitter.Language
}

func (s *TreeSitterScanner) Scan(content []byte) ([]Comment, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(s.Language)

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("invalid python query: %w", err)
	}
	defer tree.Close()

	queryMessage := `(comment) @comment`
	query, err := sitter.NewQuery([]byte(queryMessage), s.Language)
	if err != nil {
		return nil, fmt.Errorf("invalid python query: %w", err)
	}
	defer query.Close()

	queryCursor := sitter.NewQueryCursor()
	defer queryCursor.Close()
	queryCursor.Exec(query, tree.RootNode())

	var comments []Comment
	for {
		match, ok := queryCursor.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			node := capture.Node

			text := node.Content(content)

			comments = append(comments, Comment{
				Text:      text,
				Line:      int(node.StartPoint().Row),
				StartByte: node.StartByte(),
				EndByte:   node.EndByte(),
			})
		}

	}
	return comments, nil
}
