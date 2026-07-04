package fswalk

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

var percentEnvPattern = regexp.MustCompile(`%([^%]+)%`)

type GlobFilter struct {
	matchers []glob.Glob
}

func NewGlobFilter(patterns []string) *GlobFilter {
	filter := &GlobFilter{matchers: make([]glob.Glob, 0, len(patterns))}
	for _, raw := range patterns {
		pattern := normalizePattern(raw)
		if pattern == "" {
			continue
		}
		if g, err := glob.Compile(pattern, '/'); err == nil {
			filter.matchers = append(filter.matchers, g)
		}
	}
	return filter
}

func (f *GlobFilter) Match(path string, isDir bool) bool {
	if f == nil || len(f.matchers) == 0 {
		return false
	}
	candidate := normalizePath(path)
	if candidate == "" {
		return false
	}
	if isDir {
		candidate += "/"
	}
	for _, g := range f.matchers {
		if g.Match(candidate) {
			return true
		}
	}
	return false
}

func normalizePattern(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "~" || strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		if raw == "~" {
			raw = home
		} else {
			raw = filepath.Join(home, raw[2:])
		}
	}
	raw = expandEnv(raw)
	return strings.ToLower(filepath.ToSlash(raw))
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
}

func expandEnv(raw string) string {
	raw = os.ExpandEnv(raw)
	if !strings.Contains(raw, "%") {
		return raw
	}
	return percentEnvPattern.ReplaceAllStringFunc(raw, func(match string) string {
		name := strings.Trim(match, "%")
		if name == "" {
			return match
		}
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
		return match
	})
}
