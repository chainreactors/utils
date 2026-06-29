package parsers

import (
	"strconv"
	"strings"
)

func FormatOutputValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, " \t\r\n\"") {
		return strconv.Quote(value)
	}
	return value
}

func JoinOutput(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, FormatOutputValue(value))
		}
	}
	return strings.Join(parts, " ")
}

func NamesOutput(names []string) string {
	names = NormalizeNames(names)
	if len(names) == 0 {
		return ""
	}
	return "[" + strings.Join(names, ",") + "]"
}

func FrameworkOutput(frames interface{ GetNames() []string }) string {
	if frames == nil {
		return ""
	}
	return NamesOutput(frames.GetNames())
}
