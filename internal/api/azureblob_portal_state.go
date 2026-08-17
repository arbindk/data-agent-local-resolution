package api

import (
	"strconv"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) persistAzureWorkbenchResponse(req azureBlobWorkbenchRequest, resp *azureBlobWorkbenchResponse) {
	if s.portalState == nil || resp == nil {
		return
	}

	cfg := s.azureBlobConfig()
	if strings.TrimSpace(req.Container) != "" {
		cfg.ContainerName = strings.TrimSpace(req.Container)
	}
	if strings.TrimSpace(resp.Container) != "" {
		cfg.ContainerName = strings.TrimSpace(resp.Container)
	}
	if strings.TrimSpace(req.Prefix) != "" {
		cfg.Prefix = strings.Trim(strings.TrimSpace(req.Prefix), "/")
	}
	if strings.TrimSpace(req.ConnectionString) != "" && strings.TrimSpace(cfg.StorageAccount) == "" {
		cfg.StorageAccount = storageAccountFromConnectionString(req.ConnectionString)
	}
	if strings.TrimSpace(req.SourceID) != "" {
		sourceID := strings.TrimSpace(req.SourceID)
		if sourceID != strings.TrimSpace(cfg.DataStoreID) && sourceID != strings.TrimSpace(cfg.StorageAccount) {
			cfg.StorageAccount = ""
		}
		cfg.DataStoreID = sourceID
	}

	source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg) || strings.TrimSpace(req.ConnectionString) != "")
	s.portalState.UpsertAzureSource(source)

	if len(resp.Containers) > 0 {
		containers := make([]portalstate.AzureContainer, 0, len(resp.Containers))
		for _, item := range resp.Containers {
			containers = append(containers, portalstate.AzureContainer{
				Name:         item.Name,
				LastModified: item.LastModified,
				ETag:         item.ETag,
				Metadata:     item.Metadata,
			})
		}
		s.portalState.UpsertAzureContainers(source, containers)
	}

	if len(resp.Items) > 0 {
		blobs := make([]portalstate.AzureBlob, 0, len(resp.Items))
		for _, item := range resp.Items {
			blobs = append(blobs, portalstate.AzureBlob{
				Container:    resp.Container,
				Name:         item.Name,
				FileID:       "azureblob:" + resp.Container + ":" + item.Name,
				SizeBytes:    item.SizeBytes,
				ContentType:  item.ContentType,
				LastModified: item.LastModified,
				AccessTier:   item.AccessTier,
			})
		}
		s.portalState.UpsertAzureBlobs(source, blobs)
	}

	if resp.Details != nil {
		containerName := firstNonBlank(resp.Container, req.Container)
		blobName := firstNonBlank(resp.Details.Name, resp.Blob, req.Blob)
		if containerName != "" && blobName != "" {
			s.portalState.UpsertAzureBlobs(source, []portalstate.AzureBlob{{
				Container:    containerName,
				Name:         blobName,
				FileID:       "azureblob:" + containerName + ":" + blobName,
				SizeBytes:    resp.Details.SizeBytes,
				ContentType:  resp.Details.ContentType,
				LastModified: resp.Details.LastModified,
				AccessTier:   resp.Details.AccessTier,
			}})
		}
	}

	s.portalState.RecordWorkbench(portalstate.WorkbenchHistory{
		Operation:   resp.Operation,
		Status:      resp.Status,
		SourceID:    source.SourceID,
		DataStoreID: source.DataStoreID,
		Container:   firstNonBlank(resp.Container, req.Container),
		Blob:        firstNonBlank(resp.Blob, req.Blob),
		Count:       resp.Count,
		DryRun:      resp.DryRun,
		Message:     resp.Message,
	})
	containerName := firstNonBlank(resp.Container, req.Container)
	blobName := firstNonBlank(resp.Blob, req.Blob)
	details := map[string]any{
		"count":     resp.Count,
		"dry_run":   resp.DryRun,
		"auth_mode": resp.AuthMode,
	}
	if strings.TrimSpace(resp.TargetContainer) != "" {
		details["target_container"] = resp.TargetContainer
	}
	if strings.TrimSpace(resp.TargetBlob) != "" {
		details["target_blob"] = resp.TargetBlob
	}
	if strings.TrimSpace(resp.TargetTier) != "" {
		details["target_tier"] = resp.TargetTier
	}
	if resp.Applied != nil {
		details["applied"] = resp.Applied
	}
	if resp.DACIntent != nil {
		details["dac_intent"] = resp.DACIntent
		if resp.DACIntent.GuardianDecision != nil {
			details["guardian_decision_id"] = resp.DACIntent.GuardianDecision.DecisionID
			details["policy_bundle_hash"] = resp.DACIntent.GuardianDecision.PolicyBundleHash
		}
		if resp.DACIntent.ActionReceipt != nil {
			details["action_receipt"] = resp.DACIntent.ActionReceipt
		}
	}
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "azure_blob_workbench",
		SubjectID:    firstNonBlank(blobName, containerName, resp.Operation),
		EventType:    "azureblob.workbench." + resp.Operation,
		Status:       resp.Status,
		SourceSystem: "azure_blob",
		DataStoreID:  source.DataStoreID,
		Container:    containerName,
		Blob:         blobName,
		FileID:       fileIDForWorkbench(containerName, blobName),
		Operation:    resp.Operation,
		Summary:      workbenchEvidenceSummary(resp),
		Details:      details,
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:   "azureblob.workbench." + resp.Operation,
		Actor:       "portal-user",
		Status:      resp.Status,
		TargetType:  "azure_blob",
		TargetID:    firstNonBlank(fileIDForWorkbench(containerName, blobName), containerName),
		SourceID:    source.SourceID,
		DataStoreID: source.DataStoreID,
		Operation:   resp.Operation,
		DryRun:      resp.DryRun,
		Summary:     workbenchEvidenceSummary(resp),
		Details:     details,
	})
	if workbenchOperationCreatesAction(resp.Operation) {
		s.upsertActionRecord(portalstate.ActionRecord{
			ExecutionID:         "",
			DecisionID:          dacDecisionID(resp.DACIntent),
			FileID:              fileIDForWorkbench(containerName, blobName),
			BlobPath:            blobName,
			Container:           containerName,
			Operation:           resp.Operation,
			Status:              actionStatusFromAPIStatus(resp.Status),
			RequestedBy:         "portal-user",
			ActionMode:          workbenchActionMode(req, resp),
			WritebackEnabled:    !resp.DryRun && resp.Status == "ok",
			GuardianDecision:    dacDecisionString(resp.DACIntent),
			ApprovalEnforcement: "role_or_confirmation_checked_before_workbench_apply",
			GuardianEnforcement: "orchestrator_intent_evaluated_before_connector_dispatch",
			Reason:              resp.Message,
			Details:             details,
		})
	}
}

