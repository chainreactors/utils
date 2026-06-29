package parsers

import (
	"fmt"
	"github.com/chainreactors/utils/encode"
	"github.com/chainreactors/utils/iutils"
	"regexp"
	"strings"
)

type ExtractItem struct {
	Value  string `json:"value"`
	Ctx    string `json:"ctx,omitempty"`
	Line   int    `json:"line,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type Extracted struct {
	Name          string        `json:"name"`
	Severity      string        `json:"severity,omitempty"`
	ExtractResult []string      `json:"extract_result"`
	Items         []ExtractItem `json:"items,omitempty"`
}

func (e *Extracted) String() string {
	if len(e.ExtractResult) == 1 {
		if len(e.ExtractResult[0]) > 30 {
			return fmt.Sprintf("%s:%s ... %d bytes", e.Name, iutils.AsciiEncode(e.ExtractResult[0][:30]), len(e.ExtractResult[0]))
		}
		return fmt.Sprintf("%s:%s", e.Name, iutils.AsciiEncode(e.ExtractResult[0]))
	} else {
		return fmt.Sprintf("%s:%d items", e.Name, len(e.ExtractResult))
	}
}

type Extracteds []*Extracted

func (es Extracteds) String() string {
	var s strings.Builder
	for _, e := range es {
		s.WriteString("[ " + e.String() + " ]")
	}
	return s.String() + " "
}

func (es *Extracteds) Merge(other Extracteds) {
	for _, e := range other {
		if len(e.ExtractResult) > 0 {
			*es = append(*es, e)
		}
	}
}

type Extractor struct {
	Name            string           `json:"name"` // extractor name
	Regexps         []string         `json:"regexps"`
	Tags            []string         `json:"tags"`
	CompiledRegexps []*regexp.Regexp `json:"-"`
	Cases           []string         `json:"cases"`
}

func (e *Extractor) Compile() {
	e.CompiledRegexps = make([]*regexp.Regexp, len(e.Regexps))
	for i, r := range e.Regexps {
		if parsed, ok := encode.DSLParserToString(iutils.ToString(r)); ok {
			e.CompiledRegexps[i] = regexp.MustCompile(parsed)
		} else {
			e.CompiledRegexps[i] = regexp.MustCompile(iutils.ToString(r))
		}
	}
}

func (e *Extractor) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (e *Extractor) Extract(body string) *Extracted {
	extracts := &Extracted{
		Name: e.Name,
	}
	for _, r := range e.CompiledRegexps {
		matches := r.FindAllString(body, -1)
		if len(matches) > 0 {
			extracts.ExtractResult = append(extracts.ExtractResult, matches...)
		}
	}
	return extracts
}

func (e *Extractor) ExtractUnique(body string) *Extracted {
	extracts := &Extracted{
		Name: e.Name,
	}
	uniqueMatches := make(map[string]struct{})

	for _, r := range e.CompiledRegexps {
		matches := r.FindAllString(body, -1)
		for _, match := range matches {
			uniqueMatches[match] = struct{}{}
		}
	}

	// 直接存储去重后的结果
	extracts.ExtractResult = make([]string, 0, len(uniqueMatches))
	for match := range uniqueMatches {
		extracts.ExtractResult = append(extracts.ExtractResult, match)
	}

	return extracts
}

type Extractors map[string][]*Extractor

func (es Extractors) Extract(content string, unique bool) (extracteds []*Extracted) {
	if len(content) == 0 {
		return
	}

	for _, extract := range es {
		for _, e := range extract {
			var extracted *Extracted
			if unique {
				extracted = e.ExtractUnique(content)
			} else {
				extracted = e.Extract(content)
			}
			if extracted.ExtractResult != nil {
				extracteds = append(extracteds, extracted)
			}
		}
	}
	return extracteds
}
