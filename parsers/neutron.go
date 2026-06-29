package parsers

type NeutronResult struct {
	Matched        bool
	Extracted      bool
	Matches        map[string][]string
	Extracts       map[string][]string
	OutputExtracts []string
	DynamicValues  map[string]interface{}
	PayloadValues  map[string]interface{}
	Request        string
	Response       string
}
