package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	DecisionAllow          = "allow"
	DecisionDeny           = "deny"
	DecisionModify         = "modify"
	DecisionFlag           = "flag"
	DecisionReviewRequired = "review_required"

	ActionStatusPlanned   = "planned"
	ActionStatusCompleted = "completed"
	ActionStatusBlocked   = "blocked"
	ActionStatusFailed    = "failed"
	ActionStatusSkipped   = "skipped"

	ObligationRecordAudit      = "record_audit"
	ObligationAttachPolicyHash = "attach_policy_hash"
	ObligationWriteProvenance  = "write_provenance"
	ObligationBoundedPreview   = "bounded_preview_only"
)

type ActorContext struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
	AuthType string `json:"auth_type,omitempty"`
	ClientID string `json:"client_id,omitempty"`
}

type ObjectRef struct {
	URI         string `json:"uri,omitempty"`
	ConnectorID string `json:"connector_id,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
	Container   string `json:"container,omitempty"`
	Path        string `json:"path,omitempty"`
	VersionID   string `json:"version_id,omitempty"`
	ETag        string `json:"etag,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Region      string `json:"region,omitempty"`
}

type DataScope struct {
	SourceID     string            `json:"source_id,omitempty"`
	Container    string            `json:"container,omitempty"`
	Prefix       string            `json:"prefix,omitempty"`
	ObjectRefs   []ObjectRef       `json:"object_refs,omitempty"`
	TargetRefs   []ObjectRef       `json:"target_refs,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	BatchID      string            `json:"batch_id,omitempty"`
	Filters      map[string]string `json:"filters,omitempty"`
	Boundary     string            `json:"boundary,omitempty"`
	ZeroCopyMode string            `json:"zero_copy_mode,omitempty"`
}

type PolicyContextRef struct {
	PolicyID            string   `json:"policy_id,omitempty"`
	PolicyBundleVersion string   `json:"policy_bundle_version,omitempty"`
	PolicyBundleHash    string   `json:"policy_bundle_hash,omitempty"`
	WorkflowVersion     string   `json:"workflow_version,omitempty"`
	RulesetIDs          []string `json:"ruleset_ids,omitempty"`
}

type RuntimeContext struct {
	Environment            string `json:"environment,omitempty"`
	Region                 string `json:"region,omitempty"`
	DataBoundary           string `json:"data_boundary,omitempty"`
	ActionMode             string `json:"action_mode,omitempty"`
	ProcessingTier         string `json:"processing_tier,omitempty"`
	ProcessingMode         string `json:"processing_mode,omitempty"`
	LocalExecutionRequired bool   `json:"local_execution_required,omitempty"`
	ZeroCopyRequired       bool   `json:"zero_copy_required,omitempty"`
	AirGapped              bool   `json:"air_gapped,omitempty"`
}

type ActionRequestEnvelope struct {
	RequestID      string            `json:"request_id"`
	TenantID       string            `json:"tenant_id"`
	Actor          ActorContext      `json:"actor"`
	SourceID       string            `json:"source_id"`
	ConnectorID    string            `json:"connector_id"`
	Operation      string            `json:"operation"`
	Scope          DataScope         `json:"scope"`
	PolicyContext  PolicyContextRef  `json:"policy_context"`
	RuntimeContext RuntimeContext    `json:"runtime_context,omitempty"`
	CorrelationID  string            `json:"correlation_id"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	RequestedAt    time.Time         `json:"requested_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (e ActionRequestEnvelope) Validate() error {
	missing := make([]string, 0)
	if strings.TrimSpace(e.RequestID) == "" {
		missing = append(missing, "request_id")
	}
	if strings.TrimSpace(e.TenantID) == "" {
		missing = append(missing, "tenant_id")
	}
	if strings.TrimSpace(e.Actor.ID) == "" {
		missing = append(missing, "actor.id")
	}
	if strings.TrimSpace(e.SourceID) == "" {
		missing = append(missing, "source_id")
	}
	if strings.TrimSpace(e.ConnectorID) == "" {
		missing = append(missing, "connector_id")
	}
	if strings.TrimSpace(e.Operation) == "" {
		missing = append(missing, "operation")
	}
	if strings.TrimSpace(e.CorrelationID) == "" {
		missing = append(missing, "correlation_id")
	}
	if strings.TrimSpace(e.PolicyContext.PolicyID) == "" &&
		strings.TrimSpace(e.PolicyContext.PolicyBundleVersion) == "" {
		missing = append(missing, "policy_context.policy_id_or_policy_bundle_version")
	}
	if len(missing) > 0 {
		return errors.New("invalid action request envelope: missing " + strings.Join(missing, ", "))
	}
	return nil
}

type ExecutionPlan struct {
	PlanID        string            `json:"plan_id"`
	RequestID     string            `json:"request_id"`
	ExecutionID   string            `json:"execution_id,omitempty"`
	WorkflowID    string            `json:"workflow_id,omitempty"`
	WorkflowSteps []string          `json:"workflow_steps,omitempty"`
	BatchSize     int               `json:"batch_size,omitempty"`
	ReadWorkers   int               `json:"read_workers,omitempty"`
	WriteWorkers  int               `json:"write_workers,omitempty"`
	Checkpoints   []string          `json:"checkpoints,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ConnectorCapability struct {
	ConnectorID         string   `json:"connector_id"`
	Runtime             string   `json:"runtime,omitempty"`
	SupportsDiscover    bool     `json:"supports_discover"`
	SupportsMetadata    bool     `json:"supports_metadata"`
	SupportsFeatures    bool     `json:"supports_features"`
	SupportsBoundedRead bool     `json:"supports_bounded_read"`
	SupportsWriteBack   bool     `json:"supports_writeback"`
	SupportedActions    []string `json:"supported_actions,omitempty"`
	MaxReadWorkers      int      `json:"max_read_workers,omitempty"`
	MaxWriteWorkers     int      `json:"max_write_workers,omitempty"`
	PageSize            int      `json:"page_size,omitempty"`
	MaxSampleBytes      int64    `json:"max_sample_bytes,omitempty"`
}

