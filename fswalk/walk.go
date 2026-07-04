package fswalk

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charlievieth/fastwalk"
)

// FilterFunc is called for each entry before the main callback.
// Return false to skip the entry (directories are skipped via filepath.SkipDir).
type FilterFunc func(path string, d fs.DirEntry) bool

// WalkFunc is called for each entry that passes all filters.
type WalkFunc func(path string, d fs.DirEntry) error

// Options configures Walk behavior.
type Options struct {
	// Skip contains glob patterns; matching paths are skipped.
	// Directories matching a pattern cause the entire subtree to be skipped.
	Skip []string

	// Include contains glob patterns; when non-empty, only matching files pass.
	// Does not affect directory traversal.
	Include []string

	// DenyExts skips files with these extensions (e.g. ".exe", ".dll").
	// Extensions should include the leading dot and are compared case-insensitively.
	DenyExts []string

	// AllowExts, when non-empty, only allows files with these extensions.
	// Extensions should include the leading dot and are compared case-insensitively.
	AllowExts []string

	// Filter is an optional custom filter applied after built-in filters.
	// Return false to skip the entry.
	Filter FilterFunc

	// IncludeDirs passes directories to WalkFunc when true.
	// By default only regular files are passed.
	IncludeDirs bool

	// FollowLinks follows symlinks during traversal when true.
	FollowLinks bool
}

// Walk concurrently traverses root applying filters, calling fn for each
// entry that passes. Directories matching Skip globs are skipped entirely.
func Walk(root string, opts *Options, fn WalkFunc) error {
	if opts == nil {
		opts = &Options{}
	}

	var skipFilter *GlobFilter
	if len(opts.Skip) > 0 {
		skipFilter = NewGlobFilter(opts.Skip)
	}

	var includeFilter *GlobFilter
	if len(opts.Include) > 0 {
		includeFilter = NewGlobFilter(opts.Include)
	}

	var denyExts map[string]struct{}
	if len(opts.DenyExts) > 0 {
		denyExts = make(map[string]struct{}, len(opts.DenyExts))
		for _, ext := range opts.DenyExts {
			denyExts[strings.ToLower(ext)] = struct{}{}
		}
	}

	var allowExts map[string]struct{}
	if len(opts.AllowExts) > 0 {
		allowExts = make(map[string]struct{}, len(opts.AllowExts))
		for _, ext := range opts.AllowExts {
			allowExts[strings.ToLower(ext)] = struct{}{}
		}
	}

	conf := &fastwalk.Config{
		Follow: opts.FollowLinks,
	}

	return fastwalk.Walk(conf, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		isDir := d.IsDir()

		// glob skip
		if skipFilter != nil && skipFilter.Match(path, isDir) {
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		// directories: skip from callback unless IncludeDirs
		if isDir {
			if opts.IncludeDirs {
				if opts.Filter != nil && !opts.Filter(path, d) {
					return filepath.SkipDir
				}
				return fn(path, d)
			}
			return nil
		}

		// only regular files
		if !d.Type().IsRegular() {
			return nil
		}

		// extension filtering
		ext := strings.ToLower(filepath.Ext(path))
		if denyExts != nil {
			if _, denied := denyExts[ext]; denied {
				return nil
			}
		}
		if allowExts != nil {
			if _, allowed := allowExts[ext]; !allowed {
				return nil
			}
		}

		// include glob
		if includeFilter != nil && !includeFilter.Match(path, false) {
			return nil
		}

		// custom filter
		if opts.Filter != nil && !opts.Filter(path, d) {
			return nil
		}

		return fn(path, d)
	})
}
