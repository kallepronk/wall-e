package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// supportedExtensions contains file extensions that can be parsed for comments
var supportedExtensions = map[string]bool{
	".py":  true,
	".ts":  true,
	".tsx": true,
	".go":  true,
}

// isSupportedFile checks if a file has a supported extension
func isSupportedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return supportedExtensions[ext]
}

type Scanner interface {
	GetFiles(opts ScanOptions) ([]File, error)
}

type GitScanner struct{}

func (g *GitScanner) GetFiles(opts ScanOptions) ([]File, error) {
	if len(opts.SpecificFiles) > 0 {
		return g.getSpecificFiles(opts.SpecificFiles, opts.Type)
	}

	if opts.BaseCommit != "" || opts.TargetCommit != "" {
		return g.getCommitDiff(opts.BaseCommit, opts.TargetCommit, opts.Type)
	}

	return g.getWorkingTreeChanges(opts)
}

func (g *GitScanner) getSpecificFiles(filePaths []string, scanType ScanType) ([]File, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no source repository found (are you in a source dir?)")
	}

	var files []File

	for _, filePath := range filePaths {
		// Skip unsupported file types early to avoid expensive operations
		if !isSupportedFile(filePath) {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		file := File{
			Path:    filePath,
			Content: content,
		}

		// For ScanWhole, treat file as added so all comments are found
		// For ScanDiff, calculate the diff ranges (only added lines)
		if scanType == ScanWhole {
			file.Status = StatusAdded
		} else {
			file.Status = StatusModified
			diffRanges, err := g.getAddedLineRanges(repo, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get diff ranges for %s: %w", filePath, err)
			}
			file.DiffRanges = diffRanges
		}

		files = append(files, file)
	}

	return files, nil
}

func (g *GitScanner) getCommitDiff(baseCommit string, targetCommit string, scanType ScanType) ([]File, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no source repository found (are you in a source dir?)")
	}

	// Resolve base commit (defaults to HEAD if empty)
	var baseTree *object.Tree
	if baseCommit == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		baseCommitObj, err := repo.CommitObject(head.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
		}
		baseTree, err = baseCommitObj.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get base tree: %w", err)
		}
	} else {
		baseHash, err := repo.ResolveRevision(plumbing.Revision(baseCommit))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base commit %s: %w", baseCommit, err)
		}
		baseCommitObj, err := repo.CommitObject(*baseHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get base commit: %w", err)
		}
		baseTree, err = baseCommitObj.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get base tree: %w", err)
		}
	}

	// Resolve target commit (defaults to HEAD if empty)
	var targetTree *object.Tree
	if targetCommit == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		targetCommitObj, err := repo.CommitObject(head.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
		}
		targetTree, err = targetCommitObj.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get target tree: %w", err)
		}
	} else {
		targetHash, err := repo.ResolveRevision(plumbing.Revision(targetCommit))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target commit %s: %w", targetCommit, err)
		}
		targetCommitObj, err := repo.CommitObject(*targetHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get target commit: %w", err)
		}
		targetTree, err = targetCommitObj.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get target tree: %w", err)
		}
	}

	// Get changes between commits
	changes, err := baseTree.Diff(targetTree)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	var files []File

	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return nil, fmt.Errorf("failed to get change action: %w", err)
		}

		// Skip deleted files - only interested in added code
		if action == merkletrie.Delete {
			continue
		}

		_, toFile, err := change.Files()
		if err != nil {
			return nil, fmt.Errorf("failed to get change files: %w", err)
		}

		if toFile == nil {
			continue
		}

		file, err := g.processTreeFile(toFile, action, baseTree, scanType)
		if err != nil {
			return nil, err
		}

		if file != nil {
			files = append(files, *file)
		}
	}

	return files, nil
}

func (g *GitScanner) processTreeFile(toFile *object.File, action merkletrie.Action, baseTree *object.Tree, scanType ScanType) (*File, error) {
	// Skip unsupported file types early to avoid expensive operations
	if !isSupportedFile(toFile.Name) {
		return nil, nil
	}

	file := &File{
		Path: toFile.Name,
	}

	// Set status based on action
	switch action {
	case merkletrie.Insert:
		file.Status = StatusAdded
	case merkletrie.Modify:
		file.Status = StatusModified
	default:
		return nil, nil
	}

	// Get file content
	content, err := toFile.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", toFile.Name, err)
	}
	file.Content = []byte(content)

	// For ScanDiff, calculate the diff ranges (only added lines)
	if scanType == ScanDiff {
		var oldContent string
		if baseTree != nil {
			baseFile, err := baseTree.File(toFile.Name)
			if err == nil {
				oldContent, _ = baseFile.Contents()
			}
		}
		file.DiffRanges = calculateAddedRanges(oldContent, content)
	}

	return file, nil
}

func (g *GitScanner) getWorkingTreeChanges(opts ScanOptions) ([]File, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no source repository found (are you in a source dir?)")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var files []File

	for filePath, fileStatus := range status {
		// Skip unsupported file types early to avoid expensive operations
		if !isSupportedFile(filePath) {
			continue
		}

		// Skip deleted files
		if fileStatus.Staging == git.Deleted || fileStatus.Worktree == git.Deleted {
			continue
		}

		var file File
		file.Path = filePath

		// Handle untracked files
		if fileStatus.Worktree == git.Untracked {
			if !opts.IncludeUntracked {
				continue
			}
			file.Status = StatusUntracked

			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			file.Content = content
			files = append(files, file)
			continue
		}

		// Handle modified files
		// Check Staging.Added first (takes precedence over Worktree.Modified for newly added files)
		if fileStatus.Staging == git.Added {
			file.Status = StatusAdded
		} else if fileStatus.Staging == git.Modified || fileStatus.Worktree == git.Modified {
			file.Status = StatusModified
		} else {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		file.Content = content

		// For ScanDiff, calculate the diff ranges (only added lines)
		if opts.Type == ScanDiff {
			diffRanges, err := g.getAddedLineRanges(repo, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get diff ranges for %s: %w", filePath, err)
			}
			file.DiffRanges = diffRanges
		}

		files = append(files, file)
	}

	return files, nil
}

func (g *GitScanner) getAddedLineRanges(repo *git.Repository, filePath string) ([]LineRange, error) {
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

	// Get the file from HEAD
	headFile, err := headTree.File(filePath)
	if err != nil {
		// File doesn't exist in HEAD, entire file is new
		return nil, nil
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to read head file content: %w", err)
	}

	// Get current file content
	currentContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read current file: %w", err)
	}

	// Calculate diff and extract added line ranges
	return calculateAddedRanges(headContent, string(currentContent)), nil
}

func calculateAddedRanges(oldContent, newContent string) []LineRange {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	// Use a simple LCS-based diff to find added lines
	lcs := computeLCS(oldLines, newLines)

	var ranges []LineRange
	var currentRange *LineRange

	lcsIndex := 0
	for newLineNum, newLine := range newLines {
		lineNum := newLineNum + 1 // 1-based line numbers

		// Check if this line is in the LCS (i.e., not added)
		if lcsIndex < len(lcs) && newLine == lcs[lcsIndex] {
			// Line exists in old content, close current range if open
			if currentRange != nil {
				ranges = append(ranges, *currentRange)
				currentRange = nil
			}
			lcsIndex++
		} else {
			// Line is added
			if currentRange == nil {
				currentRange = &LineRange{Start: lineNum, End: lineNum}
			} else {
				currentRange.End = lineNum
			}
		}
	}

	// Close final range if open
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

// computeLCS computes the Longest Common Subsequence of two string slices
func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return nil
	}

	// Create DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
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
