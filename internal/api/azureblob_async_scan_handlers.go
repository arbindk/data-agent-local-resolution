package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type azureBlobAsyncScanRequest struct {
	SourceID    string                            `json:"source_id,omitempty"`
	DataStoreID string                            `json:"data_store_id,omitempty"`
	BatchID     string                            `json:"batch_id,omitempty"`
	RequestedBy string                            `json:"requested_by,omitempty"`
	Mode        string                            `json:"mode,omitempty"`
	Container   string                            `json:"container,omitempty"`
	Prefix      string                            `json:"prefix,omitempty"`
	Scopes      []controlplane.AzureBlobScanScope `json:"scopes,omitempty"`
	Limit       int                               `json:"limit,omitempty"`
	DryRun      bool                              `json:"dry_run"`
}

func (s *Server) azureBlobAsyncScanJobs(w http.ResponseWriter, r *http.Request) {
	store := s.azureBlobScanJobStore()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"items":  store.List(),
		})
		return

	case http.MethodPost:
		if !s.authorizeEnterpriseAction(w, r, "azureblob.async_scan.queue", "platform_admin", "security_officer", "data_steward", "data_analyst") {
			return
		}
		var req azureBlobAsyncScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		cfg := s.azureBlobConfig()
		sourceID := firstNonBlank(req.SourceID, req.DataStoreID, cfg.DataStoreID, cfg.StorageAccount, "azureblob-local")
		if !s.azureBlobCredentialSet(sourceID) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Azure Blob account access is not configured for the selected source"})
			return
		}

		scopes := req.Scopes
		if len(scopes) == 0 {
			containerName := firstNonBlank(req.Container, cfg.ContainerName)
			prefix := firstNonBlank(req.Prefix, cfg.Prefix)
			if containerName != "" {
				scopes = []controlplane.AzureBlobScanScope{{Container: containerName, Prefixes: []string{prefix}}}
			}
		}

		actor := s.enterpriseActor(r)
		mode := firstNonBlank(req.Mode, "metadata_scan")
		intent, err := s.evaluateDACIntent(r, dacIntentInput{
			Operation:  mode,
			SourceID:   sourceID,
			ActionMode: execution.ActionModeDryRun,
			Container:  firstScopeContainer(scopes),
			Prefix:     firstScopePrefix(scopes),
			Limit:      req.Limit,
			ObjectRefs: objectRefsFromAzureBlobScanScopes(sourceID, scopes),
			Metadata: map[string]string{
				"entrypoint":   "azure_blob_async_scan",
				"scan_mode":    mode,
				"requested_by": firstNonBlank(req.RequestedBy, actor.Actor),
				"scope_count":  strconv.Itoa(len(scopes)),
			},
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if intent.Status != "ok" || intent.GuardianDecision == nil || !intent.GuardianDecision.Approved {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":     "blocked",
				"reason":     "Async scan request blocked by Data Agent Cell Orchestrator.",
				"dac_intent": intent,
			})
			return
		}

		job, err := store.Start(controlplane.AzureBlobScanJobRequest{
			SourceID:         sourceID,
			DataStoreID:      firstNonBlank(req.DataStoreID, cfg.DataStoreID, sourceID),
			BatchID:          firstNonBlank(req.BatchID, cfg.BatchID),
			RequestedBy:      firstNonBlank(req.RequestedBy, actor.Actor, "portal-user"),
			Mode:             mode,
			Scopes:           scopes,
			Limit:            req.Limit,
			DryRun:           req.DryRun,
			RequestEnvelope:  &intent.RequestEnvelope,
			GuardianDecision: intent.GuardianDecision,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		s.recordAsyncScanLifecycle(job, "azureblob.async_scan.queued", "Async Azure Blob scan job was queued.")
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":     "queued",
			"job":        job,
			"dac_intent": intent,
		})
		return

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}

