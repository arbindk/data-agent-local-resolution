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
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
)

type ApplyApprovedWritebacksRequest struct {
	ExecutionID      string `json:"execution_id"`
	ActionMode       string `json:"action_mode"`
	WritebackEnabled bool   `json:"writeback_enabled"`
	Limit            int    `json:"limit,omitempty"`
}

type ApplyApprovedWritebacksResponse struct {
	Status      string `json:"status"`
	ExecutionID string `json:"execution_id"`
	Checked     int    `json:"checked"`
	Applied     int    `json:"applied"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	Results     []any  `json:"results"`
	CompletedAt string `json:"completed_at"`
}

func (s *Server) applyApprovedAzureBlobWritebacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ApplyApprovedWritebacksRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req.ExecutionID = strings.TrimSpace(req.ExecutionID)
	if req.ExecutionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id is required"})
		return
	}

	result := s.applyApprovedAzureBlobWritebacksInternal(
		r,
		req.ExecutionID,
		req.ActionMode,
		req.WritebackEnabled,
		req.Limit,
	)

	writeJSON(w, http.StatusOK, result)
}

func uploadWritebackResultsToControlPlane(
	r *http.Request,
	s *Server,
	executionID string,
) map[string]string {
	status := map[string]string{}

	platformURL := os.Getenv("EVEREST_PLATFORM_URL")
	if platformURL == "" {
		platformURL = "http://localhost:9090"
	}

	client := controlplane.NewClient(platformURL)

	actions, _, err := s.store.ListActionPlan(r.Context(), executionID, 5000, 0)
	if err != nil {
		status["action_results"] = "failed: " + err.Error()
		return status
	}

	if err := client.UploadActionResults(r.Context(), executionID, actions); err != nil {
		status["action_results"] = "failed: " + err.Error()
		return status
	}

	status["action_results"] = "uploaded"
	return status
}

func (s *Server) applyApprovedAzureBlobWritebacksInternal(
	r *http.Request,
	executionID string,
	actionMode string,
	writebackEnabled bool,
	limit int,
) map[string]any {
	if executionID == "" {
		return map[string]any{
			"status": "skipped",
			"reason": "execution_id is required",
		}
	}

	if actionMode == "" {
		actionMode = execution.ActionModeDryRun
	}
	if actionMode == execution.ActionModeApply && writebackEnabled {
		roles := azureBlobApplyRoles("apply_approved_writebacks")
		if ok, reason := s.checkEnterpriseApplyAction(r, "azureblob.writeback.apply_approved", roles...); !ok {
			return map[string]any{
				"status":                    "blocked",
				"reason":                    reason,
				"execution_id":              executionID,
				"authorization_enforcement": "apply_enforced",
			}
		}
	}

	intent, err := s.evaluateDACIntent(r, dacIntentInput{
		Operation:  "apply_approved_writebacks",
		SourceID:   s.azureBlobConfig().DataStoreID,
		ActionMode: actionMode,
		Metadata: map[string]string{
			"entrypoint":        "azure_blob_batch_writeback",
			"execution_id":      executionID,
			"writeback_enabled": strconv.FormatBool(writebackEnabled),
		},
	})
	if err != nil {
		return map[string]any{
			"status": "failed",
			"reason": err.Error(),
		}
	}
	if intent.Status != "ok" || intent.GuardianDecision == nil || !intent.GuardianDecision.Approved {
		return map[string]any{
			"status":     "blocked",
			"reason":     "Batch write-back blocked by Data Agent Cell Orchestrator.",
			"dac_intent": intent,
		}
	}

	if actionMode == execution.ActionModeApply && writebackEnabled {
		if ok, reason := s.hasApprovedWritebackApproval(r, executionID); !ok {
			return map[string]any{
				"status":               "blocked",
				"reason":               reason,
				"execution_id":         executionID,
				"guardian_enforcement": "required_before_writeback",
				"hitl_enforcement":     "approved_control_plane_approval_required",
			}
		}
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	items, total, err := s.store.ListActionPlan(r.Context(), executionID, limit, 0)
	if err != nil {
		return map[string]any{
			"status": "failed",
			"reason": err.Error(),
		}
	}

	baseCfg := s.azureBlobConfig()

	resp := ApplyApprovedWritebacksResponse{
		Status:      "completed",
		ExecutionID: executionID,
		Checked:     total,
		Results:     []any{},
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, item := range items {
		actionType := strings.TrimSpace(item.ActionRequested)
		guardianDecision := strings.TrimSpace(item.GuardianDecision)
		actionID := "action-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

		if !execution.CanApplyWriteback(actionMode, writebackEnabled, guardianDecision, actionType) {
			resp.Skipped++

			result := map[string]any{
				"status":            "skipped",
				"file_id":           item.FileID,
				"path":              item.Path,
				"action_type":       actionType,
				"guardian_decision": guardianDecision,
				"reason":            "Not eligible for real write-back.",
			}
			result["action_receipt"] = buildBatchWritebackReceipt(executionID, item, actionID, firstNonBlank(item.ProvenanceID, dacDecisionID(intent)), "skipped", actionMode, false, result["reason"].(string))

			resp.Results = append(resp.Results, result)

			_ = s.store.InsertWritebackAction(
				r.Context(),
				executionID,
				actionID,
				item.ProvenanceID,
				item.FileID,
				actionType,
				"skipped",
				"not eligible for write-back",
			)

			continue
		}

		if actionMode == execution.ActionModeApply && strings.TrimSpace(item.ProvenanceID) == "" {
			resp.Skipped++
			reason := "Valid Guardian decision ID is required before applying connector write-back."
			result := map[string]any{
				"status":            "skipped",
				"file_id":           item.FileID,
				"path":              item.Path,
				"action_type":       actionType,
				"guardian_decision": guardianDecision,
				"reason":            reason,
				"action_receipt":    buildBatchWritebackReceipt(executionID, item, actionID, dacDecisionID(intent), "skipped", actionMode, false, reason),
			}
			resp.Results = append(resp.Results, result)
			_ = s.store.InsertWritebackAction(
				r.Context(),
				executionID,
				actionID,
				dacDecisionID(intent),
				item.FileID,
				actionType,
				"skipped",
				reason,
			)
			continue
		}

		writeReq := azureblob.WritebackRequest{
			FileID: item.FileID,
			Metadata: map[string]string{
				"everest_classification": item.Classification,
				"everest_policy_action":  actionType,
				"everest_guardian":       guardianDecision,
				"everest_execution_id":   executionID,
			},
			IndexTags: map[string]string{
				"everest_classification": item.Classification,
				"everest_risk":           item.RiskLevel,
				"everest_guardian":       guardianDecision,
			},
		}

		var connectorResult any
		actionCfg := baseCfg
		if container, err := azureblob.ContainerFromFileID(item.FileID); err == nil && container != "" {
			actionCfg.ContainerName = container
		}

		scanner, err := s.newAzureBlobScanner(actionCfg)
		if err != nil {
			resp.Failed++

			_ = s.store.UpdateManifestActionStatus(r.Context(), executionID, item.FileID, "failed", err.Error())
			_ = s.store.InsertWritebackAction(
				r.Context(),
				executionID,
				actionID,
				item.ProvenanceID,
				item.FileID,
				actionType,
				"failed",
				err.Error(),
			)

			resp.Results = append(resp.Results, map[string]any{
				"status":         "failed",
				"file_id":        item.FileID,
				"path":           item.Path,
				"action_type":    actionType,
				"error":          err.Error(),
				"action_receipt": buildBatchWritebackReceipt(executionID, item, actionID, item.ProvenanceID, "failed", actionMode, false, err.Error()),
			})

			continue
		}

		switch actionType {
		case execution.ActionApplyBlobMetadata:
			connectorResult, err = scanner.ApplyMetadata(r.Context(), writeReq)

		case execution.ActionApplyBlobIndexTags:
			connectorResult, err = scanner.ApplyIndexTags(r.Context(), writeReq)

		default:
			err = nil
			resp.Skipped++

			result := map[string]any{
				"status":      "skipped",
				"file_id":     item.FileID,
				"path":        item.Path,
				"action_type": actionType,
				"reason":      "Unsupported action type for Azure Blob write-back.",
			}
			result["action_receipt"] = buildBatchWritebackReceipt(executionID, item, actionID, item.ProvenanceID, "skipped", actionMode, false, "Unsupported action type for Azure Blob write-back.")

			resp.Results = append(resp.Results, result)

			_ = s.store.InsertWritebackAction(
				r.Context(),
				executionID,
				actionID,
				item.ProvenanceID,
				item.FileID,
				actionType,
				"skipped",
				"unsupported action type",
			)

			continue
		}

		if err != nil {
			resp.Failed++

			_ = s.store.UpdateManifestActionStatus(r.Context(), executionID, item.FileID, "failed", err.Error())
			_ = s.store.InsertWritebackAction(
				r.Context(),
				executionID,
				actionID,
				item.ProvenanceID,
				item.FileID,
				actionType,
				"failed",
				err.Error(),
			)

			resp.Results = append(resp.Results, map[string]any{
				"status":         "failed",
				"file_id":        item.FileID,
				"path":           item.Path,
				"action_type":    actionType,
				"error":          err.Error(),
				"action_receipt": buildBatchWritebackReceipt(executionID, item, actionID, item.ProvenanceID, "failed", actionMode, false, err.Error()),
			})

			continue
		}

		resp.Applied++

		_ = s.store.UpdateManifestActionStatus(r.Context(), executionID, item.FileID, "completed", "")
		_ = s.store.InsertWritebackAction(
			r.Context(),
			executionID,
			actionID,
			item.ProvenanceID,
			item.FileID,
			actionType,
			"completed",
			"",
		)

		resp.Results = append(resp.Results, map[string]any{
			"status":             "completed",
			"file_id":            item.FileID,
			"path":               item.Path,
			"action_type":        actionType,
			"connector_response": connectorResult,
			"action_receipt":     buildBatchWritebackReceipt(executionID, item, actionID, item.ProvenanceID, "completed", actionMode, true, "Azure Blob write-back completed."),
		})
	}

	if resp.Failed > 0 {
		resp.Status = "completed_with_failures"
	}

	uploadStatus := uploadWritebackResultsToControlPlane(r, s, executionID)

	return map[string]any{
		"writeback":            resp,
		"control_plane_upload": uploadStatus,
		"system_of_record":     "control_plane",
		"local_store_role":     "execution_working_store",
		"guardian_enforcement": "required_before_writeback",
		"dac_intent":           intent,
	}
}

func buildBatchWritebackReceipt(
	executionID string,
	item duckdbstore.ActionPlanView,
	actionID string,
	decisionID string,
	status string,
	actionMode string,
	applied bool,
	resultSummary string,
) contracts.ActionReceipt {
	now := time.Now().UTC()
	decisionID = firstNonBlank(decisionID, "gd-unavailable")
	receiptID := "receipt-" + strings.TrimPrefix(contracts.StableHash(executionID, actionID, item.FileID, status), "sha256:")
	if len(receiptID) > len("receipt-")+20 {
		receiptID = receiptID[:len("receipt-")+20]
	}
	containerName := ""
	if parsed, err := azureblob.ContainerFromFileID(item.FileID); err == nil {
		containerName = parsed
	}
	return contracts.ActionReceipt{
		ReceiptID:   receiptID,
		ActionID:    actionID,
		RequestID:   firstNonBlank(executionID, "req-unavailable"),
		ExecutionID: executionID,
		DecisionID:  decisionID,
		ConnectorID: "azure_blob",
		ObjectRefs: []contracts.ObjectRef{
			{
				URI:         firstNonBlank(item.FileID, item.Path),
				ConnectorID: "azure_blob",
				SourceID:    "azure_blob",
				Container:   containerName,
				Path:        item.Path,
			},
		},
		ActionType:         item.ActionRequested,
		Status:             status,
		Applied:            applied,
		DryRun:             actionMode != execution.ActionModeApply,
		ResultSummary:      resultSummary,
		ObligationsApplied: []string{contracts.ObligationRecordAudit, contracts.ObligationAttachPolicyHash, contracts.ObligationWriteProvenance},
		EvidenceRefs:       []string{"evidence:batch_writeback:" + actionID},
		ProvenanceRefs:     []string{"provenance:" + decisionID},
		StartedAt:          now,
		CompletedAt:        now,
		ReceiptHash:        contracts.StableHash("batch_writeback_receipt", receiptID, actionID, decisionID, item.FileID, status),
	}
}

func (s *Server) hasApprovedWritebackApproval(r *http.Request, executionID string) (bool, string) {
	return s.hasApprovedAzureBlobActionApproval(
		r,
		executionID,
		execution.ActionApplyBlobMetadata,
		execution.ActionApplyBlobIndexTags,
	)
}
