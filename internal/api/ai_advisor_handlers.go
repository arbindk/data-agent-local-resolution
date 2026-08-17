package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	aicatalog "everest.local/data-agent-policy-resolver/internal/ai/catalog"
	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type AzureBlobAIAdvisorRequest struct {
	ExecutionID         string `json:"execution_id"`
	TenantID            string `json:"tenant_id"`
	DataStoreID         string `json:"data_store_id"`
	Actor               string `json:"actor"`
	Role                string `json:"role"`
	Prompt              string `json:"prompt"`
	SourceRegion        string `json:"source_region"`
	TargetRegion        string `json:"target_region"`
	ContractOwner       string `json:"contract_owner"`
	ProviderID          string `json:"provider_id"`
	Model               string `json:"model"`
	Limit               int    `json:"limit,omitempty"`
	AutoSubmitApprovals bool   `json:"auto_submit_approvals"`
}

type AzureBlobAIRecommendation struct {
	RecommendationID string  `json:"recommendation_id"`
	ProviderID       string  `json:"provider_id"`
	Operation        string  `json:"operation"`
	FileID           string  `json:"file_id,omitempty"`
	Path             string  `json:"path,omitempty"`
	RiskLevel        string  `json:"risk_level,omitempty"`
	Classification   string  `json:"classification,omitempty"`
	GuardianDecision string  `json:"guardian_decision,omitempty"`
	Confidence       float64 `json:"confidence"`
	RequiresHITL     bool    `json:"requires_hitl"`
	AutomationMode   string  `json:"automation_mode"`
	Rationale        string  `json:"rationale"`
	ApprovalID       string  `json:"approval_id,omitempty"`
	Status           string  `json:"status"`
}

