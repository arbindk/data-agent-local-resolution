package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

const asyncAzureBlobPageSize = 500
const asyncAzureBlobLeaseTTL = 2 * time.Minute
const asyncAzureBlobMaxPageRetries = 5

func (s *Server) startAzureBlobAsyncWorker(job controlplane.AzureBlobScanJob) {
	workerID := "worker-" + time.Now().UTC().Format("20060102-150405.000000000")
	go func() {
		if _, err := s.runAzureBlobAsyncWorker(context.Background(), job.JobID, workerID); err != nil {
			failed, failErr := s.azureBlobScanJobStore().Fail(job.JobID, err.Error())
			if failErr == nil {
				s.recordAsyncScanLifecycle(failed, "azureblob.async_scan.failed", "Async Azure Blob scan job failed.")
			}
		}
	}()
}

func (s *Server) runAzureBlobAsyncWorker(ctx context.Context, jobID string, workerID string) (controlplane.AzureBlobScanJob, error) {
	store := s.azureBlobScanJobStore()
	job, ok := store.Get(jobID)
	if !ok {
		return controlplane.AzureBlobScanJob{}, fmt.Errorf("scan job not found")
	}
	if job.Status != "running" {
		var err error
		job, err = store.Resume(jobID)
		if err != nil {
			return job, err
		}
	}
	leasedJob, acquired, err := store.AcquireLease(jobID, workerID, asyncAzureBlobLeaseTTL)
	if err != nil {
		return job, err
	}
	if !acquired {
		return leasedJob, nil
	}
	defer func() {
		_, _ = store.ReleaseLease(jobID, workerID)
	}()
	job = leasedJob

	client, authMode, err := s.azureBlobWorkbenchClient(azureBlobWorkbenchRequest{SourceID: job.SourceID})
	if err != nil {
		return job, err
	}

	s.recordAsyncScanLifecycle(job, "azureblob.async_scan.worker_started", "Async Azure Blob scan worker started.")

	for scopeIndex, scope := range job.Scopes {
		if scopeIndex < job.Checkpoint.ScopeIndex {
			continue
		}
		prefixes := scope.Prefixes
		if len(prefixes) == 0 {
			prefixes = []string{""}
		}
		for prefixIndex, prefix := range prefixes {
			prefix = strings.Trim(strings.TrimSpace(prefix), "/")
			if scopeIndex == job.Checkpoint.ScopeIndex && prefixIndex < job.Checkpoint.PrefixIndex {
				continue
			}

			nextMarker := ""
			if scopeIndex == job.Checkpoint.ScopeIndex && prefixIndex == job.Checkpoint.PrefixIndex {
				nextMarker = strings.TrimSpace(job.Checkpoint.PageMarker)
			}
			for {
				current, ok := store.Get(jobID)
				if !ok {
					return job, fmt.Errorf("scan job not found")
				}
				if current.CancelRequested || current.Status == "cancelled" {
					return current, nil
				}
				if current.Status == "paused" {
					return current, nil
				}
				if _, err := store.HeartbeatLease(jobID, workerID, asyncAzureBlobLeaseTTL); err != nil {
					return current, err
				}

				pageItems, marker, err := s.scanAzureBlobAsyncPageWithRetry(ctx, store, jobID, client, scope.Container, prefix, nextMarker, job)
				if err != nil {
					return current, err
				}
				if len(pageItems) > 0 {
					s.persistAzureBlobAsyncPage(job, scope.Container, pageItems, authMode)
					if written, normalizedPath, err := s.persistAzureBlobAsyncNormalizedPage(job, scope.Container, prefix, pageItems); err == nil && written > 0 {
						job, _ = store.RecordNormalizedWrite(jobID, normalizedPath, written)
						s.recordAsyncScanLifecycle(job, "azureblob.async_scan.normalized_written", "Async Azure Blob scan records were written to the normalized result artifact.")
					}
				}

				current.Checkpoint.ScopeIndex = scopeIndex
				current.Checkpoint.PrefixIndex = prefixIndex
				current.Checkpoint.Container = scope.Container
				current.Checkpoint.Prefix = prefix
				current.Checkpoint.PageMarker = marker
				current.Checkpoint.PagesScanned++
				current.Checkpoint.ObjectsScanned += len(pageItems)
				job, err = store.UpdateCheckpoint(jobID, current.Checkpoint)
				if err != nil {
					return current, err
				}
				s.recordAsyncScanLifecycle(job, "azureblob.async_scan.checkpoint", "Async Azure Blob scan checkpoint updated.")

				if shouldStopAsyncAzureBlobScan(job) {
					completed, err := store.Complete(jobID)
					if err != nil {
						return job, err
					}
					s.recordAsyncScanLifecycle(completed, "azureblob.async_scan.completed", "Async Azure Blob scan job completed.")
					return completed, nil
				}
				if marker == "" {
					if nextCheckpoint, ok := nextAzureBlobAsyncCheckpoint(job, scopeIndex, prefixIndex); ok {
						job, _ = store.UpdateCheckpoint(jobID, nextCheckpoint)
					}
					break
				}
				nextMarker = marker
			}
		}
	}

	completed, err := store.Complete(jobID)
	if err != nil {
		return job, err
	}
	s.recordAsyncScanLifecycle(completed, "azureblob.async_scan.completed", "Async Azure Blob scan job completed.")
	return completed, nil
}

