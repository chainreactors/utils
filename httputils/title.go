package httputils

import (
	"regexp"
	"strings"
)

var (
	titleRegexp = regexp.MustCompile("(?Uis)<title>(.*)</title>")
)

func MatchTitle(content []byte) string {
	matched := titleRegexp.FindSubmatch(content)
	if len(matched) >= 2 {
		return strings.TrimSpace(string(matched[1]))
	}
	return ""
}

func MatchCharacter(content []byte) string {
	if len(content) > 13 {
		return string(content[0:13])
	}
	return string(content)
}
