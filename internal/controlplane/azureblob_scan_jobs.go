package controlplane

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type AzureBlobScanScope struct {
	Container string   `json:"container"`
	Prefixes  []string `json:"prefixes,omitempty"`
}

type AzureBlobScanJobRequest struct {
	SourceID         string                           `json:"source_id"`
	DataStoreID      string                           `json:"data_store_id"`
	BatchID          string                           `json:"batch_id,omitempty"`
	RequestedBy      string                           `json:"requested_by,omitempty"`
	Mode             string                           `json:"mode,omitempty"`
	Scopes           []AzureBlobScanScope             `json:"scopes"`
	Limit            int                              `json:"limit,omitempty"`
	DryRun           bool                             `json:"dry_run"`
	RequestEnvelope  *contracts.ActionRequestEnvelope `json:"request_envelope,omitempty"`
	GuardianDecision *contracts.GuardianDecision      `json:"guardian_decision,omitempty"`
}

type AzureBlobScanCheckpoint struct {
	ScopeIndex     int    `json:"scope_index"`
	PrefixIndex    int    `json:"prefix_index"`
	Container      string `json:"container,omitempty"`
	Prefix         string `json:"prefix,omitempty"`
	PageMarker     string `json:"page_marker,omitempty"`
	ObjectsScanned int    `json:"objects_scanned"`
	PagesScanned   int    `json:"pages_scanned"`
	UpdatedAt      string `json:"updated_at"`
}

type AzureBlobScanJob struct {
	JobID               string                           `json:"job_id"`
	RequestID           string                           `json:"request_id,omitempty"`
	CorrelationID       string                           `json:"correlation_id,omitempty"`
	SourceID            string                           `json:"source_id"`
	DataStoreID         string                           `json:"data_store_id"`
	BatchID             string                           `json:"batch_id"`
	RequestedBy         string                           `json:"requested_by,omitempty"`
	Mode                string                           `json:"mode"`
	Status              string                           `json:"status"`
	DryRun              bool                             `json:"dry_run"`
	Limit               int                              `json:"limit,omitempty"`
	Scopes              []AzureBlobScanScope             `json:"scopes"`
	Checkpoint          AzureBlobScanCheckpoint          `json:"checkpoint"`
	Attempts            int                              `json:"attempts"`
	RetryCount          int                              `json:"retry_count,omitempty"`
	ThrottleCount       int                              `json:"throttle_count,omitempty"`
	RecordsWritten      int                              `json:"records_written,omitempty"`
	NormalizedPath      string                           `json:"normalized_path,omitempty"`
	PolicyBundleVersion string                           `json:"policy_bundle_version,omitempty"`
	PolicyBundleHash    string                           `json:"policy_bundle_hash,omitempty"`
	GuardianDecisionID  string                           `json:"guardian_decision_id,omitempty"`
	RequestEnvelope     *contracts.ActionRequestEnvelope `json:"request_envelope,omitempty"`
	LeaseOwner          string                           `json:"lease_owner,omitempty"`
	LeaseExpiresAt      string                           `json:"lease_expires_at,omitempty"`
	HeartbeatAt         string                           `json:"heartbeat_at,omitempty"`
	RetryAfter          string                           `json:"retry_after,omitempty"`
	LastError           string                           `json:"last_error,omitempty"`
	CancelRequested     bool                             `json:"cancel_requested,omitempty"`
	CreatedAt           string                           `json:"created_at"`
	UpdatedAt           string                           `json:"updated_at"`
	CompletedAt         string                           `json:"completed_at,omitempty"`
}

type AzureBlobScanJobStore struct {
	mu   sync.RWMutex
	path string
	jobs map[string]AzureBlobScanJob
}

func NewAzureBlobScanJobStore() *AzureBlobScanJobStore {
	return &AzureBlobScanJobStore{jobs: map[string]AzureBlobScanJob{}}
}

func NewAzureBlobScanJobStoreWithPath(path string) *AzureBlobScanJobStore {
	store := &AzureBlobScanJobStore{
		path: strings.TrimSpace(path),
		jobs: map[string]AzureBlobScanJob{},
	}
	store.load()
	return store
}