type ConnectorReadRequest struct {
	RequestID       string            `json:"request_id"`
	ExecutionID     string            `json:"execution_id,omitempty"`
	ConnectorID     string            `json:"connector_id"`
	Scope           DataScope         `json:"scope"`
	ReadMode        string            `json:"read_mode"`
	MaxBytes        int64             `json:"max_bytes,omitempty"`
	ZeroCopy        bool              `json:"zero_copy"`
	IncludeMetadata bool              `json:"include_metadata"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type MetadataBatch struct {
	BatchID     string            `json:"batch_id"`
	RequestID   string            `json:"request_id"`
	ExecutionID string            `json:"execution_id,omitempty"`
	SourceID    string            `json:"source_id"`
	ConnectorID string            `json:"connector_id"`
	ObjectCount int               `json:"object_count"`
	ObjectRefs  []ObjectRef       `json:"object_refs,omitempty"`
	Signals     map[string]int    `json:"signals,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at,omitempty"`
}

type ClassificationResult struct {
	ObjectRef          ObjectRef         `json:"object_ref"`
	Classification     string            `json:"classification"`
	RiskScore          int               `json:"risk_score"`
	EntityTypes        []string          `json:"entity_types,omitempty"`
	RecommendedActions []string          `json:"recommended_actions,omitempty"`
	RequiresHITL       bool              `json:"requires_hitl"`
	Signals            map[string]string `json:"signals,omitempty"`
}

type GuardianDecisionRequest struct {
	DecisionRequestID string            `json:"decision_request_id"`
	RequestID         string            `json:"request_id"`
	ExecutionID       string            `json:"execution_id,omitempty"`
	TenantID          string            `json:"tenant_id"`
	Actor             ActorContext      `json:"actor"`
	Operation         string            `json:"operation"`
	ObjectRefs        []ObjectRef       `json:"object_refs,omitempty"`
	ObjectCount       int               `json:"object_count,omitempty"`
	Classification    string            `json:"classification,omitempty"`
	RiskScore         int               `json:"risk_score,omitempty"`
	SourceRegion      string            `json:"source_region,omitempty"`
	TargetRegion      string            `json:"target_region,omitempty"`
	PolicyContext     PolicyContextRef  `json:"policy_context"`
	RuntimeContext    RuntimeContext    `json:"runtime_context,omitempty"`
	ActionParameters  map[string]string `json:"action_parameters,omitempty"`
	RequestedAt       time.Time         `json:"requested_at,omitempty"`
}

type GuardianObligation struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

type GuardianDecision struct {
	DecisionID              string               `json:"decision_id"`
	DecisionRequestID       string               `json:"decision_request_id,omitempty"`
	RequestID               string               `json:"request_id"`
	ExecutionID             string               `json:"execution_id,omitempty"`
	Decision                string               `json:"decision"`
	Approved                bool                 `json:"approved"`
	PolicyBundleVersion     string               `json:"policy_bundle_version"`
	PolicyBundleHash        string               `json:"policy_bundle_hash"`
	RulesApplied            []string             `json:"rules_applied"`
	Obligations             []GuardianObligation `json:"obligations,omitempty"`
	HITLRequired            bool                 `json:"hitl_required"`
	HumanReviewRequired     bool                 `json:"human_review_required"`
	Reasoning               string               `json:"reasoning"`
	RiskScore               int                  `json:"risk_score,omitempty"`
	ModifiedAction          *ActionCommand       `json:"modified_action,omitempty"`
	EvaluatedAt             time.Time            `json:"evaluated_at,omitempty"`
	LocalExecutionConfirmed bool                 `json:"local_execution_confirmed"`
}

type ActionCommand struct {
	ActionID       string            `json:"action_id"`
	RequestID      string            `json:"request_id"`
	ExecutionID    string            `json:"execution_id,omitempty"`
	DecisionID     string            `json:"decision_id"`
	ConnectorID    string            `json:"connector_id"`
	ActionType     string            `json:"action_type"`
	ObjectRefs     []ObjectRef       `json:"object_refs,omitempty"`
	Parameters     map[string]string `json:"parameters,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	DryRun         bool              `json:"dry_run"`
	RequestedAt    time.Time         `json:"requested_at,omitempty"`
}

