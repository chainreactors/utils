package parsers

// ProtonFinding is the output of a proton file scan — one per (template, file).
type ProtonFinding struct {
	TemplateID   string          `json:"template-id"`
	TemplateName string          `json:"template-name"`
	Severity     string          `json:"severity"`
	FilePath     string          `json:"file"`
	Class        string          `json:"class"`
	Events       []TemplateEvent `json:"events,omitempty"`
	Result       *NeutronResult  `json:"-"`
}
