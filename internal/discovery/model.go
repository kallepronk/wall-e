package discovery

import "fmt"

// Collect is the strategy type for file discovery. Each implementation
// captures its own parameters in a closure and returns the discovered
// files when called. The pipeline calls Collect once, then passes the
// result through filters before scanning for comments.
//
// Non-fatal problems (e.g. symlinks, unreadable files) are reported as
// warnings so the caller can log them without aborting the entire run.
type Collect func() ([]File, []Warning, error)

// Warning describes a non-fatal issue encountered during file discovery.
type Warning struct {
	Path    string
	Message string
}

func (w Warning) String() string {
	return fmt.Sprintf("%s: %s", w.Path, w.Message)
}

type File struct {
	Path       string
	Status     FileStatus
	Content    []byte
	DiffRanges []LineRange
}

type LineRange struct {
	Start int
	End   int
}

type FileStatus int

const (
	StatusModified FileStatus = iota
	StatusAdded
	StatusUntracked
	StatusDeleted
)
