package api

import (
	"strings"

	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) recordEvidence(record portalstate.EvidenceRecord) portalstate.EvidenceRecord {
	if s.portalState == nil {
		return record
	}
	recorded := s.portalState.RecordEvidence(record)
	s.publishSearchDocuments(searchDocumentFromEvidence(recorded))
	return recorded
}

func (s *Server) recordAudit(event portalstate.AuditEvent) portalstate.AuditEvent {
	if s.portalState == nil {
		return event
	}
	recorded := s.portalState.RecordAudit(event)
	s.publishSearchDocuments(searchDocumentFromAudit(recorded))
	return recorded
}

func (s *Server) upsertActionRecord(action portalstate.ActionRecord) portalstate.ActionRecord {
	if s.portalState == nil {
		return action
	}
	recorded := s.portalState.UpsertAction(action)
	s.publishSearchDocuments(searchDocumentFromAction(recorded))
	return recorded
}

func actionStatusFromAPIStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "planned", "preview":
		return "planned"
	case "blocked":
		return "blocked"
	case "completed", "ok":
		return "executed"
	case "failed", "error":
		return "failed"
	case "pending", "pending_approval", "needs_approval", "guardian_review":
		return "pending_approval"
	case "rejected":
		return "rejected"
	case "approved":
		return "approved"
	default:
		if strings.TrimSpace(status) == "" {
			return "draft"
		}
		return strings.TrimSpace(status)
	}
}

func severityFromRiskScore(score int) string {
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
