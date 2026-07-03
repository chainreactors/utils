package parsers

// TemplateEvent is a single match or extract event produced by the template engine.
// Shared by both proton (file scanning) and neutron (network scanning).
type TemplateEvent struct {
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Rule   string `json:"rule,omitempty"`
	Value  string `json:"value"`
	Line   int    `json:"line,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
