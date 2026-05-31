package batchsummary

import "time"

type BatchSummary struct {
	BatchSummaryID        string                `json:"batch_summary_id"`
	ExecutionID           string                `json:"execution_id"`
	BatchID               string                `json:"batch_id"`
	LeaseID               string                `json:"lease_id"`
	DataStoreID           string                `json:"data_store_id"`
	ProcessedAt           time.Time             `json:"processed_at"`
	FileCount             int                   `json:"file_count"`
	ProcessingTier        string                `json:"processing_tier"`
	ProcessingMode        string                `json:"processing_mode"`
	CompiledPolicyVersion string                `json:"compiled_policy_version"`
	WorkflowVersion       string                `json:"workflow_version"`
	RiskSummary           RiskSummary           `json:"risk_summary"`
	Classification        ClassificationSummary `json:"classification"`
	ActionsTaken          ActionsTaken          `json:"actions_taken"`
	Sovereignty           SovereigntySummary    `json:"sovereignty"`
	PolicyExceptions      []PolicyException     `json:"policy_exceptions"`
	Metadata              map[string]any        `json:"metadata,omitempty"`
}

type RiskSummary struct {
	Critical     int     `json:"critical"`
	High         int     `json:"high"`
	Medium       int     `json:"medium"`
	Low          int     `json:"low"`
	AvgRiskScore float64 `json:"avg_risk_score"`
}

type ClassificationSummary struct {
	SensitivityDistribution map[string]int `json:"sensitivity_distribution"`
	EntityTypeCounts        map[string]int `json:"entity_type_counts"`
}

type ActionsTaken struct {
	Tagged           int `json:"tagged"`
	FlaggedForReview int `json:"flagged_for_review"`
	Blocked          int `json:"blocked"`
	WriteBackPlanned int `json:"write_back_planned"`
}

type SovereigntySummary struct {
	Required                string `json:"required"`
	Current                 string `json:"current"`
	Compliant               bool   `json:"compliant"`
	LocalExecutionConfirmed bool   `json:"local_execution_confirmed"`
	Violations              int    `json:"violations"`
}

type PolicyException struct {
	FileID     string `json:"file_id"`
	Path       string `json:"path"`
	Reason     string `json:"reason"`
	Severity   string `json:"severity"`
	Action     string `json:"action"`
	DecisionID string `json:"decision_id"`
}
