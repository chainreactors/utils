package parsers

type FindingResult struct {
	Result
	FilePath string `json:"file"`
	Class    string `json:"class"`
}
