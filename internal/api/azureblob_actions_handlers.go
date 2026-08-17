package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type AzureBlobActionExecuteRequest struct {
	ExecutionID       string            `json:"execution_id"`
	DecisionID        string            `json:"decision_id,omitempty"`
	FileID            string            `json:"file_id"`
	BlobPath          string            `json:"blob_path,omitempty"`
	Operation         string            `json:"operation"`
	ActionType        string            `json:"action_type,omitempty"`
	ActionMode        string            `json:"action_mode"`
	WritebackEnabled  bool              `json:"writeback_enabled"`
	GuardianDecision  string            `json:"guardian_decision"`
	TargetContainer   string            `json:"target_container,omitempty"`
	TargetPrefix      string            `json:"target_prefix,omitempty"`
	TargetBlobPath    string            `json:"target_blob_path,omitempty"`
	TargetTier        string            `json:"target_tier,omitempty"`
	RehydratePriority string            `json:"rehydrate_priority,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	IndexTags         map[string]string `json:"index_tags,omitempty"`
	Confirm           string            `json:"confirm,omitempty"`
	PreviewOnly       bool              `json:"preview_only,omitempty"`
}

type AzureBlobActionExecuteResponse struct {
	Status              string                            `json:"status"`
	ExecutionID         string                            `json:"execution_id,omitempty"`
	DecisionID          string                            `json:"decision_id,omitempty"`
	FileID              string                            `json:"file_id,omitempty"`
	Operation           string                            `json:"operation"`
	ActionMode          string                            `json:"action_mode"`
	WritebackEnabled    bool                              `json:"writeback_enabled"`
	GuardianDecision    string                            `json:"guardian_decision"`
	Reason              string                            `json:"reason,omitempty"`
	ApprovalEnforcement string                            `json:"approval_enforcement,omitempty"`
	GuardianEnforcement string                            `json:"guardian_enforcement,omitempty"`
	ConnectorResponse   any                               `json:"connector_response,omitempty"`
	ActionReceipt       *contracts.ActionReceipt          `json:"action_receipt,omitempty"`
	DACIntent           *execution.IntentEvaluationResult `json:"dac_intent,omitempty"`
	CompletedAt         string                            `json:"completed_at"`
}

func (s *Server) azureBlobActionExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req AzureBlobActionExecuteRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	operation := strings.TrimSpace(req.Operation)
	if operation == "" {
		operation = strings.TrimSpace(req.ActionType)
	}
	operation = normalizeActionExecuteOperation(operation)
	req.ActionMode = strings.TrimSpace(req.ActionMode)
	req.GuardianDecision = strings.TrimSpace(req.GuardianDecision)

	if operation == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "operation is required"})
		return
	}
	if !execution.IsAzureBlobPortalAction(operation) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported azure blob operation"})
		return
	}
	if strings.TrimSpace(req.FileID) == "" && strings.TrimSpace(req.BlobPath) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file_id or blob_path is required"})
		return
	}
	if req.ActionMode == "" {
		req.ActionMode = execution.ActionModeDryRun
	}
	if req.GuardianDecision == "" {
		req.GuardianDecision = execution.GuardianDecisionPending
	}
	if req.ActionMode == execution.ActionModeApply && req.WritebackEnabled && !req.PreviewOnly {
		roles := azureBlobApplyRoles(operation)
		if !s.authorizeEnterpriseApplyAction(w, r, "azureblob.action."+operation, roles...) {
			return
		}
	}

	containerName, blobName := actionObjectParts(req.FileID, req.BlobPath)
	intentActionMode := req.ActionMode
	if req.PreviewOnly {
		intentActionMode = execution.ActionModeDryRun
	}
	intent, err := s.evaluateDACIntent(r, dacIntentInput{
		Operation:       operation,
		SourceID:        s.azureBlobConfig().DataStoreID,
		ActionMode:      intentActionMode,
		Container:       containerName,
		Blob:            blobName,
		TargetContainer: req.TargetContainer,
		TargetBlob:      req.TargetBlobPath,
		Metadata: map[string]string{
			"entrypoint":        "azure_blob_action_execute",
			"requested_file_id": req.FileID,
			"preview_only":      strconv.FormatBool(req.PreviewOnly),
		},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if intent.Status != "ok" || intent.GuardianDecision == nil || !intent.GuardianDecision.Approved {
		writeJSON(w, http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "blocked",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    dacDecisionString(intent),
			Reason:              "Action blocked by Data Agent Cell Orchestrator.",
			GuardianEnforcement: "orchestrator_intent_required_before_connector_dispatch",
			DACIntent:           intent,
			CompletedAt:         time.Now().UTC().Format(time.RFC3339),
		})
		return
	}
	if strings.TrimSpace(req.DecisionID) == "" {
		req.DecisionID = dacDecisionID(intent)
	}
	if req.GuardianDecision == execution.GuardianDecisionPending {
		req.GuardianDecision = dacDecisionString(intent)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	writeAction := func(status int, resp AzureBlobActionExecuteResponse) {
		resp.DACIntent = intent
		if resp.ActionReceipt == nil {
			receipt := buildAzureBlobActionReceipt(req, resp)
			resp.ActionReceipt = &receipt
		}
		s.persistAzureBlobActionExecution(req, resp)
		writeJSON(w, status, resp)
	}
	if req.PreviewOnly || req.ActionMode != execution.ActionModeApply || !req.WritebackEnabled {
		writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "planned",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              "Dry-run action plan created. Azure Blob was not modified.",
			ApprovalEnforcement: "not_required_for_preview",
			GuardianEnforcement: "required_before_apply",
			ConnectorResponse: map[string]any{
				"operation":          operation,
				"file_id":            req.FileID,
				"blob_path":          req.BlobPath,
				"target_container":   req.TargetContainer,
				"target_prefix":      req.TargetPrefix,
				"target_blob_path":   req.TargetBlobPath,
				"target_tier":        req.TargetTier,
				"rehydrate_priority": req.RehydratePriority,
			},
			CompletedAt: now,
		})
		return
	}

	if !execution.CanApplyAzureBlobPortalAction(req.ActionMode, req.WritebackEnabled, req.GuardianDecision, operation) {
		writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "blocked",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              "Action blocked. Apply mode requires writeback_enabled=true and guardian_decision=allow.",
			ApprovalEnforcement: "not_checked",
			GuardianEnforcement: "required_before_apply",
			CompletedAt:         now,
		})
		return
	}

	if strings.TrimSpace(req.DecisionID) == "" {
		writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "blocked",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              "Action blocked. Apply mode requires a valid Guardian decision ID.",
			ApprovalEnforcement: "not_checked",
			GuardianEnforcement: "decision_id_required_before_apply",
			CompletedAt:         now,
		})
		return
	}

	if reason := restrictedActionConfirmation(operation, req.Confirm); reason != "" {
		writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "blocked",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              reason,
			ApprovalEnforcement: "confirmation_required",
			GuardianEnforcement: "required_before_apply",
			CompletedAt:         now,
		})
		return
	}

	approvalOperations := []string{operation}
	if operation == execution.ActionApplyBlobMetadata || operation == execution.ActionApplyBlobIndexTags {
		approvalOperations = []string{execution.ActionApplyBlobMetadata, execution.ActionApplyBlobIndexTags}
	}

	if ok, reason := s.hasApprovedAzureBlobActionApproval(r, req.ExecutionID, approvalOperations...); !ok {
		writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
			Status:              "blocked",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              reason,
			ApprovalEnforcement: "approved_control_plane_approval_required",
			GuardianEnforcement: "required_before_apply",
			CompletedAt:         now,
		})
		return
	}

	cfg := s.azureBlobConfig()
	if container, err := azureblob.ContainerFromFileID(req.FileID); err == nil && container != "" {
		cfg.ContainerName = container
	}

	scanner, err := s.newAzureBlobScanner(cfg)
	if err != nil {
		writeAction(http.StatusBadRequest, AzureBlobActionExecuteResponse{
			Status:              "failed",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              err.Error(),
			ApprovalEnforcement: "approved_control_plane_approval_verified",
			GuardianEnforcement: "required_before_apply",
			CompletedAt:         time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	result, err := scanner.ExecuteAction(r.Context(), azureblob.ActionRequest{
		ExecutionID:       req.ExecutionID,
		FileID:            req.FileID,
		BlobPath:          req.BlobPath,
		Operation:         operation,
		TargetContainer:   req.TargetContainer,
		TargetPrefix:      req.TargetPrefix,
		TargetBlobPath:    req.TargetBlobPath,
		TargetTier:        req.TargetTier,
		RehydratePriority: req.RehydratePriority,
		Metadata:          req.Metadata,
		IndexTags:         req.IndexTags,
	})
	if err != nil {
		_ = s.recordAzureBlobAction(r, req.ExecutionID, req.DecisionID, req.FileID, operation, "failed", err.Error())
		writeAction(http.StatusInternalServerError, AzureBlobActionExecuteResponse{
			Status:              "failed",
			ExecutionID:         req.ExecutionID,
			DecisionID:          req.DecisionID,
			FileID:              req.FileID,
			Operation:           operation,
			ActionMode:          req.ActionMode,
			WritebackEnabled:    req.WritebackEnabled,
			GuardianDecision:    req.GuardianDecision,
			Reason:              err.Error(),
			ApprovalEnforcement: "approved_control_plane_approval_required",
			GuardianEnforcement: "required_before_apply",
			CompletedAt:         time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	_ = s.recordAzureBlobAction(r, req.ExecutionID, req.DecisionID, req.FileID, operation, "completed", "")
	if req.ExecutionID != "" && req.FileID != "" {
		_ = s.store.UpdateManifestActionStatus(r.Context(), req.ExecutionID, req.FileID, "completed", "")
	}

	writeAction(http.StatusOK, AzureBlobActionExecuteResponse{
		Status:              "completed",
		ExecutionID:         req.ExecutionID,
		DecisionID:          req.DecisionID,
		FileID:              req.FileID,
		Operation:           operation,
		ActionMode:          req.ActionMode,
		WritebackEnabled:    req.WritebackEnabled,
		GuardianDecision:    req.GuardianDecision,
		ApprovalEnforcement: "approved_control_plane_approval_verified",
		GuardianEnforcement: "required_before_apply",
		ConnectorResponse:   result,
		CompletedAt:         time.Now().UTC().Format(time.RFC3339),
	})
}

func buildAzureBlobActionReceipt(req AzureBlobActionExecuteRequest, resp AzureBlobActionExecuteResponse) contracts.ActionReceipt {
	completedAt := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339, resp.CompletedAt); err == nil {
		completedAt = parsed
	}

	decisionID := firstNonBlank(resp.DecisionID, req.DecisionID, "gd-unavailable")
	fileRef := firstNonBlank(resp.FileID, req.FileID, req.BlobPath)
	actionType := azureblob.NormalizeAction(firstNonBlank(resp.Operation, req.Operation, req.ActionType))
	seed := contracts.StableHash(resp.ExecutionID, decisionID, fileRef, actionType, resp.Status)
	shortID := strings.TrimPrefix(seed, "sha256:")
	if len(shortID) > 16 {
		shortID = shortID[:16]
	}

	dryRun := resp.ActionMode != execution.ActionModeApply || !resp.WritebackEnabled || resp.Status == "planned"
	applied := resp.Status == "completed"
	obligations := []string{contracts.ObligationRecordAudit, contracts.ObligationWriteProvenance}
	if resp.GuardianDecision == execution.GuardianDecisionAllow {
		obligations = append(obligations, contracts.ObligationAttachPolicyHash)
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

	resultSummary := firstNonBlank(resp.Reason, "Azure Blob action "+actionType+" "+resp.Status+".")

	return contracts.ActionReceipt{
		ReceiptID:          "receipt-" + shortID,
		ActionID:           "act-" + shortID,
		RequestID:          firstNonBlank(req.ExecutionID, "req-unavailable"),
		ExecutionID:        resp.ExecutionID,
		DecisionID:         decisionID,
		ConnectorID:        "azure_blob",
		ObjectRefs:         objectRefs,
		ActionType:         actionType,
		Status:             resp.Status,
		Applied:            applied,
		DryRun:             dryRun,
		ResultSummary:      resultSummary,
		ObligationsApplied: obligations,
		EvidenceRefs:       []string{"evidence:action_execution:" + shortID},
		ProvenanceRefs:     []string{"provenance:" + decisionID},
		StartedAt:          completedAt,
		CompletedAt:        completedAt,
		ReceiptHash:        contracts.StableHash("receipt", shortID, decisionID, actionType, resp.Status, resultSummary),
	}
}

func actionObjectParts(fileID string, blobPath string) (string, string) {
	if containerName, err := azureblob.ContainerFromFileID(fileID); err == nil {
		parts := strings.SplitN(strings.TrimPrefix(fileID, "azureblob:"), ":", 2)
		if len(parts) == 2 {
			return containerName, strings.Trim(strings.TrimSpace(parts[1]), "/")
		}
		return containerName, strings.Trim(strings.TrimSpace(blobPath), "/")
	}
	return "", strings.Trim(strings.TrimSpace(blobPath), "/")
}

func (s *Server) recordAzureBlobAction(
	r *http.Request,
	executionID string,
	decisionID string,
	fileID string,
	operation string,
	status string,
	errText string,
) error {
	if strings.TrimSpace(executionID) == "" {
		return nil
	}
	actionID := "action-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	return s.store.InsertWritebackAction(
		r.Context(),
		executionID,
		actionID,
		decisionID,
		fileID,
		operation,
		status,
		errText,
	)
}

func (s *Server) persistAzureBlobActionExecution(req AzureBlobActionExecuteRequest, resp AzureBlobActionExecuteResponse) {
	containerName := ""
	if parsed, err := azureblob.ContainerFromFileID(resp.FileID); err == nil {
		containerName = parsed
	}
	details := map[string]any{
		"action_mode":           resp.ActionMode,
		"writeback_enabled":     resp.WritebackEnabled,
		"guardian_decision":     resp.GuardianDecision,
		"approval_enforcement":  resp.ApprovalEnforcement,
		"guardian_enforcement":  resp.GuardianEnforcement,
		"target_container":      req.TargetContainer,
		"target_prefix":         req.TargetPrefix,
		"target_blob_path":      req.TargetBlobPath,
		"target_tier":           req.TargetTier,
		"rehydrate_priority":    req.RehydratePriority,
		"connector_response":    resp.ConnectorResponse,
		"metadata_keys":         len(req.Metadata),
		"index_tag_keys":        len(req.IndexTags),
		"confirmation_provided": strings.TrimSpace(req.Confirm) != "",
	}
	if resp.ActionReceipt != nil {
		details["action_receipt"] = resp.ActionReceipt
	}
	if resp.DACIntent != nil {
		details["dac_intent"] = resp.DACIntent
		if resp.DACIntent.GuardianDecision != nil {
			details["policy_bundle_hash"] = resp.DACIntent.GuardianDecision.PolicyBundleHash
		}
	}
	actionStatus := actionStatusFromAPIStatus(resp.Status)
	s.upsertActionRecord(portalstate.ActionRecord{
		ExecutionID:         resp.ExecutionID,
		DecisionID:          resp.DecisionID,
		FileID:              resp.FileID,
		BlobPath:            firstNonBlank(req.BlobPath, resp.FileID),
		Container:           containerName,
		Operation:           resp.Operation,
		Status:              actionStatus,
		RequestedBy:         "portal-user",
		ActionMode:          resp.ActionMode,
		WritebackEnabled:    resp.WritebackEnabled,
		GuardianDecision:    resp.GuardianDecision,
		ApprovalEnforcement: resp.ApprovalEnforcement,
		GuardianEnforcement: resp.GuardianEnforcement,
		Reason:              resp.Reason,
		Details:             details,
	})
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "action_execution",
		SubjectID:    firstNonBlank(resp.DecisionID, resp.FileID, resp.Operation),
		EventType:    "action." + resp.Operation + "." + actionStatus,
		Status:       actionStatus,
		Severity:     actionSeverity(actionStatus, resp.Operation),
		SourceSystem: "azure_blob",
		ExecutionID:  resp.ExecutionID,
		Container:    containerName,
		Blob:         req.BlobPath,
		FileID:       resp.FileID,
		Operation:    resp.Operation,
		Summary:      firstNonBlank(resp.Reason, "Azure Blob action "+resp.Operation+" "+actionStatus+"."),
		Details:      details,
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:   "action." + resp.Operation + "." + actionStatus,
		Actor:       "portal-user",
		Status:      actionStatus,
		TargetType:  "azure_blob",
		TargetID:    firstNonBlank(resp.FileID, req.BlobPath),
		ExecutionID: resp.ExecutionID,
		Operation:   resp.Operation,
		DryRun:      resp.ActionMode != execution.ActionModeApply || !resp.WritebackEnabled || resp.Status == "planned",
		Summary:     firstNonBlank(resp.Reason, "Azure Blob action "+resp.Operation+" "+actionStatus+"."),
		Details:     details,
	})
}

func actionSeverity(status string, operation string) string {
	if status == "failed" || status == "blocked" {
		return "high"
	}
	if execution.IsDestructiveOrRestrictedAction(operation) {
		return "high"
	}
	return "info"
}

func restrictedActionConfirmation(operation string, confirm string) string {
	operation = normalizeActionExecuteOperation(operation)
	confirm = strings.ToUpper(strings.TrimSpace(confirm))

	if operation == execution.ActionDelete {
		if confirm != "DELETE" {
			return "Delete requires confirmation text DELETE."
		}
		return ""
	}

	if execution.IsDestructiveOrRestrictedAction(operation) && confirm != "APPLY" {
		return "Restricted Azure Blob actions require confirmation text APPLY."
	}

	return ""
}

func normalizeActionExecuteOperation(operation string) string {
	operation = azureblob.NormalizeAction(strings.ReplaceAll(operation, "-", "_"))
	switch operation {
	case "metadata", "set_metadata", "apply_metadata", "apply_blob_metadata":
		return execution.ActionApplyBlobMetadata
	case "tags", "set_tags", "apply_tags", "apply_index_tags", "apply_blob_index_tags":
		return execution.ActionApplyBlobIndexTags
	case "archive", "set_tier", "set_access_tier", "tier":
		return execution.ActionArchive
	case "retrieve", "rehydrate", "rehydrate_blob":
		return execution.ActionRehydrate
	case "quarantine":
		return execution.ActionQuarantine
	case "copy", "copy_blob", "copy_object", "migration", "move", "move_blob", "move_object", "move_prep":
		return execution.ActionMigration
	case "delete", "delete_blob":
		return execution.ActionDelete
	default:
		return operation
	}
}
func (s *Server) hasApprovedAzureBlobActionApproval(r *http.Request, executionID string, operations ...string) (bool, string) {
	if strings.TrimSpace(executionID) == "" {
		return false, "Approved HITL request is required before Azure Blob action execution, but execution_id is missing."
	}

	platformURL := os.Getenv("EVEREST_PLATFORM_URL")
	if platformURL == "" {
		platformURL = "http://localhost:9090"
	}

	client := controlplane.NewClient(platformURL)
	approvals, err := client.ListApprovals(r.Context(), "APPROVED", executionID)
	if err != nil {
		return false, "Approved HITL request could not be verified from Control Plane: " + err.Error()
	}

	allowed := map[string]bool{}
	for _, op := range operations {
		op = azureblob.NormalizeAction(op)
		if op != "" {
			allowed[op] = true
		}
	}

	for _, approval := range approvals {
		if allowed[azureblob.NormalizeAction(approval.Operation)] ||
			allowed[azureblob.NormalizeAction(approval.ActionType)] {
			return true, ""
		}
	}

	return false, "Approved HITL request is required before Azure Blob " + strings.Join(operations, "/") + " action."
}