func (s *Server) scanAzureBlobAsyncPage(
	ctx context.Context,
	client *azblob.Client,
	containerName string,
	prefix string,
	marker string,
	job controlplane.AzureBlobScanJob,
) ([]azureBlobWorkbenchBlob, string, error) {
	pageSize := asyncAzureBlobPageSize
	if job.Limit > 0 {
		remaining := job.Limit - job.Checkpoint.ObjectsScanned
		if remaining <= 0 {
			return nil, "", nil
		}
		if remaining < pageSize {
			pageSize = remaining
		}
	}

	pager := client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Prefix:     workbenchOptionalString(prefix),
		Marker:     workbenchOptionalString(marker),
		MaxResults: to.Ptr(int32(pageSize)),
	})
	if !pager.More() {
		return nil, "", nil
	}

	page, err := pager.NextPage(ctx)
	if err != nil {
		var responseErr *azcore.ResponseError
		if strings.TrimSpace(containerName) == "" {
			return nil, "", fmt.Errorf("container is required")
		}
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
			return nil, "", fmt.Errorf("container %s not found", containerName)
		}
		return nil, "", fmt.Errorf("list Azure Blob async page: %w", err)
	}

	items := make([]azureBlobWorkbenchBlob, 0, len(page.Segment.BlobItems))
	for _, item := range page.Segment.BlobItems {
		if item == nil || item.Name == nil {
			continue
		}
		items = append(items, summarizeWorkbenchBlob(item))
	}
	return items, workbenchValueString(page.NextMarker), nil
}

