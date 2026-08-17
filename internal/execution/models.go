package execution

import (
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type ExecutionStartRequest struct {
	ExecutionID     string                           `json:"execution_id"`
	Lease           ExecutionLease                   `json:"lease"`
	ExecutionState  ExecutionState                   `json:"execution_state"`
	BatchID         string                           `json:"batch_id,omitempty"`
	BatchInputPath  string                           `json:"batch_input_path,omitempty"`
	RequestEnvelope *contracts.ActionRequestEnvelope `json:"request_envelope,omitempty"`
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
	DataStoreID       string                     `json:"data_store_id"`
	ProcessingTier    string                     `json:"processing_tier"`
	ProcessingMode    string                     `json:"processing_mode"`
	PolicyVersion     string                     `json:"policy_version"`
	WorkflowVersion   string                     `json:"workflow_version"`
	Settings          ExecutionSettings          `json:"settings"`
	OperationalConfig ExecutionOperationalConfig `json:"operational_config"`
}

type ExecutionSettings struct {
	EnableContentIntelligence bool `json:"enable_content_intelligence"`
	EnableGuardianEvaluation  bool `json:"enable_guardian_evaluation"`
	EnableWriteBackActions    bool `json:"enable_write_back_actions"`
	EnableElasticsearchIndex  bool `json:"enable_elasticsearch_index"`
}

type ExecutionOperationalConfig struct {
	BatchSize                    int    `json:"batch_size"`
	LocalExecutionRequired       bool   `json:"local_execution_required"`
	GuardianBeforeActionRequired bool   `json:"guardian_before_action_required"`
	ActionMode                   string `json:"action_mode"`
}

type ExecutionStatus struct {
	ExecutionID           string    `json:"execution_id"`
	RequestID             string    `json:"request_id,omitempty"`
	LeaseID               string    `json:"lease_id"`
	JobID                 string    `json:"job_id"`
	DataStoreID           string    `json:"data_store_id"`
	Status                string    `json:"status"`
	Phase                 string    `json:"phase,omitempty"`
	ProcessingTier        string    `json:"processing_tier"`
	CompiledPolicyVersion string    `json:"compiled_policy_version"`
	PolicyBundleHash      string    `json:"policy_bundle_hash,omitempty"`
	WorkflowVersion       string    `json:"workflow_version"`
	GuardianDecisionID    string    `json:"guardian_decision_id,omitempty"`
	ActionReceiptCount    int       `json:"action_receipt_count,omitempty"`
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
	Status                  ExecutionStatus                    `json:"status"`
	RequestEnvelope         contracts.ActionRequestEnvelope    `json:"request_envelope,omitempty"`
	GuardianDecisionRequest *contracts.GuardianDecisionRequest `json:"guardian_decision_request,omitempty"`
	GuardianDecision        *contracts.GuardianDecision        `json:"guardian_decision,omitempty"`
	ActionReceipts          []contracts.ActionReceipt          `json:"action_receipts,omitempty"`
	ComponentTrace          []contracts.ComponentTrace         `json:"component_trace,omitempty"`
	Summary                 any                                `json:"summary,omitempty"`
	Provenance              []any                              `json:"provenance,omitempty"`
}

type IntentEvaluationResult struct {
	Status                  string                             `json:"status"`
	RequestEnvelope         contracts.ActionRequestEnvelope    `json:"request_envelope"`
	GuardianDecisionRequest *contracts.GuardianDecisionRequest `json:"guardian_decision_request,omitempty"`
	GuardianDecision        *contracts.GuardianDecision        `json:"guardian_decision,omitempty"`
	ActionReceipt           *contracts.ActionReceipt           `json:"action_receipt,omitempty"`
	ComponentTrace          []contracts.ComponentTrace         `json:"component_trace,omitempty"`
	EvaluatedAt             time.Time                          `json:"evaluated_at"`
}
