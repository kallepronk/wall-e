package source

type ScanType int

const (
	ScanWhole ScanType = iota
	ScanDiff
)

type ScanOptions struct {
	Type ScanType

	SpecificFiles []string

	BaseCommit   string
	TargetCommit string

	IncludeUntracked bool
	IgnoreGitIgnore  bool
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
