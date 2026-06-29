package parsers

import (
	"sort"
	"strings"
)

func NormalizeNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func FrameworkNames(frames interface{ GetNames() []string }) []string {
	if frames == nil {
		return nil
	}
	return NormalizeNames(frames.GetNames())
}
