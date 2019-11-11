package collector

// indexStatsResponse is a representation of a Elasticsearch Index Stats
type IndicesSettingsResponse struct {
	Indices map[string]IndexStatsILMResponse `json:"indices"`
}

// IndexStatsIndexResponse defines index stats index information structure
type IndexStatsILMResponse struct {
	FailedStep string           `json:"failed_step"`
	StepInfo   StepInfoResponse `json:"step_info"`
}

type StepInfoResponse struct {
	Reason string `json:"reason"`
}