func (s *Server) scanAzureBlobAsyncPageWithRetry(
	ctx context.Context,
	store *controlplane.AzureBlobScanJobStore,
	jobID string,
	client *azblob.Client,
	containerName string,
	prefix string,
	marker string,
	job controlplane.AzureBlobScanJob,
) ([]azureBlobWorkbenchBlob, string, error) {
	var lastErr error
	for attempt := 0; attempt <= asyncAzureBlobMaxPageRetries; attempt++ {
		items, nextMarker, err := s.scanAzureBlobAsyncPage(ctx, client, containerName, prefix, marker, job)
		if err == nil {
			return items, nextMarker, nil
		}
		lastErr = err
		delay, retryable, throttled := azureBlobRetryDelay(err, attempt)
		if !retryable || attempt == asyncAzureBlobMaxPageRetries {
			break
		}
		_, _ = store.RecordRetry(jobID, err.Error(), delay, throttled)
		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, "", lastErr
}

func azureBlobRetryDelay(err error, attempt int) (time.Duration, bool, bool) {
	var responseErr *azcore.ResponseError
	if !errors.As(err, &responseErr) {
		return 0, false, false
	}
	switch responseErr.StatusCode {
	case http.StatusTooManyRequests, http.StatusRequestTimeout, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		delay := time.Duration(1<<attempt) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		return delay, true, responseErr.StatusCode == http.StatusTooManyRequests
	default:
		return 0, false, false
	}
}

func (s *Server) persistAzureBlobAsyncPage(job controlplane.AzureBlobScanJob, containerName string, items []azureBlobWorkbenchBlob, authMode string) {
	if s.portalState == nil || len(items) == 0 {
		return
	}
	cfg := s.azureBlobConfig()
	cfg.DataStoreID = firstNonBlank(job.DataStoreID, job.SourceID, cfg.DataStoreID)
	cfg.ContainerName = containerName
	source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSet(job.SourceID))

	blobs := make([]portalstate.AzureBlob, 0, len(items))
	for _, item := range items {
		blobs = append(blobs, portalstate.AzureBlob{
			Container:    containerName,
			Name:         item.Name,
			FileID:       "azureblob:" + containerName + ":" + item.Name,
			SizeBytes:    item.SizeBytes,
			ContentType:  item.ContentType,
			LastModified: item.LastModified,
			AccessTier:   item.AccessTier,
		})
	}
	s.portalState.UpsertAzureBlobs(source, blobs)
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "azure_blob_async_scan_page",
		SubjectID:    job.JobID + ":" + containerName,
		EventType:    "azureblob.async_scan.page_scanned",
		Status:       job.Status,
		Severity:     "info",
		SourceSystem: "azure_blob",
		DataStoreID:  job.DataStoreID,
		JobID:        job.JobID,
		Container:    containerName,
		Operation:    job.Mode,
		Summary:      "Async Azure Blob scan page was scanned.",
		Details: map[string]any{
			"source_id":  job.SourceID,
			"auth_mode":  authMode,
			"item_count": len(items),
			"dry_run":    job.DryRun,
			"scanned_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

type asyncAzureBlobNormalizedBatch struct {
	Version      string                           `json:"version"`
	JobID        string                           `json:"job_id"`
	SourceID     string                           `json:"source_id"`
	DataStoreID  string                           `json:"data_store_id"`
	BatchID      string                           `json:"batch_id"`
	SourceSystem string                           `json:"source_system"`
	UpdatedAt    string                           `json:"updated_at"`
	Records      []asyncAzureBlobNormalizedRecord `json:"records"`
}

type asyncAzureBlobNormalizedRecord struct {
	FileID       string `json:"file_id"`
	SourceID     string `json:"source_id"`
	DataStoreID  string `json:"data_store_id"`
	Container    string `json:"container"`
	Path         string `json:"path"`
	Name         string `json:"name"`
	Prefix       string `json:"prefix,omitempty"`
	SizeBytes    int64  `json:"size_bytes"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	AccessTier   string `json:"access_tier,omitempty"`
	ScannedAt    string `json:"scanned_at"`
}

func (s *Server) persistAzureBlobAsyncNormalizedPage(job controlplane.AzureBlobScanJob, containerName string, prefix string, items []azureBlobWorkbenchBlob) (int, string, error) {
	if len(items) == 0 {
		return 0, "", nil
	}
	root := strings.TrimSpace(s.cfg.AzureBlobAsyncResultsPath)
	if root == "" {
		root = "data/async-normalized"
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return 0, "", fmt.Errorf("create async normalized directory: %w", err)
	}
	path := filepath.Join(root, sanitizeAsyncArtifactName(job.JobID)+".json")

	batch := asyncAzureBlobNormalizedBatch{
		Version:      "azure_blob_async_normalized_v1",
		JobID:        job.JobID,
		SourceID:     job.SourceID,
		DataStoreID:  job.DataStoreID,
		BatchID:      job.BatchID,
		SourceSystem: "azure_blob",
		Records:      []asyncAzureBlobNormalizedRecord{},
	}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &batch)
		if batch.Records == nil {
			batch.Records = []asyncAzureBlobNormalizedRecord{}
		}
	}

	scannedAt := time.Now().UTC().Format(time.RFC3339)
	seen := map[string]bool{}
	for _, record := range batch.Records {
		seen[record.FileID] = true
	}
	written := 0
	for _, item := range items {
		fileID := "azureblob:" + containerName + ":" + item.Name
		if seen[fileID] {
			continue
		}
		batch.Records = append(batch.Records, asyncAzureBlobNormalizedRecord{
			FileID:       fileID,
			SourceID:     job.SourceID,
			DataStoreID:  job.DataStoreID,
			Container:    containerName,
			Path:         item.Name,
			Name:         filepath.Base(item.Name),
			Prefix:       prefix,
			SizeBytes:    item.SizeBytes,
			ContentType:  item.ContentType,
			LastModified: item.LastModified,
			AccessTier:   item.AccessTier,
			ScannedAt:    scannedAt,
		})
		seen[fileID] = true
		written++
	}
	batch.UpdatedAt = scannedAt

	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return 0, "", fmt.Errorf("encode async normalized batch: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return 0, "", fmt.Errorf("write async normalized batch: %w", err)
	}
	return written, path, nil
}

func sanitizeAsyncArtifactName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "azureblob-async-scan"
	}
	var b strings.Builder
	lastSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastSeparator = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			lastSeparator = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSeparator = false
		case r == '-' || r == '_':
			b.WriteRune(r)
			lastSeparator = false
		default:
			if !lastSeparator {
				b.WriteByte('_')
				lastSeparator = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func shouldStopAsyncAzureBlobScan(job controlplane.AzureBlobScanJob) bool {
	return job.Limit > 0 && job.Checkpoint.ObjectsScanned >= job.Limit
}

func nextAzureBlobAsyncCheckpoint(job controlplane.AzureBlobScanJob, scopeIndex int, prefixIndex int) (controlplane.AzureBlobScanCheckpoint, bool) {
	checkpoint := job.Checkpoint
	checkpoint.PageMarker = ""
	if scopeIndex < 0 || scopeIndex >= len(job.Scopes) {
		return checkpoint, false
	}
	scope := job.Scopes[scopeIndex]
	if prefixIndex+1 < len(scope.Prefixes) {
		checkpoint.ScopeIndex = scopeIndex
		checkpoint.PrefixIndex = prefixIndex + 1
		checkpoint.Container = scope.Container
		checkpoint.Prefix = strings.Trim(strings.TrimSpace(scope.Prefixes[prefixIndex+1]), "/")
		return checkpoint, true
	}
	if scopeIndex+1 < len(job.Scopes) {
		nextScope := job.Scopes[scopeIndex+1]
		checkpoint.ScopeIndex = scopeIndex + 1
		checkpoint.PrefixIndex = 0
		checkpoint.Container = nextScope.Container
		checkpoint.Prefix = ""
		if len(nextScope.Prefixes) > 0 {
			checkpoint.Prefix = strings.Trim(strings.TrimSpace(nextScope.Prefixes[0]), "/")
		}
		return checkpoint, true
	}
	return checkpoint, false
}
