package filter

import "walle/internal/discovery"

// Filter removes files from a discovery result that should not be scanned.
// Filters are applied sequentially in the pipeline after collection, before
// comment scanning. Each implementation is a named struct so it can be
// inspected, logged, and tested independently.
type Filter interface {
	Apply(files []discovery.File) []discovery.File
}
