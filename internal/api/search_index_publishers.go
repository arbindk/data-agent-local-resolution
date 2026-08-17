package api

import (
	"context"
	"fmt"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) publishSearchDocuments(docs ...contracts.SearchIndexDocument) {
	if s == nil || s.searchIndex == nil || len(docs) == 0 {
		return
	}
	ctx := context.Background()
	if s.cfg.SearchIndexTimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(s.cfg.SearchIndexTimeoutSeconds)*time.Second)
		defer cancel()
	}
	_, _ = s.searchIndex.Publish(ctx, contracts.SearchIndexBatch{
		DataStoreID: "portal-state",
		Documents:   docs,
		CreatedAt:   time.Now().UTC(),
	})
}

func searchDocumentFromEvidence(record portalstate.EvidenceRecord) contracts.SearchIndexDocument {
	return contracts.SearchIndexDocument{
		DocumentID:   contracts.StableHash("search", contracts.SearchIndexKindEvidence, record.EvidenceID),
		Kind:         contracts.SearchIndexKindEvidence,
		ExecutionID:  record.ExecutionID,
		JobID:        record.JobID,
		DataStoreID:  record.DataStoreID,
		SourceSystem: record.SourceSystem,
		SubjectType:  record.SubjectType,
		SubjectID:    record.SubjectID,
		FileID:       record.FileID,
		Container:    record.Container,
		Path:         record.Blob,
		Title:        firstNonBlank(record.Blob, record.SubjectID, record.FileID, record.EventType),
		Summary:      record.Summary,
		Status:       record.Status,
		Severity:     record.Severity,
		Action:       record.Operation,
		EvidenceRefs: []string{record.EvidenceID},
		Tags:         []string{"evidence", record.EventType},
		Metadata:     stringMapFromAnyMap(record.Details),
		CreatedAt:    parsePortalTime(record.CreatedAt),
	}
}

func searchDocumentFromAudit(event portalstate.AuditEvent) contracts.SearchIndexDocument {
	return contracts.SearchIndexDocument{
		DocumentID:  contracts.StableHash("search", contracts.SearchIndexKindAudit, event.EventID),
		Kind:        contracts.SearchIndexKindAudit,
		ExecutionID: event.ExecutionID,
		DataStoreID: event.DataStoreID,
		SubjectType: event.TargetType,
		SubjectID:   event.TargetID,
		Title:       firstNonBlank(event.EventType, event.Operation, event.TargetID),
		Summary:     event.Summary,
		Status:      event.Status,
		Action:      event.Operation,
		Tags:        []string{"audit", event.EventType, event.Role},
		Metadata:    stringMapFromAnyMap(event.Details),
		CreatedAt:   parsePortalTime(event.CreatedAt),
	}
}

func searchDocumentFromAction(action portalstate.ActionRecord) contracts.SearchIndexDocument {
	return contracts.SearchIndexDocument{
		DocumentID:   contracts.StableHash("search", contracts.SearchIndexKindAction, action.ActionID),
		Kind:         contracts.SearchIndexKindAction,
		ExecutionID:  action.ExecutionID,
		DataStoreID:  action.Container,
		SubjectType:  "action",
		SubjectID:    action.ActionID,
		FileID:       action.FileID,
		Container:    action.Container,
		Path:         action.BlobPath,
		Title:        firstNonBlank(action.Operation, action.ActionID),
		Summary:      action.Reason,
		Status:       action.Status,
		Severity:     searchSeverityForPortalAction(action.Status),
		Decision:     firstNonBlank(action.GuardianDecision, action.DecisionID),
		Action:       action.Operation,
		EvidenceRefs: []string{action.ActionID},
		ActionRefs:   []string{action.ActionID},
		Tags:         []string{"action", action.ActionMode, action.ApprovalEnforcement, action.GuardianEnforcement},
		Metadata:     stringMapFromAnyMap(action.Details),
		CreatedAt:    parsePortalTime(firstNonBlank(action.UpdatedAt, action.CreatedAt)),
	}
}

func searchDocumentFromDataPassport(passport contracts.DataPassport) contracts.SearchIndexDocument {
	return contracts.SearchIndexDocument{
		DocumentID:          contracts.StableHash("search", contracts.SearchIndexKindDataPassport, passport.PassportID),
		Kind:                contracts.SearchIndexKindDataPassport,
		ExecutionID:         passport.ExecutionID,
		JobID:               passport.JobID,
		BatchID:             passport.BatchID,
		DataStoreID:         passport.DataStoreID,
		SourceSystem:        passport.SourceSystem,
		SubjectType:         passport.SubjectType,
		SubjectID:           firstNonBlank(passport.Subject.URI, passport.Subject.Path, passport.PassportID),
		FileID:              firstNonBlank(passport.Subject.URI, passport.Subject.Path),
		Container:           passport.Subject.Container,
		Path:                passport.Subject.Path,
		Title:               firstNonBlank(passport.Subject.Path, passport.Subject.URI, passport.PassportID),
		Summary:             "Data Passport " + passport.LifecycleStage,
		Status:              passport.LifecycleStage,
		Severity:            searchSeverityForRiskScore(passport.RiskScore),
		Decision:            passport.GuardianDecision,
		Action:              passport.ActionRequested,
		Classification:      passport.Classification,
		RiskScore:           passport.RiskScore,
		Entities:            passport.Entities,
		PolicyBundleVersion: passport.PolicyBundleVersion,
		PolicyBundleHash:    passport.PolicyBundleHash,
		WorkflowVersion:     passport.WorkflowVersion,
		EvidenceRefs:        passport.EvidenceRefs,
		ProvenanceRefs:      passport.ProvenanceRefs,
		PassportRefs:        []string{passport.PassportID},
		ActionRefs:          passport.ActionRefs,
		Tags:                []string{"data_passport", passport.LifecycleStage},
		Metadata:            passport.Metadata,
		CreatedAt:           parsePortalTime(passport.CreatedAt),
	}
}

func stringMapFromAnyMap(input map[string]any) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range input {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func parsePortalTime(value string) time.Time {
	if value == "" {
		return time.Now().UTC()
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed
}

func searchSeverityForPortalAction(status string) string {
	switch actionStatusFromAPIStatus(status) {
	case "blocked", "failed", "rejected":
		return "high"
	case "pending_approval", "planned":
		return "medium"
	default:
		return "info"
	}
}

func searchSeverityForRiskScore(score int) string {
	switch {
	case score >= 90:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 45:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "info"
	}
}
