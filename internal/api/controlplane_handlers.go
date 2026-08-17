package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) runNextControlPlaneLease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	platformURL := os.Getenv("EVEREST_PLATFORM_URL")
	if platformURL == "" {
		platformURL = "http://localhost:9090"
	}

	agentID := os.Getenv("EVEREST_AGENT_ID")
	if agentID == "" {
		agentID = "agentcell-win-001"
	}

	client := controlplane.NewClient(platformURL)

	next, err := client.NextLease(r.Context(), agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if next.Status == "empty" {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "empty",
			"message":  next.Message,
			"agent_id": agentID,
		})
		return
	}

	lease := next.Lease

	_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
		AgentID: agentID,
		Status:  "RUNNING",
		Message: "Data Agent accepted lease and started Azure Blob local execution.",
	})

	scopes := normalizeAzureScopes(lease.Scope)
	if len(scopes) == 0 {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Lease has no Azure Blob container scope.",
		})

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lease has no Azure Blob container scope"})
		return
	}

	batchID := "batch-" + lease.ExecutionID
	if batchID == "batch-" {
		batchID = "batch-" + time.Now().UTC().Format("20060102-150405")
	}

	cfg := s.azureBlobConfig()

	// Product rule:
	// Control Plane lease scope is the source of truth for execution scope.
	// Data Agent local config only supplies secure connection material.
	cfg.DataStoreID = strings.TrimSpace(lease.DataStoreID)
	cfg.BatchID = batchID
	cfg.ActionMode = strings.TrimSpace(lease.ActionMode)
	cfg.WritebackEnabled = boolString(lease.WritebackEnabled)

	if strings.TrimSpace(cfg.ActionMode) == "" {
		cfg.ActionMode = "dry_run"
	}

	if strings.TrimSpace(cfg.ConnectionString) == "" {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Azure Blob connection string is not configured on Data Agent.",
		})

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Azure Blob connection string is not configured on Data Agent"})
		return
	}

	batch, scanResults, err := s.scanAzureBlobLeaseScopes(r, cfg, lease, scopes)
	if err != nil {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Azure Blob scan failed: " + err.Error(),
		})

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := s.store.ReplaceNormalizedFiles(r.Context(), batch); err != nil {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Persist normalized batch failed: " + err.Error(),
		})

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	count, err := s.store.CountNormalizedFiles(r.Context(), batch.BatchID)
	if err != nil {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Count normalized records failed: " + err.Error(),
		})

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	req, err := loadDefaultExecutionRequest()
	if err != nil {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Load default execution request failed: " + err.Error(),
		})

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	req.ExecutionID = lease.ExecutionID
	req.BatchID = batch.BatchID
	req.BatchInputPath = ""

	req.Lease.LeaseID = lease.LeaseID
	req.Lease.JobID = lease.JobID
	req.Lease.DataStoreID = lease.DataStoreID
	req.Lease.CompiledPolicyVersion = lease.CompiledPolicyVersion
	req.Lease.WorkflowVersion = lease.WorkflowVersion

	req.ExecutionState.DataStoreID = lease.DataStoreID
	req.ExecutionState.ProcessingTier = lease.ProcessingTier
	req.ExecutionState.ProcessingMode = lease.ProcessingMode
	req.ExecutionState.WorkflowVersion = lease.WorkflowVersion
	req.ExecutionState.OperationalConfig.ActionMode = lease.ActionMode
	req.ExecutionState.OperationalConfig.LocalExecutionRequired = true
	req.ExecutionState.OperationalConfig.GuardianBeforeActionRequired = true
	req.ExecutionState.Settings.EnableContentIntelligence = true
	req.ExecutionState.Settings.EnableGuardianEvaluation = true
	req.ExecutionState.Settings.EnableWriteBackActions = lease.WritebackEnabled

	result, err := s.orchestrator.Start(r.Context(), *req)
	if err != nil {
		_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
			AgentID: agentID,
			Status:  "FAILED",
			Message: "Agent Core execution failed: " + err.Error(),
		})

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	finalStatus := "COMPLETED"
	if result.Status.Status != "COMPLETED" {
		finalStatus = "FAILED"
	}
	if s.portalState != nil {
		s.portalState.UpsertJob(portalstate.Job{
			JobID:       lease.JobID,
			DataStoreID: lease.DataStoreID,
			Status:      finalStatus,
		})
		source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg))
		containers := make([]portalstate.AzureContainer, 0, len(scopes))
		for _, scope := range scopes {
			containers = append(containers, portalstate.AzureContainer{Name: scope.Container})
		}
		s.portalState.UpsertAzureContainers(source, containers)
		blobs := make([]portalstate.AzureBlob, 0, len(batch.Records))
		for _, record := range batch.Records {
			containerName := cfg.ContainerName
			if parsed, err := azureblob.ContainerFromFileID(record.FileID); err == nil && parsed != "" {
				containerName = parsed
			}
			blobs = append(blobs, portalstate.AzureBlob{
				Container:    containerName,
				Name:         record.Path,
				FileID:       record.FileID,
				SizeBytes:    record.SizeBytes,
				ContentType:  record.ContentType,
				LastModified: record.LastModified,
			})
		}
		s.portalState.UpsertAzureBlobs(source, blobs)
	}

	_, _ = client.HeartbeatLease(r.Context(), lease.LeaseID, controlplane.LeaseHeartbeatRequest{
		AgentID:          agentID,
		Status:           finalStatus,
		RecordsProcessed: result.Status.RecordsProcessed,
		Message:          "Data Agent completed lease execution.",
	})

	autoWriteback := map[string]any{
		"status": "skipped",
		"reason": "Automatic write-back runs only when action_mode=apply and writeback_enabled=true.",
	}

	if finalStatus == "COMPLETED" &&
		lease.ActionMode == execution.ActionModeApply &&
		lease.WritebackEnabled {
		autoWriteback = s.applyApprovedAzureBlobWritebacksInternal(
			r,
			lease.ExecutionID,
			lease.ActionMode,
			lease.WritebackEnabled,
			100,
		)
	}

	evidenceUpload := uploadExecutionEvidenceToControlPlane(r, s, client, lease.ExecutionID, result)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       strings.ToLower(finalStatus),
		"agent_id":     agentID,
		"platform_url": platformURL,
		"lease":        lease,
		"scan_result":  scanResults,
		"effective_scan_config": map[string]any{
			"container":          scopes[0].Container,
			"prefix":             firstPrefix(scopes[0]),
			"scopes":             scopes,
			"data_store_id":      cfg.DataStoreID,
			"batch_id":           cfg.BatchID,
			"action_mode":        cfg.ActionMode,
			"writeback_enabled":  cfg.WritebackEnabled,
			"credential_present": cfg.ConnectionString != "",
			"source_of_truth":    "control_plane_lease_scope",
		},
		"normalized_written": count,
		"execution_result":   result,
		"enterprise_flow":    "control_plane_lease_to_go_data_agent_execution",
		"evidence_upload":    evidenceUpload,
		"auto_writeback":     autoWriteback,
		"local_store":        "ephemeral_working_store",
		"recovery_source":    "control_plane_lease_checkpoint_source_marker",
		"writeback_readiness": map[string]any{
			"action_mode":              lease.ActionMode,
			"writeback_enabled":        lease.WritebackEnabled,
			"real_writeback_supported": lease.ActionMode == execution.ActionModeApply && lease.WritebackEnabled,
			"allowed_real_actions": []string{
				execution.ActionApplyBlobMetadata,
				execution.ActionApplyBlobIndexTags,
			},
			"dry_run_only_actions": []string{
				execution.ActionDryRunDelete,
				execution.ActionDryRunArchive,
				execution.ActionDryRunRetrieve,
				execution.ActionDryRunQuarantine,
				execution.ActionDryRunMigration,
			},
			"guardian_required": true,
			"note":              "Real write-back is allowed only for Azure Blob metadata/index tags after Guardian allow decision.",
		},
	})
}

