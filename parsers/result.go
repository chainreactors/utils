package parsers

type Result struct {
	TemplateID    string                 `json:"template_id"`
	TemplateName  string                 `json:"template_name,omitempty"`
	Severity      string                 `json:"severity,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Matched       bool                   `json:"matched"`
	Extracted     bool                   `json:"extracted"`
	Events        []TemplateEvent        `json:"events,omitempty"`
	DynamicValues map[string]interface{} `json:"-"`
}

func (r *Result) OutputExtracts() []string {
	if r == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, ev := range r.Events {
		if ev.Type == "extract" {
			if _, ok := seen[ev.Value]; !ok {
				seen[ev.Value] = struct{}{}
				out = append(out, ev.Value)
			}
		}
	}
	return out
}

func (r *Result) MatchesByName() map[string][]string {
	if r == nil {
		return nil
	}
	m := make(map[string][]string)
	for _, ev := range r.Events {
		if ev.Type == "match" {
			m[ev.Name] = append(m[ev.Name], ev.Value)
		}
	}
	return m
}

func (r *Result) ExtractsByName() map[string][]string {
	if r == nil {
		return nil
	}
	m := make(map[string][]string)
	for _, ev := range r.Events {
		if ev.Type == "extract" {
			m[ev.Name] = append(m[ev.Name], ev.Value)
		}
	}
	return m
}
