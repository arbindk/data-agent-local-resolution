package contracts

type ManifestSummary struct {
	ExecutionID           string  `json:"execution_id"`
	JobID                 string  `json:"job_id"`
	LeaseID               string  `json:"lease_id"`
	BatchID               string  `json:"batch_id"`
	DataStoreID           string  `json:"data_store_id"`
	SourceSystem          string  `json:"source_system"`
	Status                string  `json:"status"`
	TotalRecords          int     `json:"total_records"`
	ProcessedRecords      int     `json:"processed_records"`
	TotalSizeBytes        int64   `json:"total_size_bytes"`
	CriticalCount         int     `json:"critical_count"`
	HighCount             int     `json:"high_count"`
	MediumCount           int     `json:"medium_count"`
	LowCount              int     `json:"low_count"`
	AverageRiskScore      float64 `json:"average_risk_score"`
	ActionsPlanned        int     `json:"actions_planned"`
	ActionsCompleted      int     `json:"actions_completed"`
	ActionsFailed         int     `json:"actions_failed"`
	CompiledPolicyVersion string  `json:"compiled_policy_version"`
	WorkflowVersion       string  `json:"workflow_version"`
	ProcessingTier        string  `json:"processing_tier"`
	ProcessingMode        string  `json:"processing_mode"`
	StartedAt             string  `json:"started_at"`
	CompletedAt           string  `json:"completed_at"`
}

type ManifestDetail struct {
	ExecutionID       string   `json:"execution_id"`
	BatchID           string   `json:"batch_id"`
	DataStoreID       string   `json:"data_store_id"`
	FileID            string   `json:"file_id"`
	Path              string   `json:"path"`
	Name              string   `json:"name"`
	SizeBytes         int64    `json:"size_bytes"`
	ContentType       string   `json:"content_type"`
	Extension         string   `json:"extension"`
	SourceSystem      string   `json:"source_system"`
	RiskScore         int      `json:"risk_score"`
	RiskLevel         string   `json:"risk_level"`
	Classification    string   `json:"classification"`
	Entities          []string `json:"entities"`
	GuardianDecision  string   `json:"guardian_decision"`
	ActionRequested   string   `json:"action_requested"`
	ActionStatus      string   `json:"action_status"`
	Error             string   `json:"error"`
	ProvenanceID      string   `json:"provenance_id"`
	CompiledPolicyVer string   `json:"compiled_policy_version"`
	WorkflowVersion   string   `json:"workflow_version"`
}