func (s *Server) scanAzureBlobLeaseScopes(
	r *http.Request,
	baseCfg azureblob.Config,
	lease controlplane.Lease,
	scopes []controlplane.AzureBlobContainerScope,
) (contracts.NormalizedBatch, []any, error) {
	combined := contracts.NormalizedBatch{
		BatchID:     baseCfg.BatchID,
		LeaseID:     lease.LeaseID,
		DataStoreID: lease.DataStoreID,
		Records:     []contracts.NormalizedFileRecord{},
	}
	results := []any{}

	for _, scope := range scopes {
		prefixes := scope.Prefixes
		if len(prefixes) == 0 {
			prefixes = []string{""}
		}

		for _, prefix := range prefixes {
			cfg := baseCfg
			cfg.ContainerName = strings.TrimSpace(scope.Container)
			cfg.Prefix = strings.TrimSpace(prefix)

			scanner, err := s.newAzureBlobScanner(cfg)
			if err != nil {
				return combined, results, err
			}

			batch, scanResult, err := scanner.Scan(r.Context())
			if err != nil {
				return combined, results, err
			}

			if batch != nil {
				combined.Records = append(combined.Records, batch.Records...)
			}
			if scanResult != nil {
				results = append(results, scanResult)
			}
		}
	}

	return combined, results, nil
}

