package parsers

type NeutronResult struct {
	Result
	Target        string                 `json:"target"`
	Request       string                 `json:"request,omitempty"`
	Response      string                 `json:"response,omitempty"`
	PayloadValues map[string]interface{} `json:"payload_values,omitempty"`
}
