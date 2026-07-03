package parsers

type NeutronResult struct {
	Matched       bool
	Extracted     bool
	Events        []TemplateEvent
	DynamicValues map[string]interface{}
	PayloadValues map[string]interface{}
	Request       string
	Response      string
}

// OutputExtracts returns a flat deduplicated list of extract event values.
func (r *NeutronResult) OutputExtracts() []string {
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

// MatchesByName returns matcher events grouped by name (backward compat helper).
func (r *NeutronResult) MatchesByName() map[string][]string {
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

// ExtractsByName returns extractor events grouped by name (backward compat helper).
func (r *NeutronResult) ExtractsByName() map[string][]string {
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