func (s *AzureBlobScanJobStore) Start(req AzureBlobScanJobRequest) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	req.SourceID = cleanScanValue(req.SourceID)
	req.DataStoreID = cleanScanValue(firstScanNonEmpty(req.DataStoreID, req.SourceID))
	if req.SourceID == "" {
		return AzureBlobScanJob{}, fmt.Errorf("source_id is required")
	}
	if req.DataStoreID == "" {
		return AzureBlobScanJob{}, fmt.Errorf("data_store_id is required")
	}
	scopes := normalizeAzureBlobScanScopes(req.Scopes)
	if len(scopes) == 0 {
		return AzureBlobScanJob{}, fmt.Errorf("at least one container scope is required")
	}

	now := scanNow()
	job := AzureBlobScanJob{
		JobID:               "scan-azureblob-" + time.Now().UTC().Format("20060102-150405.000000000"),
		RequestID:           requestIDFromScanRequest(req),
		CorrelationID:       correlationIDFromScanRequest(req),
		SourceID:            req.SourceID,
		DataStoreID:         req.DataStoreID,
		BatchID:             cleanScanValue(firstScanNonEmpty(req.BatchID, "batch-"+time.Now().UTC().Format("20060102-150405"))),
		RequestedBy:         cleanScanValue(firstScanNonEmpty(req.RequestedBy, "portal-user")),
		Mode:                cleanScanValue(firstScanNonEmpty(req.Mode, "metadata_scan")),
		Status:              "queued",
		DryRun:              req.DryRun,
		Limit:               req.Limit,
		Scopes:              scopes,
		PolicyBundleVersion: policyVersionFromScanRequest(req),
		PolicyBundleHash:    policyHashFromScanRequest(req),
		GuardianDecisionID:  decisionIDFromScanRequest(req),
		RequestEnvelope:     req.RequestEnvelope,
		Checkpoint: AzureBlobScanCheckpoint{
			ScopeIndex:  0,
			PrefixIndex: 0,
			Container:   scopes[0].Container,
			Prefix:      firstScanPrefix(scopes[0]),
			UpdatedAt:   now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.JobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) List() []AzureBlobScanJob {
	if s == nil {
		return []AzureBlobScanJob{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]AzureBlobScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		out = append(out, job)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt > out[j].CreatedAt
	})
	return out
}