func normalizeAzureScopes(scopes []controlplane.AzureBlobContainerScope) []controlplane.AzureBlobContainerScope {
	out := []controlplane.AzureBlobContainerScope{}
	for _, scope := range scopes {
		container := strings.TrimSpace(scope.Container)
		if container == "" {
			continue
		}

		prefixes := []string{}
		for _, prefix := range scope.Prefixes {
			prefixes = append(prefixes, strings.TrimSpace(prefix))
		}
		if len(prefixes) == 0 {
			prefixes = []string{""}
		}

		out = append(out, controlplane.AzureBlobContainerScope{
			Container: container,
			Prefixes:  prefixes,
		})
	}
	return out
}

func firstPrefix(scope controlplane.AzureBlobContainerScope) string {
	if len(scope.Prefixes) == 0 {
		return ""
	}
	return strings.TrimSpace(scope.Prefixes[0])
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

var _ = execution.ExecutionStartRequest{}

func uploadExecutionEvidenceToControlPlane(
	r *http.Request,
	s *Server,
	client *controlplane.Client,
	executionID string,
	result *execution.ExecutionResult,
) map[string]string {
	status := map[string]string{}

	if result == nil {
		status["execution_status"] = "failed: nil execution result"
		status["manifest_summary"] = "skipped: execution failed"
		status["manifest_details"] = "skipped: execution failed"
		status["provenance"] = "skipped: execution failed"
		status["action_results"] = "skipped: execution failed"
		return status
	}

	if result.Status.Status != "COMPLETED" {
		_ = client.UploadExecutionStatus(r.Context(), executionID, result.Status)
		status["execution_status"] = "uploaded"
		status["manifest_summary"] = "skipped: execution did not complete"
		status["manifest_details"] = "skipped: execution did not complete"
		status["provenance"] = "uploaded_if_available"
		status["action_results"] = "skipped: execution did not complete"
		return status
	}

	if err := client.UploadExecutionStatus(r.Context(), executionID, result.Status); err != nil {
		status["execution_status"] = "failed: " + err.Error()
	} else {
		status["execution_status"] = "uploaded"
	}

	summary, err := s.store.GetManifestSummary(r.Context(), executionID)
	if err != nil {
		status["manifest_summary"] = "failed: " + err.Error()
	} else if err := client.UploadManifestSummary(r.Context(), summary); err != nil {
		status["manifest_summary"] = "failed: " + err.Error()
	} else {
		status["manifest_summary"] = "uploaded"
	}

	details, _, err := s.store.ListManifestDetails(r.Context(), executionID, 5000, 0)
	if err != nil {
		status["manifest_details"] = "failed: " + err.Error()
	} else if err := client.UploadManifestDetails(r.Context(), executionID, details); err != nil {
		status["manifest_details"] = "failed: " + err.Error()
	} else {
		status["manifest_details"] = "uploaded"
	}

	if err := client.UploadProvenance(r.Context(), executionID, result.Provenance); err != nil {
		status["provenance"] = "failed: " + err.Error()
	} else {
		status["provenance"] = "uploaded"
	}

	actions, _, err := s.store.ListActionPlan(r.Context(), executionID, 5000, 0)
	if err != nil {
		status["action_results"] = "failed: " + err.Error()
	} else if err := client.UploadActionResults(r.Context(), executionID, actions); err != nil {
		status["action_results"] = "failed: " + err.Error()
	} else {
		status["action_results"] = "uploaded"
	}

	return status
}
