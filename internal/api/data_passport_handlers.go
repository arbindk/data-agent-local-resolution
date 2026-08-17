package api

import (
	"net/http"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type DataPassportMaterializeRequest struct {
	ExecutionID      string            `json:"execution_id"`
	FileID           string            `json:"file_id,omitempty"`
	PolicyBundleHash string            `json:"policy_bundle_hash,omitempty"`
	SourceRegion     string            `json:"source_region,omitempty"`
	TargetRegion     string            `json:"target_region,omitempty"`
	DataMovement     string            `json:"data_movement,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func (s *Server) dataPassports(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.portalState == nil {
			writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "count": 0, "items": []contracts.DataPassport{}})
			return
		}
		items := s.portalState.ListDataPassports(portalstate.DataPassportFilter{
			ExecutionID: strings.TrimSpace(r.URL.Query().Get("execution_id")),
			FileID:      strings.TrimSpace(r.URL.Query().Get("file_id")),
			DataStoreID: strings.TrimSpace(r.URL.Query().Get("data_store_id")),
			Limit:       queryInt(r, "limit", 100),
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})
	case http.MethodPost:
		s.materializeDataPassport(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) dataPassportByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.portalState == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data passport store is not configured"})
		return
	}
	passportID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/data-passports/"), "/")
	if passportID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "passport_id required"})
		return
	}
	passport, ok := s.portalState.GetDataPassport(passportID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "data passport not found"})
		return
	}
	writeJSON(w, http.StatusOK, passport)
}

func (s *Server) materializeDataPassport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.authorizeEnterpriseAction(w, r, "enterprise.data_passport.materialize", "platform_admin", "security_officer", "compliance_officer", "data_steward", "auditor") {
		return
	}
	if s.portalState == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "data passport store is not configured"})
		return
	}

	var req DataPassportMaterializeRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	req.ExecutionID = strings.TrimSpace(req.ExecutionID)
	req.FileID = strings.TrimSpace(req.FileID)
	if req.ExecutionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id is required"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "local evidence store is not configured"})
		return
	}

	summary, err := s.store.GetManifestSummary(r.Context(), req.ExecutionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	details, _, err := s.store.ListManifestDetails(r.Context(), req.ExecutionID, 1000, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var selected *contracts.ManifestDetail
	for i := range details {
		if req.FileID == "" {
			continue
		}
		if details[i].FileID == req.FileID || details[i].Path == req.FileID {
			selected = &details[i]
			break
		}
	}
	if req.FileID != "" && selected == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest detail not found for file_id=" + req.FileID})
		return
	}

	passport := s.buildDataPassport(req, *summary, selected)
	passport = s.portalState.UpsertDataPassport(passport)
	s.publishSearchDocuments(searchDocumentFromDataPassport(passport))
	s.recordDataPassportEvidence(passport)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "materialized",
		"data_passport": passport,
	})
}

func (s *Server) buildDataPassport(req DataPassportMaterializeRequest, summary contracts.ManifestSummary, detail *contracts.ManifestDetail) contracts.DataPassport {
	now := time.Now().UTC()
	subjectType := contracts.DataPassportSubjectExecution
	subject := contracts.ObjectRef{
		URI:         "execution:" + summary.ExecutionID,
		ConnectorID: firstNonBlank(summary.SourceSystem, "everest"),
		SourceID:    summary.DataStoreID,
	}
	classification := ""
	riskLevel := ""
	riskScore := 0
	entities := []string{}
	guardianDecision := ""
	actionRequested := ""
	actionStatus := ""
	provenanceRefs := []string{}

	if detail != nil {
		subjectType = contracts.DataPassportSubjectObject
		subject = contracts.ObjectRef{
			URI:         firstNonBlank(detail.FileID, detail.Path),
			ConnectorID: firstNonBlank(detail.SourceSystem, summary.SourceSystem, "azure_blob"),
			SourceID:    detail.DataStoreID,
			Path:        detail.Path,
			SizeBytes:   detail.SizeBytes,
			ContentType: detail.ContentType,
		}
		classification = detail.Classification
		riskLevel = detail.RiskLevel
		riskScore = detail.RiskScore
		entities = append([]string(nil), detail.Entities...)
		guardianDecision = detail.GuardianDecision
		actionRequested = detail.ActionRequested
		actionStatus = detail.ActionStatus
		if detail.ProvenanceID != "" {
			provenanceRefs = append(provenanceRefs, "provenance:"+detail.ProvenanceID)
		}
	}

	actions := matchingPassportActions(s.portalState.ListActions(500), summary.ExecutionID, subject.URI, subject.Path)
	evidence := matchingPassportEvidence(s.portalState.ListEvidence(500), summary.ExecutionID, subject.URI, subject.Path)
	actionRefs := make([]string, 0, len(actions))
	evidenceRefs := make([]string, 0, len(evidence))
	lastAction := ""
	lastActionStatus := ""
	decisionID := ""
	for _, action := range actions {
		actionRefs = append(actionRefs, "action:"+action.ActionID)
		if lastAction == "" {
			lastAction = action.Operation
			lastActionStatus = action.Status
			decisionID = action.DecisionID
		}
		if action.DecisionID != "" {
			provenanceRefs = append(provenanceRefs, "provenance:"+action.DecisionID)
		}
	}
	for _, item := range evidence {
		evidenceRefs = append(evidenceRefs, "evidence:"+item.EvidenceID)
	}

	stage := dataPassportStage(classification, actionStatus, lastActionStatus)
	policyHash := firstNonBlank(req.PolicyBundleHash, s.cfg.ActiveCompiledPolicyHash)
	obligations := []string{contracts.ObligationRecordAudit, contracts.ObligationWriteProvenance}
	if policyHash != "" {
		obligations = append(obligations, contracts.ObligationAttachPolicyHash)
	}

	metadata := map[string]string{
		"materialized_from": "manifest_evidence",
		"record_count":      firstNonBlank(req.Metadata["record_count"], ""),
	}
	for k, v := range req.Metadata {
		if strings.TrimSpace(k) != "" {
			metadata[k] = v
		}
	}
	if metadata["record_count"] == "" {
		delete(metadata, "record_count")
	}

	passportID := "passport-" + strings.TrimPrefix(contracts.StableHash(summary.ExecutionID, subject.URI, subject.Path, policyHash), "sha256:")[:20]
	return contracts.DataPassport{
		PassportID:          passportID,
		SchemaVersion:       contracts.DataPassportSchemaVersion,
		SubjectType:         subjectType,
		Subject:             subject,
		ExecutionID:         summary.ExecutionID,
		JobID:               summary.JobID,
		BatchID:             summary.BatchID,
		DataStoreID:         summary.DataStoreID,
		SourceSystem:        summary.SourceSystem,
		LifecycleStage:      stage,
		Classification:      classification,
		RiskScore:           riskScore,
		RiskLevel:           riskLevel,
		Entities:            entities,
		PolicyBundleVersion: summary.CompiledPolicyVersion,
		PolicyBundleHash:    policyHash,
		WorkflowVersion:     summary.WorkflowVersion,
		GuardianDecision:    guardianDecision,
		DecisionID:          decisionID,
		ActionRequested:     actionRequested,
		ActionStatus:        actionStatus,
		LastAction:          lastAction,
		LastActionStatus:    lastActionStatus,
		Locality: contracts.DataPassportLocality{
			ExecutionBoundary: "customer_controlled_boundary",
			SourceRegion:      firstNonBlank(req.SourceRegion, "customer_region"),
			TargetRegion:      strings.TrimSpace(req.TargetRegion),
			DataMovement:      firstNonBlank(req.DataMovement, "local_only"),
			ZeroCopyMode:      "reference_stream_range_or_bounded_preview",
		},
		Obligations:    obligations,
		EvidenceRefs:   uniqueStrings(evidenceRefs),
		ProvenanceRefs: uniqueStrings(provenanceRefs),
		ActionRefs:     uniqueStrings(actionRefs),
		Metadata:       metadata,
		CreatedAt:      now.Format(time.RFC3339),
	}.WithDefaults(now)
}

func (s *Server) recordDataPassportEvidence(passport contracts.DataPassport) {
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "data_passport",
		SubjectID:    passport.PassportID,
		EventType:    "data_passport.materialized",
		Status:       "materialized",
		Severity:     "info",
		SourceSystem: passport.SourceSystem,
		DataStoreID:  passport.DataStoreID,
		ExecutionID:  passport.ExecutionID,
		FileID:       passport.Subject.URI,
		Operation:    "materialize_data_passport",
		Summary:      "Data Passport materialized for " + passport.SubjectType + ".",
		Details: map[string]any{
			"passport_hash":   passport.PassportHash,
			"lifecycle_stage": passport.LifecycleStage,
			"policy_hash":     passport.PolicyBundleHash,
		},
	})
}

func matchingPassportActions(actions []portalstate.ActionRecord, executionID string, fileID string, path string) []portalstate.ActionRecord {
	out := []portalstate.ActionRecord{}
	for _, action := range actions {
		if executionID != "" && action.ExecutionID != executionID {
			continue
		}
		if fileID != "" && action.FileID != "" && action.FileID != fileID {
			continue
		}
		if path != "" && action.BlobPath != "" && action.BlobPath != path {
			continue
		}
		out = append(out, action)
	}
	return out
}

func matchingPassportEvidence(items []portalstate.EvidenceRecord, executionID string, fileID string, path string) []portalstate.EvidenceRecord {
	out := []portalstate.EvidenceRecord{}
	for _, item := range items {
		if executionID != "" && item.ExecutionID != executionID && item.JobID != executionID {
			continue
		}
		if fileID != "" && item.FileID != "" && item.FileID != fileID {
			continue
		}
		if path != "" && item.Blob != "" && item.Blob != path {
			continue
		}
		out = append(out, item)
	}
	return out
}

func dataPassportStage(classification string, actionStatus string, lastActionStatus string) string {
	status := strings.ToLower(firstNonBlank(lastActionStatus, actionStatus))
	switch {
	case strings.Contains(status, "executed") || strings.Contains(status, "completed"):
		return contracts.DataPassportStageActionApplied
	case strings.Contains(status, "block") || strings.Contains(status, "reject") || strings.Contains(status, "fail"):
		return contracts.DataPassportStageBlocked
	case strings.Contains(status, "pending") || strings.Contains(status, "approval"):
		return contracts.DataPassportStagePendingApproval
	case strings.Contains(status, "planned") || status == "draft":
		return contracts.DataPassportStageActionPlanned
	case strings.TrimSpace(classification) != "":
		return contracts.DataPassportStageClassified
	default:
		return contracts.DataPassportStageObserved
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
