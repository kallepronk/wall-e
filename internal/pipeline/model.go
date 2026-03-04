package pipeline

import (
	"walle/internal/comment"
	"walle/internal/discovery"
	"walle/internal/filter"
)

// RunConfig is everything the pipeline needs to perform a scan.
// Cmd constructs it by mapping CLI flags to a Collect strategy and
// a slice of Filters; the pipeline itself has no knowledge of flags.
type RunConfig struct {
	Collect discovery.Collect
	Filters []filter.Filter
}

// ScanResult holds both the discovered comments and any non-fatal warnings
// produced during file discovery.
type ScanResult struct {
	Comments []comment.Comment
	Warnings []discovery.Warning
}
