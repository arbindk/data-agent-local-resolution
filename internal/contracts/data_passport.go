package contracts

import (
	"fmt"
	"strings"
	"time"
)

const (
	DataPassportSchemaVersion = "data_passport_v1"

	DataPassportSubjectObject    = "object"
	DataPassportSubjectExecution = "execution"

	DataPassportStageObserved        = "observed"
	DataPassportStageClassified      = "classified"
	DataPassportStageActionPlanned   = "action_planned"
	DataPassportStageActionApplied   = "action_applied"
	DataPassportStageBlocked         = "blocked"
	DataPassportStagePendingApproval = "pending_approval"
)

type DataPassport struct {
	PassportID          string               `json:"passport_id"`
	SchemaVersion       string               `json:"schema_version"`
	SubjectType         string               `json:"subject_type"`
	Subject             ObjectRef            `json:"subject"`
	ExecutionID         string               `json:"execution_id,omitempty"`
	JobID               string               `json:"job_id,omitempty"`
	BatchID             string               `json:"batch_id,omitempty"`
	DataStoreID         string               `json:"data_store_id,omitempty"`
	SourceSystem        string               `json:"source_system,omitempty"`
	LifecycleStage      string               `json:"lifecycle_stage"`
	Classification      string               `json:"classification,omitempty"`
	RiskScore           int                  `json:"risk_score,omitempty"`
	RiskLevel           string               `json:"risk_level,omitempty"`
	Entities            []string             `json:"entities,omitempty"`
	PolicyBundleVersion string               `json:"policy_bundle_version,omitempty"`
	PolicyBundleHash    string               `json:"policy_bundle_hash,omitempty"`
	WorkflowVersion     string               `json:"workflow_version,omitempty"`
	GuardianDecision    string               `json:"guardian_decision,omitempty"`
	DecisionID          string               `json:"decision_id,omitempty"`
	ActionRequested     string               `json:"action_requested,omitempty"`
	ActionStatus        string               `json:"action_status,omitempty"`
	LastAction          string               `json:"last_action,omitempty"`
	LastActionStatus    string               `json:"last_action_status,omitempty"`
	Locality            DataPassportLocality `json:"locality"`
	Obligations         []string             `json:"obligations,omitempty"`
	EvidenceRefs        []string             `json:"evidence_refs,omitempty"`
	ProvenanceRefs      []string             `json:"provenance_refs,omitempty"`
	ActionRefs          []string             `json:"action_refs,omitempty"`
	Metadata            map[string]string    `json:"metadata,omitempty"`
	CreatedAt           string               `json:"created_at"`
	UpdatedAt           string               `json:"updated_at"`
	ExpiresAt           string               `json:"expires_at,omitempty"`
	PassportHash        string               `json:"passport_hash"`
}

type DataPassportLocality struct {
	ExecutionBoundary string `json:"execution_boundary,omitempty"`
	SourceRegion      string `json:"source_region,omitempty"`
	TargetRegion      string `json:"target_region,omitempty"`
	DataMovement      string `json:"data_movement,omitempty"`
	ZeroCopyMode      string `json:"zero_copy_mode,omitempty"`
}

func (p DataPassport) ComputeHash() string {
	return StableHash(
		p.PassportID,
		p.SchemaVersion,
		p.SubjectType,
		p.Subject.URI,
		p.Subject.SourceID,
		p.Subject.ConnectorID,
		p.Subject.Container,
		p.Subject.Path,
		p.ExecutionID,
		p.BatchID,
		p.DataStoreID,
		p.SourceSystem,
		p.LifecycleStage,
		p.Classification,
		fmt.Sprintf("%d", p.RiskScore),
		p.RiskLevel,
		p.PolicyBundleVersion,
		p.PolicyBundleHash,
		p.WorkflowVersion,
		p.GuardianDecision,
		p.DecisionID,
		p.ActionRequested,
		p.ActionStatus,
		p.LastAction,
		p.LastActionStatus,
		strings.Join(p.EvidenceRefs, "|"),
		strings.Join(p.ProvenanceRefs, "|"),
		strings.Join(p.ActionRefs, "|"),
	)
}

func (p DataPassport) WithDefaults(now time.Time) DataPassport {
	if p.SchemaVersion == "" {
		p.SchemaVersion = DataPassportSchemaVersion
	}
	if p.SubjectType == "" {
		p.SubjectType = DataPassportSubjectObject
	}
	if p.LifecycleStage == "" {
		p.LifecycleStage = DataPassportStageObserved
	}
	if p.CreatedAt == "" {
		p.CreatedAt = now.UTC().Format(time.RFC3339)
	}
	p.UpdatedAt = now.UTC().Format(time.RFC3339)
	p.PassportHash = p.ComputeHash()
	return p
}
