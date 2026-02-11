package comment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"walle/internal/source"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type Scanner interface {
	Scan(file source.File) ([]Comment, error)
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

func (s *TreeSitterScanner) Scan(file source.File) ([]Comment, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(s.Language)

	queryMessage := `(comment) @comment`
	query, err := sitter.NewQuery([]byte(queryMessage), s.Language)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}
	defer query.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", file.Path, err)
	}
	defer tree.Close()

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
			line := int(node.StartPoint().Row) + 1

			if file.Status == source.StatusAdded || file.Status == source.StatusUntracked {
				comments = append(comments, Comment{
					FilePath:  file.Path,
					Text:      node.Content(file.Content),
					Line:      line,
					StartByte: node.StartByte(),
					EndByte:   node.EndByte(),
				})
			} else if isLineInDiffRanges(line, file.DiffRanges) {
				comments = append(comments, Comment{
					FilePath:  file.Path,
					Text:      node.Content(file.Content),
					Line:      line,
					StartByte: node.StartByte(),
					EndByte:   node.EndByte(),
				})
			}
		}
	}

	return comments, nil
}

func isLineInDiffRanges(line int, ranges []LineRange) bool {
	for _, r := range ranges {
		if line >= r.Start && line <= r.End {
			return true
		}
	}
	return false
}

type LineRange = source.LineRange