func (s *Server) azureBlobAsyncScanJobByID(w http.ResponseWriter, r *http.Request) {
	store := s.azureBlobScanJobStore()
	path := strings.TrimPrefix(r.URL.Path, "/v1/scan/azureblob/jobs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "job_id is required"})
		return
	}
	jobID := strings.TrimSpace(parts[0])
	if strings.EqualFold(jobID, "reclaim-stale") {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if !s.authorizeEnterpriseAction(w, r, "azureblob.async_scan.reclaim_stale", "platform_admin", "security_officer") {
			return
		}
		jobs := store.ReclaimStaleLeases(time.Now().UTC())
		for _, job := range jobs {
			s.recordAsyncScanLifecycle(job, "azureblob.async_scan.stale_lease_reclaimed", "Stale async scan worker lease was reclaimed.")
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(jobs),
			"items":  jobs,
		})
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		job, ok := store.Get(jobID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan job not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "job": job})
		return
	}

	if len(parts) == 2 && strings.EqualFold(parts[1], "normalized") {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		job, ok := store.Get(jobID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan job not found"})
			return
		}
		if strings.TrimSpace(job.NormalizedPath) == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "normalized async scan artifact is not available yet"})
			return
		}
		data, err := os.ReadFile(job.NormalizedPath)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "read normalized async scan artifact: " + err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(parts[1]))
	if !s.authorizeAsyncScanAction(w, r, action) {
		return
	}
	var (
		job        controlplane.AzureBlobScanJob
		err        error
		wasRunning bool
	)
	switch action {
	case "resume":
		if existing, ok := store.Get(jobID); ok && existing.Status == "running" {
			wasRunning = true
		}
		job, err = store.Resume(jobID)
	case "pause":
		job, err = store.Pause(jobID)
	case "cancel":
		job, err = store.Cancel(jobID)
	case "complete":
		job, err = store.Complete(jobID)
	case "fail":
		var body struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		job, err = store.Fail(jobID, firstNonBlank(body.Reason, "Scan worker reported failure."))
	case "checkpoint":
		var checkpoint controlplane.AzureBlobScanCheckpoint
		if err := json.NewDecoder(r.Body).Decode(&checkpoint); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid checkpoint body: " + err.Error()})
			return
		}
		job, err = store.UpdateCheckpoint(jobID, checkpoint)
	default:
		err = fmt.Errorf("unsupported scan job action: %s", action)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.recordAsyncScanLifecycle(job, "azureblob.async_scan."+action, "Async Azure Blob scan job "+action+" recorded.")
	if action == "resume" && !wasRunning {
		s.startAzureBlobAsyncWorker(job)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": job.Status,
		"job":    job,
	})
}

func (s *Server) azureBlobScanJobStore() *controlplane.AzureBlobScanJobStore {
	if s.scanJobs == nil {
		s.scanJobs = controlplane.NewAzureBlobScanJobStore()
	}
	return s.scanJobs
}

func (s *Server) recordAsyncScanLifecycle(job controlplane.AzureBlobScanJob, eventType string, summary string) {
	if s.portalState != nil {
		s.portalState.UpsertJob(portalstate.Job{
			JobID:       job.JobID,
			DataStoreID: job.DataStoreID,
			Status:      job.Status,
		})
	}

	details := map[string]any{
		"source_id":             job.SourceID,
		"batch_id":              job.BatchID,
		"mode":                  job.Mode,
		"dry_run":               job.DryRun,
		"limit":                 job.Limit,
		"scopes":                job.Scopes,
		"checkpoint":            job.Checkpoint,
		"attempts":              job.Attempts,
		"records_written":       job.RecordsWritten,
		"normalized_path":       job.NormalizedPath,
		"lease_owner":           job.LeaseOwner,
		"lease_expires_at":      job.LeaseExpiresAt,
		"heartbeat_at":          job.HeartbeatAt,
		"retry_count":           job.RetryCount,
		"throttle_count":        job.ThrottleCount,
		"retry_after":           job.RetryAfter,
		"last_error":            job.LastError,
		"cancel_state":          job.CancelRequested,
		"request_id":            job.RequestID,
		"correlation_id":        job.CorrelationID,
		"policy_bundle_version": job.PolicyBundleVersion,
		"policy_bundle_hash":    job.PolicyBundleHash,
		"guardian_decision_id":  job.GuardianDecisionID,
		"request_envelope":      job.RequestEnvelope,
	}
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "azure_blob_async_scan",
		SubjectID:    job.JobID,
		EventType:    eventType,
		Status:       job.Status,
		Severity:     "info",
		SourceSystem: "azure_blob",
		DataStoreID:  job.DataStoreID,
		JobID:        job.JobID,
		Container:    job.Checkpoint.Container,
		Operation:    job.Mode,
		Summary:      summary,
		Details:      details,
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:   eventType,
		Actor:       firstNonBlank(job.RequestedBy, "portal-user"),
		Status:      job.Status,
		TargetType:  "azure_blob_async_scan",
		TargetID:    job.JobID,
		SourceID:    job.SourceID,
		DataStoreID: job.DataStoreID,
		Operation:   job.Mode,
		DryRun:      job.DryRun,
		Summary:     summary,
		Details:     details,
	})
}

func objectRefsFromAzureBlobScanScopes(sourceID string, scopes []controlplane.AzureBlobScanScope) []contracts.ObjectRef {
	out := make([]contracts.ObjectRef, 0, len(scopes))
	for _, scope := range scopes {
		containerName := strings.TrimSpace(scope.Container)
		if containerName == "" {
			continue
		}
		prefixes := scope.Prefixes
		if len(prefixes) == 0 {
			prefixes = []string{""}
		}
		for _, prefix := range prefixes {
			prefix = strings.Trim(strings.TrimSpace(prefix), "/")
			out = append(out, contracts.ObjectRef{
				URI:         "azureblob:" + containerName + ":" + prefix,
				ConnectorID: "azure_blob",
				SourceID:    sourceID,
				Container:   containerName,
				Path:        prefix,
			})
		}
	}
	return out
}

func firstScopeContainer(scopes []controlplane.AzureBlobScanScope) string {
	if len(scopes) == 0 {
		return ""
	}
	return strings.TrimSpace(scopes[0].Container)
}

func firstScopePrefix(scopes []controlplane.AzureBlobScanScope) string {
	if len(scopes) == 0 || len(scopes[0].Prefixes) == 0 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(scopes[0].Prefixes[0]), "/")
}
