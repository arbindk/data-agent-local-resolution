package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/execution"
)

type AzureBlobWritebackRequest struct {
	ExecutionID      string            `json:"execution_id"`
	DecisionID       string            `json:"decision_id"`
	FileID           string            `json:"file_id"`
	BlobPath         string            `json:"blob_path,omitempty"`
	ActionType       string            `json:"action_type"`
	ActionMode       string            `json:"action_mode"`
	WritebackEnabled bool              `json:"writeback_enabled"`
	GuardianDecision string            `json:"guardian_decision"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	IndexTags        map[string]string `json:"index_tags,omitempty"`
}

type AzureBlobWritebackResponse struct {
	Status            string                            `json:"status"`
	ExecutionID       string                            `json:"execution_id,omitempty"`
	DecisionID        string                            `json:"decision_id,omitempty"`
	FileID            string                            `json:"file_id,omitempty"`
	ActionType        string                            `json:"action_type"`
	ActionMode        string                            `json:"action_mode"`
	WritebackEnabled  bool                              `json:"writeback_enabled"`
	GuardianDecision  string                            `json:"guardian_decision"`
	Reason            string                            `json:"reason,omitempty"`
	ConnectorResponse any                               `json:"connector_response,omitempty"`
	ActionReceipt     *contracts.ActionReceipt          `json:"action_receipt,omitempty"`
	DACIntent         *execution.IntentEvaluationResult `json:"dac_intent,omitempty"`
	CompletedAt       string                            `json:"completed_at"`
}

func (s *Server) azureBlobWriteback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req AzureBlobWritebackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req.ActionType = strings.TrimSpace(req.ActionType)
	req.ActionMode = strings.TrimSpace(req.ActionMode)
	req.GuardianDecision = strings.TrimSpace(req.GuardianDecision)

	if req.ActionType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action_type is required"})
		return
	}

	if req.ActionMode == "" {
		req.ActionMode = execution.ActionModeDryRun
	}

	if req.GuardianDecision == "" {
		req.GuardianDecision = execution.GuardianDecisionPending
	}
	if req.ActionMode == execution.ActionModeApply && req.WritebackEnabled {
		roles := azureBlobApplyRoles(req.ActionType)
		if !s.authorizeEnterpriseApplyAction(w, r, "azureblob.writeback."+req.ActionType, roles...) {
			return
		}
	}

	var intent *execution.IntentEvaluationResult
	writeResp := func(status int, resp AzureBlobWritebackResponse) {
		resp.DACIntent = intent
		if resp.ActionReceipt == nil {
			receipt := buildAzureBlobWritebackReceipt(req, resp)
			resp.ActionReceipt = &receipt
		}
		writeJSON(w, status, resp)
	}

	containerName, blobName := actionObjectParts(req.FileID, req.BlobPath)
	intent, err := s.evaluateDACIntent(r, dacIntentInput{
		Operation:  req.ActionType,
		SourceID:   s.azureBlobConfig().DataStoreID,
		ActionMode: req.ActionMode,
		Container:  containerName,
		Blob:       blobName,
		Metadata: map[string]string{
			"entrypoint":        "azure_blob_writeback",
			"requested_file_id": req.FileID,
			"metadata_keys":     strconv.Itoa(len(req.Metadata)),
			"index_tag_keys":    strconv.Itoa(len(req.IndexTags)),
		},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if intent.Status != "ok" || intent.GuardianDecision == nil || !intent.GuardianDecision.Approved {
		writeResp(http.StatusOK, AzureBlobWritebackResponse{
			Status:           "blocked",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: dacDecisionString(intent),
			Reason:           "Write-back blocked by Data Agent Cell Orchestrator.",
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	if strings.TrimSpace(req.DecisionID) == "" {
		req.DecisionID = dacDecisionID(intent)
	}
	if req.GuardianDecision == execution.GuardianDecisionPending {
		req.GuardianDecision = dacDecisionString(intent)
	}

	if !execution.CanApplyWriteback(req.ActionMode, req.WritebackEnabled, req.GuardianDecision, req.ActionType) {
		writeResp(http.StatusOK, AzureBlobWritebackResponse{
			Status:           "blocked",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           "Write-back blocked. Requires action_mode=apply, writeback_enabled=true, guardian_decision=allow, and a supported non-destructive Azure Blob action.",
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if strings.TrimSpace(req.DecisionID) == "" {
		writeResp(http.StatusOK, AzureBlobWritebackResponse{
			Status:           "blocked",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           "Write-back blocked. Apply mode requires a valid Guardian decision ID.",
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if ok, reason := s.hasApprovedWritebackApproval(r, req.ExecutionID); !ok {
		writeResp(http.StatusOK, AzureBlobWritebackResponse{
			Status:           "blocked",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           reason,
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	cfg := s.azureBlobConfig()
	if container, err := azureblob.ContainerFromFileID(req.FileID); err == nil && container != "" {
		cfg.ContainerName = container
	}

	scanner, err := s.newAzureBlobScanner(cfg)
	if err != nil {
		writeResp(http.StatusBadRequest, AzureBlobWritebackResponse{
			Status:           "failed",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           err.Error(),
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeReq := azureblob.WritebackRequest{
		FileID:    req.FileID,
		BlobPath:  req.BlobPath,
		Metadata:  req.Metadata,
		IndexTags: req.IndexTags,
	}

	var connectorResult any

	switch req.ActionType {
	case execution.ActionApplyBlobMetadata:
		connectorResult, err = scanner.ApplyMetadata(r.Context(), writeReq)

	case execution.ActionApplyBlobIndexTags:
		connectorResult, err = scanner.ApplyIndexTags(r.Context(), writeReq)

	default:
		writeResp(http.StatusBadRequest, AzureBlobWritebackResponse{
			Status:           "failed",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           "unsupported action_type",
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err != nil {
		writeResp(http.StatusInternalServerError, AzureBlobWritebackResponse{
			Status:           "failed",
			ExecutionID:      req.ExecutionID,
			DecisionID:       req.DecisionID,
			FileID:           req.FileID,
			ActionType:       req.ActionType,
			ActionMode:       req.ActionMode,
			WritebackEnabled: req.WritebackEnabled,
			GuardianDecision: req.GuardianDecision,
			Reason:           err.Error(),
			CompletedAt:      time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeResp(http.StatusOK, AzureBlobWritebackResponse{
		Status:            "completed",
		ExecutionID:       req.ExecutionID,
		DecisionID:        req.DecisionID,
		FileID:            req.FileID,
		ActionType:        req.ActionType,
		ActionMode:        req.ActionMode,
		WritebackEnabled:  req.WritebackEnabled,
		GuardianDecision:  req.GuardianDecision,
		ConnectorResponse: connectorResult,
		CompletedAt:       time.Now().UTC().Format(time.RFC3339),
	})
}

func buildAzureBlobWritebackReceipt(req AzureBlobWritebackRequest, resp AzureBlobWritebackResponse) contracts.ActionReceipt {
	completedAt := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339, resp.CompletedAt); err == nil {
		completedAt = parsed
	}

	decisionID := firstNonBlank(resp.DecisionID, req.DecisionID, "gd-unavailable")
	fileRef := firstNonBlank(resp.FileID, req.FileID, req.BlobPath)
	actionType := azureblob.NormalizeAction(firstNonBlank(resp.ActionType, req.ActionType))
	seed := contracts.StableHash(resp.ExecutionID, decisionID, fileRef, actionType, resp.Status)
	shortID := strings.TrimPrefix(seed, "sha256:")
	if len(shortID) > 16 {
		shortID = shortID[:16]
	}

	objectRefs := []contracts.ObjectRef{}
	if fileRef != "" {
		containerName := ""
		if parsed, err := azureblob.ContainerFromFileID(fileRef); err == nil {
			containerName = parsed
		}
		objectRefs = append(objectRefs, contracts.ObjectRef{
			URI:         firstNonBlank(req.BlobPath, fileRef),
			ConnectorID: "azure_blob",
			SourceID:    "azure_blob",
			Container:   containerName,
			Path:        firstNonBlank(req.BlobPath, fileRef),
		})
	}

	obligations := []string{contracts.ObligationRecordAudit, contracts.ObligationWriteProvenance}
	if resp.GuardianDecision == execution.GuardianDecisionAllow {
		obligations = append(obligations, contracts.ObligationAttachPolicyHash)
	}
	resultSummary := firstNonBlank(resp.Reason, "Azure Blob write-back "+actionType+" "+resp.Status+".")

	return contracts.ActionReceipt{
		ReceiptID:          "receipt-" + shortID,
		ActionID:           "act-" + shortID,
		RequestID:          firstNonBlank(resp.ExecutionID, "req-unavailable"),
		ExecutionID:        resp.ExecutionID,
		DecisionID:         decisionID,
		ConnectorID:        "azure_blob",
		ObjectRefs:         objectRefs,
		ActionType:         actionType,
		Status:             resp.Status,
		Applied:            resp.Status == "completed",
		DryRun:             resp.ActionMode != execution.ActionModeApply || !resp.WritebackEnabled,
		ResultSummary:      resultSummary,
		ObligationsApplied: obligations,
		EvidenceRefs:       []string{"evidence:writeback:" + shortID},
		ProvenanceRefs:     []string{"provenance:" + decisionID},
		StartedAt:          completedAt,
		CompletedAt:        completedAt,
		ReceiptHash:        contracts.StableHash("receipt", shortID, decisionID, actionType, resp.Status, resultSummary),
	}
}
