package provenance

import "time"

type ProvenanceRecord struct {
	DecisionID              string    `json:"decision_id"`
	ExecutionID             string    `json:"execution_id"`
	LeaseID                 string    `json:"lease_id"`
	DataStoreID             string    `json:"data_store_id"`
	FileID                  string    `json:"file_id,omitempty"`
	DecisionType            string    `json:"decision_type"`
	SourcePolicyVersion     string    `json:"source_policy_version"`
	CompiledPolicyVersion   string    `json:"compiled_policy_version"`
	WorkflowVersion         string    `json:"workflow_version"`
	ProcessingTier          string    `json:"processing_tier"`
	RulesApplied            []string  `json:"rules_applied"`
	Reasoning               string    `json:"reasoning"`
	GuardianApproved        bool      `json:"guardian_approved"`
	HumanReviewRequired     bool      `json:"human_review_required"`
	LocalExecutionConfirmed bool      `json:"local_execution_confirmed"`
	ActionRequested         string    `json:"action_requested,omitempty"`
	Actor                   string    `json:"actor"`
	Timestamp               time.Time `json:"timestamp"`
}