func (s *Server) azureBlobAIRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req AzureBlobAIAdvisorRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req.ExecutionID = strings.TrimSpace(req.ExecutionID)
	if req.ExecutionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id is required for AI recommendations"})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 12
	}
	if strings.TrimSpace(req.TenantID) == "" {
		req.TenantID = "tenant-local-001"
	}
	if strings.TrimSpace(req.Actor) == "" {
		req.Actor = "ai-copilot"
	}
	if strings.TrimSpace(req.Role) == "" {
		req.Role = "data_steward"
	}
	if strings.TrimSpace(req.DataStoreID) == "" {
		req.DataStoreID = s.azureBlobConfig().DataStoreID
	}
	if strings.TrimSpace(req.SourceRegion) == "" {
		req.SourceRegion = "IN"
	}
	if strings.TrimSpace(req.TargetRegion) == "" {
		req.TargetRegion = req.SourceRegion
	}

	providerCfg := s.withAIProviderManifestDefaults(s.aiProviderConfig())
	if strings.TrimSpace(req.ProviderID) != "" {
		providerCfg.ProviderID = strings.TrimSpace(req.ProviderID)
		providerCfg = s.withAIProviderManifestDefaults(providerCfg)
	}
	if strings.TrimSpace(req.Model) != "" {
		providerCfg.Model = strings.TrimSpace(req.Model)
	}
	providerManifest, providerKnown := aicatalog.FindProvider(aicatalog.LoadCatalog(aiProviderManifestDirs()...), providerCfg.ProviderID)
	if !providerKnown {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI provider is not configured in catalog: " + providerCfg.ProviderID})
		return
	}

	actions, _, err := s.store.ListActionPlan(r.Context(), req.ExecutionID, req.Limit, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	recommendations := make([]AzureBlobAIRecommendation, 0, len(actions))
	for i, item := range actions {
		recommendations = append(recommendations, buildAzureBlobAIRecommendation(req, item, i, providerCfg.ProviderID))
	}

	approvalStatus := map[string]string{}
	if req.AutoSubmitApprovals {
		if !providerCfg.AllowHITLSubmission || providerCfg.AutomationMode != "submit_hitl_approvals" {
			approvalStatus["provider_policy"] = "blocked: selected AI provider configuration is recommend-only for HITL submission"
		} else {
			approvalStatus = submitAIApprovalRequests(r, req, recommendations)
			for i := range recommendations {
				if approvalID := approvalStatus[recommendations[i].RecommendationID]; approvalID != "" && !strings.HasPrefix(approvalID, "failed:") {
					recommendations[i].ApprovalID = approvalID
					recommendations[i].Status = "pending_approval"
					recommendations[i].AutomationMode = "hitl_submitted"
				}
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, AzureBlobAIRecommendation{
			RecommendationID: "ai-rec-empty",
			ProviderID:       providerCfg.ProviderID,
			Operation:        "metadata_scan",
			Confidence:       0.5,
			RequiresHITL:     false,
			AutomationMode:   "suggested",
			Rationale:        "No action plan rows are available yet. Run a governed Azure Blob scan so AI can reason over actual evidence.",
			Status:           "scan_required",
		})
	}
	s.persistAIRecommendations(req, providerCfg.ProviderID, recommendations, approvalStatus)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"advisor": "everest_ai_policy_copilot",
		"ai_provider": map[string]any{
			"provider_id":                 providerCfg.ProviderID,
			"display_name":                providerManifest.DisplayName,
			"provider_type":               providerManifest.ProviderType,
			"model":                       providerCfg.Model,
			"deployment_name":             providerCfg.DeploymentName,
			"network_mode":                providerCfg.NetworkMode,
			"automation_mode":             providerCfg.AutomationMode,
			"prompt_audit_enabled":        providerCfg.PromptAuditEnabled,
			"evidence_grounding_required": providerCfg.EvidenceGroundingRequired,
			"generation_mode":             "evidence_grounded_policy_engine",
		},
		"execution_id":        req.ExecutionID,
		"recommendations":     recommendations,
		"approval_submission": approvalStatus,
		"controls": []string{
			"evidence_driven_recommendations",
			"guardian_decision_respected",
			"hitl_required_before_apply",
			"no_silent_azure_mutation",
			"ai_provider_policy_respected",
			"no_customer_prompt_sent_during_provider_test",
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) persistAIRecommendations(req AzureBlobAIAdvisorRequest, providerID string, recommendations []AzureBlobAIRecommendation, approvalStatus map[string]string) {
	for _, rec := range recommendations {
		status := rec.Status
		if rec.ApprovalID != "" {
			status = "pending_approval"
		}
		s.recordEvidence(portalstate.EvidenceRecord{
			SubjectType:  "ai_recommendation",
			SubjectID:    rec.RecommendationID,
			EventType:    "ai.recommendation.created",
			Status:       status,
			Severity:     strings.ToLower(firstNonBlank(rec.RiskLevel, "info")),
			SourceSystem: "azure_blob",
			DataStoreID:  req.DataStoreID,
			ExecutionID:  req.ExecutionID,
			Blob:         rec.Path,
			FileID:       rec.FileID,
			Operation:    rec.Operation,
			ProviderID:   providerID,
			Summary:      rec.Rationale,
			Details: map[string]any{
				"recommendation_id":   rec.RecommendationID,
				"confidence":          rec.Confidence,
				"classification":      rec.Classification,
				"risk_level":          rec.RiskLevel,
				"guardian_decision":   rec.GuardianDecision,
				"requires_hitl":       rec.RequiresHITL,
				"automation_mode":     rec.AutomationMode,
				"approval_id":         rec.ApprovalID,
				"approval_submission": approvalStatus[rec.RecommendationID],
			},
		})
		s.recordAudit(portalstate.AuditEvent{
			EventType:   "ai.recommendation.created",
			Actor:       firstNonBlank(req.Actor, "ai-copilot"),
			Role:        req.Role,
			Status:      status,
			TargetType:  "source_item",
			TargetID:    firstNonBlank(rec.FileID, rec.Path),
			DataStoreID: req.DataStoreID,
			ExecutionID: req.ExecutionID,
			Operation:   rec.Operation,
			DryRun:      true,
			Summary:     rec.Rationale,
			Details: map[string]any{
				"provider_id":         providerID,
				"recommendation_id":   rec.RecommendationID,
				"confidence":          rec.Confidence,
				"requires_hitl":       rec.RequiresHITL,
				"approval_submission": approvalStatus[rec.RecommendationID],
			},
		})
		if rec.Operation != "" && rec.Operation != "metadata_scan" {
			s.upsertActionRecord(portalstate.ActionRecord{
				RecommendationID: rec.RecommendationID,
				ApprovalID:       rec.ApprovalID,
				ExecutionID:      req.ExecutionID,
				FileID:           rec.FileID,
				BlobPath:         rec.Path,
				Operation:        rec.Operation,
				Status:           actionStatusFromAPIStatus(status),
				RequestedBy:      firstNonBlank(req.Actor, "ai-copilot"),
				ProviderID:       providerID,
				ActionMode:       "dry_run",
				GuardianDecision: rec.GuardianDecision,
				Reason:           rec.Rationale,
				Details: map[string]any{
					"risk_level":     rec.RiskLevel,
					"classification": rec.Classification,
					"confidence":     rec.Confidence,
					"requires_hitl":  rec.RequiresHITL,
					"source":         "ai_recommendation",
				},
			})
		}
	}
}

func buildAzureBlobAIRecommendation(req AzureBlobAIAdvisorRequest, item duckdbstore.ActionPlanView, idx int, providerID string) AzureBlobAIRecommendation {
	operation := recommendedOperation(req.Prompt, item)
	requiresHITL := operation != "metadata_scan"
	guardian := strings.TrimSpace(item.GuardianDecision)
	if guardian == "" {
		guardian = execution.GuardianDecisionPending
	}

	confidence := 0.72
	switch strings.ToLower(item.RiskLevel) {
	case "critical":
		confidence = 0.94
	case "high":
		confidence = 0.88
	case "medium":
		confidence = 0.76
	}

	status := "recommended"
	mode := "suggested"
	if requiresHITL {
		status = "needs_approval"
		mode = "hitl_required"
	}
	if guardian != execution.GuardianDecisionAllow {
		status = "guardian_review"
		mode = "review_only"
	}

	return AzureBlobAIRecommendation{
		RecommendationID: "ai-rec-" + req.ExecutionID + "-" + time.Now().UTC().Format("150405") + "-" + strings.TrimSpace(item.ProvenanceID) + "-" + string(rune('a'+idx%26)),
		ProviderID:       providerID,
		Operation:        operation,
		FileID:           item.FileID,
		Path:             item.Path,
		RiskLevel:        item.RiskLevel,
		Classification:   item.Classification,
		GuardianDecision: guardian,
		Confidence:       confidence,
		RequiresHITL:     requiresHITL,
		AutomationMode:   mode,
		Rationale:        recommendationRationale(operation, item),
		Status:           status,
	}
}

func recommendedOperation(prompt string, item duckdbstore.ActionPlanView) string {
	intent := strings.ToLower(prompt)
	switch {
	case strings.Contains(intent, "delete"):
		return execution.ActionDelete
	case strings.Contains(intent, "migrat") || strings.Contains(intent, "copy"):
		return execution.ActionMigration
	case strings.Contains(intent, "archive"):
		return execution.ActionArchive
	case strings.Contains(intent, "retrieve") || strings.Contains(intent, "rehydrate"):
		return execution.ActionRetrieve
	case strings.Contains(intent, "quarantine"):
		return execution.ActionQuarantine
	case strings.Contains(intent, "index"):
		return execution.ActionApplyBlobIndexTags
	case strings.Contains(intent, "tag") || strings.Contains(intent, "classif"):
		return execution.ActionApplyBlobMetadata
	}

	action := strings.TrimSpace(item.ActionRequested)
	if action == "" || action == "none" {
		return execution.ActionApplyBlobMetadata
	}

	if strings.Contains(strings.ToLower(item.Classification+" "+item.Path), "confidential") &&
		(strings.EqualFold(item.RiskLevel, "critical") || strings.EqualFold(item.RiskLevel, "high")) {
		return execution.ActionQuarantine
	}

	return action
}

func recommendationRationale(operation string, item duckdbstore.ActionPlanView) string {
	switch operation {
	case execution.ActionQuarantine:
		return "AI selected quarantine because this object is high-impact evidence and should be isolated before broader remediation."
	case execution.ActionMigration:
		return "AI selected migration/copy because the requested intent involves moving or staging the object into a governed target location."
	case execution.ActionArchive:
		return "AI selected archive because the object can be moved to a lower-access tier after customer approval."
	case execution.ActionRetrieve:
		return "AI selected retrieve/rehydrate because the object may need to be restored to an accessible tier."
	case execution.ActionDelete:
		return "AI selected delete only as a controlled recommendation; execution remains blocked until explicit HITL and DELETE confirmation."
	case execution.ActionApplyBlobIndexTags:
		return "AI selected index tags so Azure-native search and lifecycle policies can use Everest classification signals."
	default:
		return "AI selected metadata tagging from the current action plan, classification, risk level, and Guardian decision."
	}
}

func submitAIApprovalRequests(r *http.Request, req AzureBlobAIAdvisorRequest, recommendations []AzureBlobAIRecommendation) map[string]string {
	status := map[string]string{}

	platformURL := os.Getenv("EVEREST_PLATFORM_URL")
	if platformURL == "" {
		platformURL = "http://localhost:9090"
	}
	client := controlplane.NewClient(platformURL)

	for _, rec := range recommendations {
		if !rec.RequiresHITL {
			continue
		}
		approval, err := client.CreateApproval(r.Context(), controlplane.ApprovalCreateRequest{
			TenantID:      req.TenantID,
			RequestedBy:   req.Actor,
			Role:          advisorRoleForOperation(rec.Operation, req.Role),
			Operation:     rec.Operation,
			ActionType:    rec.Operation,
			DataStoreID:   req.DataStoreID,
			ExecutionID:   req.ExecutionID,
			SourceRegion:  req.SourceRegion,
			TargetRegion:  req.TargetRegion,
			ContractOwner: req.ContractOwner,
			PolicyIDs: []string{
				"regional-data-residency",
				"destructive-action-safety",
				"azureblob-approved-tagging",
			},
			Reason: "AI Copilot recommended " + rec.Operation + " for " + rec.Path + ". " + rec.Rationale,
			Metadata: map[string]any{
				"source":            "everest_ai_policy_copilot",
				"recommendation_id": rec.RecommendationID,
				"file_id":           rec.FileID,
				"path":              rec.Path,
				"risk_level":        rec.RiskLevel,
				"classification":    rec.Classification,
				"confidence":        rec.Confidence,
			},
		})
		if err != nil {
			status[rec.RecommendationID] = "failed: " + err.Error()
			continue
		}
		status[rec.RecommendationID] = approval.ApprovalID
	}

	return status
}

func advisorRoleForOperation(operation string, fallback string) string {
	switch operation {
	case execution.ActionArchive, execution.ActionRetrieve, execution.ActionRehydrate, execution.ActionMigration, execution.ActionDelete:
		return "contract_owner"
	case execution.ActionQuarantine:
		return "security_officer"
	case execution.ActionApplyBlobMetadata, execution.ActionApplyBlobIndexTags:
		return "data_steward"
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "data_steward"
}
