package execution

import "time"

type ExecutionStartRequest struct {
	ExecutionID    string         `json:"execution_id"`
	Lease          ExecutionLease `json:"lease"`
	ExecutionState ExecutionState `json:"execution_state"`
	BatchInputPath string         `json:"batch_input_path"`
}

type ExecutionLease struct {
	LeaseID               string `json:"lease_id"`
	JobID                 string `json:"job_id"`
	DataStoreID           string `json:"data_store_id"`
	CompiledPolicyVersion string `json:"compiled_policy_version"`
	CompiledGitTag        string `json:"compiled_git_tag"`
	CompiledGitCommit     string `json:"compiled_git_commit"`
	CompiledArtifactHash  string `json:"compiled_artifact_hash"`
	WorkflowVersion       string `json:"workflow_version"`
	WorkflowGitTag        string `json:"workflow_git_tag"`
}

type ExecutionState struct {
	DataStoreID       string            `json:"data_store_id"`
	ProcessingTier    string            `json:"processing_tier"`
	ProcessingMode    string            `json:"processing_mode"`
	PolicyVersion     string            `json:"policy_version"`
	WorkflowVersion   string            `json:"workflow_version"`
	Settings          ExecutionSettings `json:"settings"`
	OperationalConfig map[string]any    `json:"operational_config"`
}

type ExecutionSettings struct {
	EnableContentIntelligence bool `json:"enable_content_intelligence"`
	EnableGuardianEvaluation  bool `json:"enable_guardian_evaluation"`
	EnableWriteBackActions    bool `json:"enable_write_back_actions"`
	EnableElasticsearchIndex  bool `json:"enable_elasticsearch_index"`
}

type NormalizedBatch struct {
	BatchID     string                 `json:"batch_id"`
	LeaseID     string                 `json:"lease_id"`
	DataStoreID string                 `json:"data_store_id"`
	Records     []NormalizedFileRecord `json:"records"`
}

type NormalizedFileRecord struct {
	FileID              string    `json:"file_id"`
	DataStoreID         string    `json:"data_store_id"`
	Path                string    `json:"path"`
	Name                string    `json:"name"`
	SizeBytes           int64     `json:"size_bytes"`
	LastModified        string    `json:"last_modified"`
	LastAccessed        string    `json:"last_accessed"`
	ContentType         string    `json:"content_type"`
	Extension           string    `json:"extension"`
	SourceSystem        string    `json:"source_system"`
	PermissionClass     string    `json:"permission_class"`
	PermissionRiskScore int       `json:"permission_risk_score"`
	DDProtectionFlags   int       `json:"dd_protection_flags"`
	ScanTimestamp       time.Time `json:"scan_timestamp"`
}

type ExecutionStatus struct {
	ExecutionID           string    `json:"execution_id"`
	LeaseID               string    `json:"lease_id"`
	JobID                 string    `json:"job_id"`
	DataStoreID           string    `json:"data_store_id"`
	Status                string    `json:"status"`
	ProcessingTier        string    `json:"processing_tier"`
	CompiledPolicyVersion string    `json:"compiled_policy_version"`
	WorkflowVersion       string    `json:"workflow_version"`
	RecordsProcessed      int       `json:"records_processed"`
	PolicyResolved        bool      `json:"policy_resolved"`
	WorkflowResolved      bool      `json:"workflow_resolved"`
	GuardianApproved      bool      `json:"guardian_approved"`
	WriteBackRequested    bool      `json:"write_back_requested"`
	Degraded              bool      `json:"degraded"`
	Reason                string    `json:"reason,omitempty"`
	StartedAt             time.Time `json:"started_at"`
	CompletedAt           time.Time `json:"completed_at,omitempty"`
}

type ExecutionResult struct {
	Status     ExecutionStatus `json:"status"`
	Summary    any             `json:"summary,omitempty"`
	Provenance []any           `json:"provenance,omitempty"`
}
