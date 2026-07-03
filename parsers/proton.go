package parsers

type ProtonMatchEvent struct {
	Value  string `json:"value"`
	Line   int    `json:"line"`
	Offset int    `json:"offset,omitempty"`
}

type ProtonFinding struct {
	TemplateID   string                          `json:"template-id"`
	TemplateName string                          `json:"template-name"`
	Severity     string                          `json:"severity"`
	FilePath     string                          `json:"file"`
	Class        string                          `json:"class"`
	Matches      map[string][]ProtonMatchEvent   `json:"matches,omitempty"`
	Extracts     []ProtonMatchEvent              `json:"extracts,omitempty"`
	Result       *NeutronResult                  `json:"-"`
}
