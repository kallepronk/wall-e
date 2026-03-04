package discovery

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

func getAddedLineRanges(repo *git.Repository, filePath string) ([]LineRange, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, nil
	}

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get head tree: %w", err)
	}

	headFile, err := headTree.File(filePath)
	if err != nil {
		return nil, nil
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to read head file content: %w", err)
	}

	currentContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read current file: %w", err)
	}

	return calculateAddedRanges(headContent, string(currentContent)), nil
}

func calculateAddedRanges(oldContent, newContent string) []LineRange {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	lcs := computeLCS(oldLines, newLines)

	var ranges []LineRange
	var currentRange *LineRange

	lcsIndex := 0
	for newLineNum, newLine := range newLines {
		lineNum := newLineNum + 1

		if lcsIndex < len(lcs) && newLine == lcs[lcsIndex] {
			if currentRange != nil {
				ranges = append(ranges, *currentRange)
				currentRange = nil
			}
			lcsIndex++
		} else {
			if currentRange == nil {
				currentRange = &LineRange{Start: lineNum, End: lineNum}
			} else {
				currentRange.End = lineNum
			}
		}
	}

	if currentRange != nil {
		ranges = append(ranges, *currentRange)
	}

	return ranges
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}

func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return nil
	}

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	lcs := make([]string, dp[m][n])
	i, j := m, n
	k := len(lcs) - 1
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[k] = a[i-1]
			k--
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}
