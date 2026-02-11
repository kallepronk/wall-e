package remover

import (
	"fmt"
	"os"
	"sort"
	"unicode"
	"walle/internal/scanner"
)

func RemoveComments(filePath string, comments []scanner.Comment) error {
	if len(comments) == 0 {
		return nil
	}

	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].StartByte > comments[j].StartByte
	})

	output := input
	for _, comment := range comments {
		if isWholeLineComment(output, comment.StartByte) {
			newStart := comment.StartByte
			for newStart > 0 && output[newStart-1] != '\n' {
				newStart--
			}

			if comment.EndByte < uint32(len(output)) && output[comment.EndByte] == '\n' {
				comment.EndByte++
			}
			comment.StartByte = newStart
		}
		output = append(output[:comment.StartByte], output[comment.EndByte:]...)

	}

	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}
	if err := os.Rename(tmpFile, filePath); err != nil {
		return fmt.Errorf("failed to rename tmp file: %w", err)
	}

	return nil
}

func isWholeLineComment(content []byte, startPos uint32) bool {
	for i := int(startPos) - 1; i >= 0; i-- {
		b := content[i]
		if b == '\n' {
			return true
		}
		if !unicode.IsSpace(rune(b)) {
			return false
		}
	}
	return true
}
