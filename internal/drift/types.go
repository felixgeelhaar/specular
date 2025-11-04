package drift

// Finding represents a drift detection finding
type Finding struct {
	Code      string `json:"code"`       // UNKNOWN_FEATURE, HASH_MISMATCH, etc.
	FeatureID string `json:"feature_id"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // error, warning, info
	Location  string `json:"location,omitempty"`
}

// Report represents a complete drift detection report
type Report struct {
	PlanDrift []Finding `json:"plan_drift"`
	CodeDrift []Finding `json:"code_drift"`
	InfraDrift []Finding `json:"infra_drift"`
	Summary   Summary   `json:"summary"`
}

// Summary provides aggregate statistics for a drift report
type Summary struct {
	TotalFindings int `json:"total_findings"`
	Errors        int `json:"errors"`
	Warnings      int `json:"warnings"`
	Info          int `json:"info"`
}