type ActionReceipt struct {
	ReceiptID          string      `json:"receipt_id"`
	ActionID           string      `json:"action_id"`
	RequestID          string      `json:"request_id"`
	ExecutionID        string      `json:"execution_id,omitempty"`
	DecisionID         string      `json:"decision_id"`
	ConnectorID        string      `json:"connector_id"`
	ObjectRefs         []ObjectRef `json:"object_refs,omitempty"`
	ActionType         string      `json:"action_type"`
	Status             string      `json:"status"`
	Applied            bool        `json:"applied"`
	DryRun             bool        `json:"dry_run"`
	ResultSummary      string      `json:"result_summary,omitempty"`
	Error              string      `json:"error,omitempty"`
	ObligationsApplied []string    `json:"obligations_applied,omitempty"`
	EvidenceRefs       []string    `json:"evidence_refs,omitempty"`
	ProvenanceRefs     []string    `json:"provenance_refs,omitempty"`
	StartedAt          time.Time   `json:"started_at,omitempty"`
	CompletedAt        time.Time   `json:"completed_at,omitempty"`
	ReceiptHash        string      `json:"receipt_hash"`
}

type TelemetryEvent struct {
	EventID     string             `json:"event_id"`
	RequestID   string             `json:"request_id"`
	ExecutionID string             `json:"execution_id,omitempty"`
	Component   string             `json:"component"`
	EventType   string             `json:"event_type"`
	Status      string             `json:"status,omitempty"`
	DurationMS  int64              `json:"duration_ms,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Attributes  map[string]string  `json:"attributes,omitempty"`
	ObservedAt  time.Time          `json:"observed_at,omitempty"`
}

type ComponentTrace struct {
	Component   string    `json:"component"`
	Stage       string    `json:"stage,omitempty"`
	Status      string    `json:"status"`
	Detail      string    `json:"detail,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type ExecutionSummary struct {
	ExecutionID           string            `json:"execution_id"`
	RequestID             string            `json:"request_id"`
	Status                string            `json:"status"`
	ObjectCount           int               `json:"object_count"`
	DecisionCounts        map[string]int    `json:"decision_counts,omitempty"`
	ActionCounts          map[string]int    `json:"action_counts,omitempty"`
	EvidenceRefs          []string          `json:"evidence_refs,omitempty"`
	ProvenanceRefs        []string          `json:"provenance_refs,omitempty"`
	Errors                []ErrorModel      `json:"errors,omitempty"`
	Warnings              []string          `json:"warnings,omitempty"`
	NextRecommendedAction string            `json:"next_recommended_action,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
}

type ErrorModel struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Component   string `json:"component,omitempty"`
	Retryable   bool   `json:"retryable"`
	Remediation string `json:"remediation,omitempty"`
}

func StableHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func ObligationIDs(obligations []GuardianObligation) []string {
	out := make([]string, 0, len(obligations))
	for _, obligation := range obligations {
		if obligation.ID != "" {
			out = append(out, obligation.ID)
		}
	}
	return out
}
