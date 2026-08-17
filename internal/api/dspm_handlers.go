package api

import (
	"net/http"

	"everest.local/data-agent-policy-resolver/internal/dspm"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) dspmCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"tiers":          []string{dspm.TierMetadata, dspm.TierRules, dspm.TierModel, dspm.TierDeep},
		"checks":         dspm.SupportedChecks(),
		"model_profiles": dspm.ModelProfiles(),
		"keyword_packs":  dspm.KeywordPacks(),
		"entity_types":   dspm.EntityTypes(),
	})
}

func (s *Server) dspmContentScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req dspm.Request
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	result := dspm.Scan(req)
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "dspm_scan",
		SubjectID:    result.ScanID,
		EventType:    "dspm.content_scan.completed",
		Status:       result.Status,
		Severity:     severityFromRiskScore(result.RiskScore),
		SourceSystem: firstNonBlank(req.SourceSystem, "azure_blob"),
		Container:    req.Container,
		Blob:         req.Path,
		FileID:       req.FileID,
		Operation:    "content_scan",
		Summary:      result.Summary,
		Details: map[string]any{
			"scan_id":        result.ScanID,
			"scan_tier":      result.ScanTier,
			"model_profile":  result.ModelProfile,
			"classification": result.Classification,
			"confidence":     result.Confidence,
			"risk_score":     result.RiskScore,
			"entities":       len(result.Entities),
			"checks":         len(result.Checks),
			"requires_hitl":  result.RequiresHITL,
			"next_tier":      result.NextTier,
		},
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:  "dspm.content_scan.completed",
		Actor:      "portal-user",
		Status:     result.Status,
		TargetType: "source_item",
		TargetID:   firstNonBlank(req.FileID, req.Container+"/"+req.Path),
		Operation:  "content_scan",
		DryRun:     true,
		Summary:    result.Summary,
		Details: map[string]any{
			"scan_id":        result.ScanID,
			"classification": result.Classification,
			"risk_score":     result.RiskScore,
			"requires_hitl":  result.RequiresHITL,
		},
	})
	writeJSON(w, http.StatusOK, result)
}
