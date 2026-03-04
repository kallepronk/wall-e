package pipeline

import (
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
