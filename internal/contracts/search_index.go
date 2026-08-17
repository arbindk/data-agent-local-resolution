package contracts

import (
	"strings"
	"time"
)

const (
	SearchIndexModeDisabled = "disabled"
	SearchIndexModeLocal    = "local"
	SearchIndexModeElastic  = "elastic"

	SearchIndexStatusPublished = "published"
	SearchIndexStatusQueued    = "queued"
	SearchIndexStatusDisabled  = "disabled"

	SearchIndexKindBatchSummary = "batch_summary"
	SearchIndexKindObject       = "object"
	SearchIndexKindEvidence     = "evidence"
	SearchIndexKindDecision     = "decision"
	SearchIndexKindAction       = "action"
	SearchIndexKindDataPassport = "data_passport"
	SearchIndexKindAudit        = "audit"
)

type SearchIndexDocument struct {
	DocumentID          string            `json:"document_id"`
	IndexName           string            `json:"index_name,omitempty"`
	Kind                string            `json:"kind"`
	TenantID            string            `json:"tenant_id,omitempty"`
	ExecutionID         string            `json:"execution_id,omitempty"`
	JobID               string            `json:"job_id,omitempty"`
	LeaseID             string            `json:"lease_id,omitempty"`
	BatchID             string            `json:"batch_id,omitempty"`
	DataStoreID         string            `json:"data_store_id,omitempty"`
	SourceSystem        string            `json:"source_system,omitempty"`
	ConnectorID         string            `json:"connector_id,omitempty"`
	SubjectType         string            `json:"subject_type,omitempty"`
	SubjectID           string            `json:"subject_id,omitempty"`
	FileID              string            `json:"file_id,omitempty"`
	Container           string            `json:"container,omitempty"`
	Path                string            `json:"path,omitempty"`
	Title               string            `json:"title"`
	Summary             string            `json:"summary,omitempty"`
	Status              string            `json:"status,omitempty"`
	Severity            string            `json:"severity,omitempty"`
	Decision            string            `json:"decision,omitempty"`
	Action              string            `json:"action,omitempty"`
	Classification      string            `json:"classification,omitempty"`
	RiskScore           int               `json:"risk_score,omitempty"`
	Entities            []string          `json:"entities,omitempty"`
	PolicyBundleVersion string            `json:"policy_bundle_version,omitempty"`
	PolicyBundleHash    string            `json:"policy_bundle_hash,omitempty"`
	WorkflowVersion     string            `json:"workflow_version,omitempty"`
	EvidenceRefs        []string          `json:"evidence_refs,omitempty"`
	ProvenanceRefs      []string          `json:"provenance_refs,omitempty"`
	PassportRefs        []string          `json:"passport_refs,omitempty"`
	ActionRefs          []string          `json:"action_refs,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	CreatedAt           time.Time         `json:"created_at,omitempty"`
	UpdatedAt           time.Time         `json:"updated_at,omitempty"`
}

type SearchIndexBatch struct {
	BatchID             string                `json:"batch_id"`
	ExecutionID         string                `json:"execution_id,omitempty"`
	DataStoreID         string                `json:"data_store_id,omitempty"`
	SourceSystem        string                `json:"source_system,omitempty"`
	PolicyBundleVersion string                `json:"policy_bundle_version,omitempty"`
	PolicyBundleHash    string                `json:"policy_bundle_hash,omitempty"`
	WorkflowVersion     string                `json:"workflow_version,omitempty"`
	Documents           []SearchIndexDocument `json:"documents"`
	CreatedAt           time.Time             `json:"created_at,omitempty"`
}

type SearchIndexReceipt struct {
	Status             string    `json:"status"`
	Mode               string    `json:"mode"`
	Backend            string    `json:"backend"`
	IndexName          string    `json:"index_name,omitempty"`
	DocumentsPublished int       `json:"documents_published"`
	DocumentsQueued    int       `json:"documents_queued,omitempty"`
	Error              string    `json:"error,omitempty"`
	PublishedAt        time.Time `json:"published_at,omitempty"`
}

func (d SearchIndexDocument) WithDefaults(now time.Time) SearchIndexDocument {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	d.Kind = strings.TrimSpace(d.Kind)
	if d.Kind == "" {
		d.Kind = "unknown"
	}
	if strings.TrimSpace(d.Title) == "" {
		d.Title = firstSearchIndexValue(d.Path, d.SubjectID, d.FileID, d.ExecutionID, d.Kind)
	}
	if strings.TrimSpace(d.DocumentID) == "" {
		d.DocumentID = StableHash(
			"search_index",
			d.Kind,
			d.ExecutionID,
			d.SubjectID,
			d.FileID,
			d.Path,
			d.Action,
			d.Decision,
			d.Title,
		)
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	return d
}

func (b SearchIndexBatch) WithDefaults(now time.Time) SearchIndexBatch {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	for i := range b.Documents {
		doc := b.Documents[i]
		doc.ExecutionID = firstSearchIndexValue(doc.ExecutionID, b.ExecutionID)
		doc.BatchID = firstSearchIndexValue(doc.BatchID, b.BatchID)
		doc.DataStoreID = firstSearchIndexValue(doc.DataStoreID, b.DataStoreID)
		doc.SourceSystem = firstSearchIndexValue(doc.SourceSystem, b.SourceSystem)
		doc.PolicyBundleVersion = firstSearchIndexValue(doc.PolicyBundleVersion, b.PolicyBundleVersion)
		doc.PolicyBundleHash = firstSearchIndexValue(doc.PolicyBundleHash, b.PolicyBundleHash)
		doc.WorkflowVersion = firstSearchIndexValue(doc.WorkflowVersion, b.WorkflowVersion)
		b.Documents[i] = doc.WithDefaults(now)
	}
	return b
}

func firstSearchIndexValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
