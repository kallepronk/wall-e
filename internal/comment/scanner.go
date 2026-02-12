package comment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"walle/internal/languages"
	"walle/internal/source"

	sitter "github.com/smacker/go-tree-sitter"
)

type Scanner interface {
	Scan(file source.File) ([]Comment, error)
}

func GetScanner(filename string) (Scanner, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	lang := languages.GetLanguageForExtension(ext)
	if lang == nil {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	return &TreeSitterScanner{Language: lang}, nil
}

type TreeSitterScanner struct {
	Language *sitter.Language
}

// commentQueries contains different query patterns for various tree-sitter grammars
// Some grammars use "comment", others use "line_comment"/"block_comment"
var commentQueries = []string{
	`(comment) @comment`,
	`(line_comment) @comment`,
	`(block_comment) @comment`,
	`(multiline_comment) @comment`,
}

func (s *TreeSitterScanner) Scan(file source.File) ([]Comment, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(s.Language)

	tree, err := parser.ParseCtx(context.Background(), nil, file.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", file.Path, err)
	}
	defer tree.Close()

	var comments []Comment

	// Try each query pattern and collect all comments
	for _, queryMessage := range commentQueries {
		query, err := sitter.NewQuery([]byte(queryMessage), s.Language)
		if err != nil {
			// This query pattern is not supported by this grammar, skip it
			continue
		}

		queryCursor := sitter.NewQueryCursor()
		queryCursor.Exec(query, tree.RootNode())

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

		queryCursor.Close()
		query.Close()
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
