package scanner

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

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
