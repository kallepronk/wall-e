package discovery

// Collect is the strategy type for file discovery. Each implementation
// captures its own parameters in a closure and returns the discovered
// files when called. The pipeline calls Collect once, then passes the
// result through filters before scanning for comments.
type Collect func() ([]File, error)

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