func fileIDForWorkbench(containerName string, blobName string) string {
	if strings.TrimSpace(containerName) == "" || strings.TrimSpace(blobName) == "" {
		return ""
	}
	return "azureblob:" + strings.TrimSpace(containerName) + ":" + strings.Trim(strings.TrimSpace(blobName), "/")
}

func workbenchOperationCreatesAction(operation string) bool {
	switch normalizeWorkbenchOperation(operation) {
	case "create_blob", "update_blob", "set_metadata", "set_tags", "copy_blob", "move_blob", "set_tier", "rehydrate_blob", "create_container", "delete_container", "delete_blob":
		return true
	default:
		return false
	}
}

func workbenchActionMode(req azureBlobWorkbenchRequest, resp *azureBlobWorkbenchResponse) string {
	if req.DryRun || resp != nil && resp.DryRun {
		return "dry_run"
	}
	return "apply"
}

func workbenchEvidenceSummary(resp *azureBlobWorkbenchResponse) string {
	if resp == nil {
		return "Azure Blob workbench operation recorded."
	}
	if strings.TrimSpace(resp.Message) != "" {
		return resp.Message
	}
	if resp.Count > 0 {
		return resp.Operation + " returned " + strconv.Itoa(resp.Count) + " records."
	}
	return "Azure Blob workbench operation " + resp.Operation + " completed with status " + resp.Status + "."
}