func (s *AzureBlobScanJobStore) StaleLeasedJobs(now time.Time) []AzureBlobScanJob {
	if s == nil {
		return []AzureBlobScanJob{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := []AzureBlobScanJob{}
	for _, job := range s.jobs {
		if job.LeaseOwner == "" {
			continue
		}
		if job.Status != "running" {
			continue
		}
		if leaseExpired(job.LeaseExpiresAt, now) {
			out = append(out, job)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func (s *AzureBlobScanJobStore) ReclaimStaleLeases(now time.Time) []AzureBlobScanJob {
	if s == nil {
		return []AzureBlobScanJob{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	out := []AzureBlobScanJob{}
	for _, job := range s.jobs {
		if job.LeaseOwner == "" || job.Status != "running" || !leaseExpired(job.LeaseExpiresAt, now) {
			continue
		}
		job.Status = "paused"
		job.LeaseOwner = ""
		job.LeaseExpiresAt = ""
		job.HeartbeatAt = ""
		job.LastError = "Worker heartbeat expired; job paused for operator-controlled resume."
		job.UpdatedAt = scanNow()
		s.jobs[job.JobID] = job
		out = append(out, job)
	}
	if len(out) > 0 {
		s.saveLocked()
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func (s *AzureBlobScanJobStore) Get(jobID string) (AzureBlobScanJob, bool) {
	if s == nil {
		return AzureBlobScanJob{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[cleanScanValue(jobID)]
	return job, ok
}

func (s *AzureBlobScanJobStore) Resume(jobID string) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	switch job.Status {
	case "running":
		return job, nil
	case "completed":
		return job, fmt.Errorf("scan job %s is already completed", jobID)
	case "cancelled":
		return job, fmt.Errorf("scan job %s is cancelled; create a new scan job", jobID)
	}
	job.Status = "running"
	job.UpdatedAt = scanNow()
	job.Attempts++
	job.CancelRequested = false
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) Pause(jobID string) (AzureBlobScanJob, error) {
	job, err := s.transition(jobID, "paused", "")
	if err != nil {
		return job, err
	}
	s.mu.Lock()
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	job.HeartbeatAt = ""
	s.jobs[job.JobID] = job
	s.saveLocked()
	s.mu.Unlock()
	return job, nil
}

func (s *AzureBlobScanJobStore) Cancel(jobID string) (AzureBlobScanJob, error) {
	job, err := s.transition(jobID, "cancelled", "")
	if err != nil {
		return job, err
	}
	job.CancelRequested = true
	job.CompletedAt = scanNow()
	s.mu.Lock()
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	job.HeartbeatAt = ""
	s.jobs[job.JobID] = job
	s.saveLocked()
	s.mu.Unlock()
	return job, nil
}

func (s *AzureBlobScanJobStore) Fail(jobID string, reason string) (AzureBlobScanJob, error) {
	return s.transition(jobID, "failed", reason)
}

func (s *AzureBlobScanJobStore) Complete(jobID string) (AzureBlobScanJob, error) {
	job, err := s.transition(jobID, "completed", "")
	if err != nil {
		return job, err
	}
	job.CompletedAt = scanNow()
	s.mu.Lock()
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	job.HeartbeatAt = ""
	s.jobs[job.JobID] = job
	s.saveLocked()
	s.mu.Unlock()
	return job, nil
}

func (s *AzureBlobScanJobStore) UpdateCheckpoint(jobID string, checkpoint AzureBlobScanCheckpoint) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	checkpoint.UpdatedAt = scanNow()
	job.Checkpoint = checkpoint
	job.UpdatedAt = checkpoint.UpdatedAt
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) RecordNormalizedWrite(jobID string, normalizedPath string, recordsWritten int) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	if recordsWritten > 0 {
		job.RecordsWritten += recordsWritten
	}
	if strings.TrimSpace(normalizedPath) != "" {
		job.NormalizedPath = strings.TrimSpace(normalizedPath)
	}
	job.UpdatedAt = scanNow()
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) AcquireLease(jobID string, workerID string, ttl time.Duration) (AzureBlobScanJob, bool, error) {
	if s == nil {
		return AzureBlobScanJob{}, false, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	workerID = cleanScanValue(workerID)
	if workerID == "" {
		return AzureBlobScanJob{}, false, fmt.Errorf("worker_id is required")
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, false, fmt.Errorf("scan job %s not found", jobID)
	}
	if job.Status == "completed" || job.Status == "cancelled" {
		return job, false, nil
	}
	now := time.Now().UTC()
	if job.LeaseOwner != "" && job.LeaseOwner != workerID && !leaseExpired(job.LeaseExpiresAt, now) {
		return job, false, nil
	}
	job.LeaseOwner = workerID
	job.HeartbeatAt = now.Format(time.RFC3339)
	job.LeaseExpiresAt = now.Add(ttl).Format(time.RFC3339)
	job.UpdatedAt = job.HeartbeatAt
	s.jobs[jobID] = job
	s.saveLocked()
	return job, true, nil
}

func (s *AzureBlobScanJobStore) HeartbeatLease(jobID string, workerID string, ttl time.Duration) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	workerID = cleanScanValue(workerID)
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	if job.LeaseOwner != workerID {
		return job, fmt.Errorf("scan job %s is leased by another worker", jobID)
	}
	now := time.Now().UTC()
	job.HeartbeatAt = now.Format(time.RFC3339)
	job.LeaseExpiresAt = now.Add(ttl).Format(time.RFC3339)
	job.UpdatedAt = job.HeartbeatAt
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) ReleaseLease(jobID string, workerID string) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	workerID = cleanScanValue(workerID)

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	if job.LeaseOwner != "" && job.LeaseOwner != workerID {
		return job, fmt.Errorf("scan job %s is leased by another worker", jobID)
	}
	job.LeaseOwner = ""
	job.LeaseExpiresAt = ""
	job.HeartbeatAt = ""
	job.UpdatedAt = scanNow()
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) RecordRetry(jobID string, reason string, retryAfter time.Duration, throttled bool) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	job.RetryCount++
	if throttled {
		job.ThrottleCount++
	}
	if reason != "" {
		job.LastError = reason
	}
	if retryAfter > 0 {
		job.RetryAfter = time.Now().UTC().Add(retryAfter).Format(time.RFC3339)
	}
	job.UpdatedAt = scanNow()
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

func (s *AzureBlobScanJobStore) transition(jobID string, status string, reason string) (AzureBlobScanJob, error) {
	if s == nil {
		return AzureBlobScanJob{}, fmt.Errorf("scan job store is not configured")
	}
	jobID = cleanScanValue(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return AzureBlobScanJob{}, fmt.Errorf("scan job %s not found", jobID)
	}
	job.Status = cleanScanValue(status)
	job.UpdatedAt = scanNow()
	if reason != "" {
		job.LastError = reason
	}
	if status == "running" {
		job.Attempts++
		job.CancelRequested = false
	}
	s.jobs[jobID] = job
	s.saveLocked()
	return job, nil
}

type azureBlobScanJobState struct {
	Version string             `json:"version"`
	Jobs    []AzureBlobScanJob `json:"jobs"`
}

func (s *AzureBlobScanJobStore) load() {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil || len(data) == 0 {
		return
	}
	var state azureBlobScanJobState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if s.jobs == nil {
		s.jobs = map[string]AzureBlobScanJob{}
	}
	for _, job := range state.Jobs {
		if strings.TrimSpace(job.JobID) != "" {
			s.jobs[job.JobID] = job
		}
	}
}

func (s *AzureBlobScanJobStore) saveLocked() {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return
	}
	jobs := make([]AzureBlobScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt > jobs[j].CreatedAt
	})
	state := azureBlobScanJobState{
		Version: "azure_blob_scan_jobs_v1",
		Jobs:    jobs,
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0600)
}

func normalizeAzureBlobScanScopes(scopes []AzureBlobScanScope) []AzureBlobScanScope {
	out := []AzureBlobScanScope{}
	seen := map[string]bool{}
	for _, scope := range scopes {
		container := cleanScanValue(scope.Container)
		if container == "" {
			continue
		}
		prefixes := []string{}
		for _, prefix := range scope.Prefixes {
			prefix = strings.Trim(strings.TrimSpace(prefix), "/")
			if prefix != "" {
				prefixes = append(prefixes, prefix)
			}
		}
		if len(prefixes) == 0 {
			prefixes = []string{""}
		}
		key := container + "\x00" + strings.Join(prefixes, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, AzureBlobScanScope{Container: container, Prefixes: prefixes})
	}
	return out
}

func leaseExpired(value string, now time.Time) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return true
	}
	return !expiresAt.After(now)
}

func firstScanPrefix(scope AzureBlobScanScope) string {
	if len(scope.Prefixes) == 0 {
		return ""
	}
	return scope.Prefixes[0]
}

func firstScanNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func cleanScanValue(value string) string {
	return strings.TrimSpace(value)
}

func scanNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func requestIDFromScanRequest(req AzureBlobScanJobRequest) string {
	if req.RequestEnvelope != nil && strings.TrimSpace(req.RequestEnvelope.RequestID) != "" {
		return strings.TrimSpace(req.RequestEnvelope.RequestID)
	}
	return ""
}

func correlationIDFromScanRequest(req AzureBlobScanJobRequest) string {
	if req.RequestEnvelope != nil && strings.TrimSpace(req.RequestEnvelope.CorrelationID) != "" {
		return strings.TrimSpace(req.RequestEnvelope.CorrelationID)
	}
	return ""
}

func policyVersionFromScanRequest(req AzureBlobScanJobRequest) string {
	if req.GuardianDecision != nil && strings.TrimSpace(req.GuardianDecision.PolicyBundleVersion) != "" {
		return strings.TrimSpace(req.GuardianDecision.PolicyBundleVersion)
	}
	if req.RequestEnvelope != nil {
		return strings.TrimSpace(req.RequestEnvelope.PolicyContext.PolicyBundleVersion)
	}
	return ""
}

func policyHashFromScanRequest(req AzureBlobScanJobRequest) string {
	if req.GuardianDecision != nil && strings.TrimSpace(req.GuardianDecision.PolicyBundleHash) != "" {
		return strings.TrimSpace(req.GuardianDecision.PolicyBundleHash)
	}
	if req.RequestEnvelope != nil {
		return strings.TrimSpace(req.RequestEnvelope.PolicyContext.PolicyBundleHash)
	}
	return ""
}

func decisionIDFromScanRequest(req AzureBlobScanJobRequest) string {
	if req.GuardianDecision != nil && strings.TrimSpace(req.GuardianDecision.DecisionID) != "" {
		return strings.TrimSpace(req.GuardianDecision.DecisionID)
	}
	return ""
}
