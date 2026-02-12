package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"walle/internal/languages"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	gitignore "github.com/sabhiram/go-gitignore"
)

func isSupportedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return languages.IsSupportedExtension(ext)
}

func loadGitIgnore(repoRoot string) *gitignore.GitIgnore {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		return nil
	}
	gi, err := gitignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil
	}
	return gi
}

func getRepoRoot(repo *git.Repository) (string, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	return worktree.Filesystem.Root(), nil
}

type Scanner interface {
	GetFiles(opts ScanOptions) ([]File, error)
}

type GitScanner struct{}

func (g *GitScanner) GetFiles(opts ScanOptions) ([]File, error) {
	if len(opts.SpecificFiles) > 0 {
		return g.getSpecificFiles(opts)
	}

	if opts.BaseCommit != "" || opts.TargetCommit != "" {
		return g.getCommitDiff(opts)
	}

	return g.getWorkingTreeChanges(opts)
}

func (g *GitScanner) getSpecificFiles(opts ScanOptions) ([]File, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no source repository found (are you in a source dir?)")
	}

	var gi *gitignore.GitIgnore
	var repoRoot string
	if !opts.IgnoreGitIgnore {
		repoRoot, err = getRepoRoot(repo)
		if err == nil {
			gi = loadGitIgnore(repoRoot)
		}
	}

	var files []File

	for _, filePath := range opts.SpecificFiles {
		if !isSupportedFile(filePath) {
			continue
		}

		if gi != nil {
			relPath, err := filepath.Rel(repoRoot, filePath)
			if err == nil && gi.MatchesPath(relPath) {
				continue
			}
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		file := File{
			Path:    filePath,
			Content: content,
		}

		if opts.Type == ScanWhole {
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

func (g *GitScanner) getCommitDiff(opts ScanOptions) ([]File, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, errors.New("no source repository found (are you in a source dir?)")
	}

	var gi *gitignore.GitIgnore
	if !opts.IgnoreGitIgnore {
		repoRoot, err := getRepoRoot(repo)
		if err == nil {
			gi = loadGitIgnore(repoRoot)
		}
	}

	var baseTree *object.Tree
	if opts.BaseCommit == "" {
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
		baseHash, err := repo.ResolveRevision(plumbing.Revision(opts.BaseCommit))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base commit %s: %w", opts.BaseCommit, err)
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

	var targetTree *object.Tree
	if opts.TargetCommit == "" {
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
		targetHash, err := repo.ResolveRevision(plumbing.Revision(opts.TargetCommit))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target commit %s: %w", opts.TargetCommit, err)
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

		if gi != nil && gi.MatchesPath(toFile.Name) {
			continue
		}

		file, err := g.processTreeFile(toFile, action, baseTree, opts.Type)
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
	if !isSupportedFile(toFile.Name) {
		return nil, nil
	}

	file := &File{
		Path: toFile.Name,
	}

	switch action {
	case merkletrie.Insert:
		file.Status = StatusAdded
	case merkletrie.Modify:
		file.Status = StatusModified
	default:
		return nil, nil
	}

	content, err := toFile.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", toFile.Name, err)
	}
	file.Content = []byte(content)

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

	var gi *gitignore.GitIgnore
	if !opts.IgnoreGitIgnore {
		repoRoot, err := getRepoRoot(repo)
		if err == nil {
			gi = loadGitIgnore(repoRoot)
		}
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
		if !isSupportedFile(filePath) {
			continue
		}

		if gi != nil && gi.MatchesPath(filePath) {
			continue
		}

		if fileStatus.Staging == git.Deleted || fileStatus.Worktree == git.Deleted {
			continue
		}

		var file File
		file.Path = filePath

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

// ValidateCommitOrder checks that target commit is not earlier than base commit
func ValidateCommitOrder(baseCommit, targetCommit string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := git.PlainOpenWithOptions(currentDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return errors.New("no git repository found")
	}

	// Resolve base commit
	baseHash, err := repo.ResolveRevision(plumbing.Revision(baseCommit))
	if err != nil {
		return fmt.Errorf("failed to resolve base commit %s: %w", baseCommit, err)
	}

	// Resolve target commit
	targetHash, err := repo.ResolveRevision(plumbing.Revision(targetCommit))
	if err != nil {
		return fmt.Errorf("failed to resolve target commit %s: %w", targetCommit, err)
	}

	// If base equals target, that's valid
	if *baseHash == *targetHash {
		return nil
	}

	// Check if base is an ancestor of target
	targetCommitObj, err := repo.CommitObject(*targetHash)
	if err != nil {
		return fmt.Errorf("failed to get target commit: %w", err)
	}

	baseCommitObj, err := repo.CommitObject(*baseHash)
	if err != nil {
		return fmt.Errorf("failed to get base commit: %w", err)
	}

	// Check if base is reachable from target (meaning base is an ancestor of target)
	isAncestor, err := baseCommitObj.IsAncestor(targetCommitObj)
	if err != nil {
		return fmt.Errorf("failed to check commit ancestry: %w", err)
	}

	if !isAncestor {
		return fmt.Errorf("target commit is earlier than base commit - target must be later than or equal to base")
	}

	return nil
}
