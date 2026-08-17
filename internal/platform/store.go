package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	mu       sync.Mutex
	db       *sql.DB
	dbPath   string
	policies map[string]PolicyTemplate
}

func NewStore(dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		dbPath = "data/platform/everest-platform.sqlite"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:       db,
		dbPath:   dbPath,
		policies: map[string]PolicyTemplate{},
	}

	if err := s.initSchema(); err != nil {
		return nil, err
	}

	s.seedPolicies()
	return s, nil
}

func (s *Store) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			agent_id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS data_sources (
			data_store_id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS jobs (
			job_id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS leases (
			lease_id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS evidence (
			kind TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (kind, execution_id)
		)`,

		`CREATE TABLE IF NOT EXISTS audit_events (
			event_id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS approvals (
			approval_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			data_store_id TEXT NOT NULL,
			execution_id TEXT,
			operation TEXT NOT NULL,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) seedPolicies() {
	defaults := []PolicyTemplate{
		{
			PolicyID:              "azureblob-metadata-discovery",
			Name:                  "Azure Blob Metadata Discovery",
			Description:           "Read-only metadata scan, quick analytics, manifest, and provenance.",
			CompiledPolicyVersion: "cpv-2026.05.29.001",
			WorkflowVersion:       "workflow-mvs-linear-v1",
			AllowedActions:        []string{"metadata_scan", "quick_analytics", "manifest_generation", "provenance_generation"},
			DryRunOnlyActions:     []string{},
			GuardianRequired:      false,
			DefaultPolicy:         true,
		},
		{
			PolicyID:              "azureblob-confidential-tagging-preview",
			Name:                  "Confidential Data Tagging Preview",
			Description:           "Preview metadata and index tags for confidential/high-risk blobs in dry-run mode.",
			CompiledPolicyVersion: "cpv-2026.05.29.001",
			WorkflowVersion:       "workflow-mvs-linear-v1",
			AllowedActions:        []string{"dry_run_apply_blob_metadata", "dry_run_apply_blob_index_tags"},
			DryRunOnlyActions:     []string{"apply_blob_metadata", "apply_blob_index_tags"},
			GuardianRequired:      true,
			DefaultPolicy:         true,
		},
		{
			PolicyID:              "azureblob-approved-tagging",
			Name:                  "Approved Azure Metadata / Index Tagging",
			Description:           "Apply Azure Blob metadata and index tags after Guardian approval.",
			CompiledPolicyVersion: "cpv-2026.05.29.001",
			WorkflowVersion:       "workflow-mvs-linear-v1",
			AllowedActions:        []string{"apply_blob_metadata", "apply_blob_index_tags"},
			DryRunOnlyActions:     []string{},
			GuardianRequired:      true,
			DefaultPolicy:         true,
		},
		{
			PolicyID:              "destructive-action-safety",
			Name:                  "Destructive Action Safety Policy",
			Description:           "Requires HITL and Guardian controls before delete, archive, retrieve, quarantine, and migration actions.",
			CompiledPolicyVersion: "cpv-2026.05.29.001",
			WorkflowVersion:       "workflow-mvs-linear-v1",
			AllowedActions:        []string{"dry_run_delete", "dry_run_archive", "dry_run_retrieve", "dry_run_quarantine", "dry_run_migration", "delete", "archive", "retrieve", "rehydrate", "quarantine", "migration"},
			DryRunOnlyActions:     []string{"delete", "archive", "retrieve", "quarantine", "migration"},
			GuardianRequired:      true,
			DefaultPolicy:         true,
		},
	}

	for _, p := range defaults {
		s.policies[p.PolicyID] = p
	}
}

func (s *Store) upsertJSON(table string, idColumn string, id string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s, data, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(%s) DO UPDATE SET
			data = excluded.data,
			updated_at = excluded.updated_at
	`, table, idColumn, idColumn)

	_, err = s.db.Exec(query, id, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func scanJSON[T any](row *sql.Row) (T, bool) {
	var zero T
	var data string

	if err := row.Scan(&data); err != nil {
		return zero, false
	}

	var out T
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return zero, false
	}

	return out, true
}

func listJSON[T any](rows *sql.Rows) ([]T, error) {
	defer rows.Close()

	items := []T{}

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}

		var item T
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *Store) RegisterAgent(req AgentRegistrationRequest) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(req.AgentID) == "" {
		return Agent{}, errors.New("agent_id is required")
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return Agent{}, errors.New("tenant_id is required")
	}

	now := time.Now().UTC()

	existing, ok := s.getAgentLocked(req.AgentID)

	agent := Agent{
		AgentID:       req.AgentID,
		TenantID:      req.TenantID,
		Hostname:      req.Hostname,
		Version:       req.Version,
		OS:            req.OS,
		Capabilities:  req.Capabilities,
		Status:        "active",
		RegisteredAt:  now,
		LastHeartbeat: now,
	}

	if ok {
		agent.RegisteredAt = existing.RegisteredAt
	}

	if err := s.upsertJSON("agents", "agent_id", agent.AgentID, agent); err != nil {
		return Agent{}, err
	}

	_ = s.writeAuditEventLocked("agent_registered", map[string]any{
		"agent_id":  agent.AgentID,
		"tenant_id": agent.TenantID,
		"hostname":  agent.Hostname,
		"version":   agent.Version,
		"os":        agent.OS,
	})

	return agent, nil
}

func (s *Store) getAgentLocked(agentID string) (Agent, bool) {
	return scanJSON[Agent](s.db.QueryRow(`SELECT data FROM agents WHERE agent_id = ?`, agentID))
}

func (s *Store) ListAgents() []Agent {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT data FROM agents ORDER BY updated_at DESC`)
	if err != nil {
		return []Agent{}
	}

	items, err := listJSON[Agent](rows)
	if err != nil {
		return []Agent{}
	}

	return items
}

func (s *Store) GetAgent(agentID string) (Agent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.getAgentLocked(agentID)
}

func (s *Store) SaveDataSource(req DataSourceCreateRequest) (DataSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.DataStoreID == "" {
		return DataSource{}, errors.New("data_store_id is required")
	}
	if req.TenantID == "" {
		req.TenantID = "tenant-local-001"
	}
	if req.SourceSystem == "" {
		req.SourceSystem = "azure_blob"
	}
	if req.ProcessingTier == "" {
		req.ProcessingTier = "tier_2"
	}
	if req.ProcessingMode == "" {
		req.ProcessingMode = "standard"
	}
	if req.ActionMode == "" {
		req.ActionMode = "dry_run"
	}
	if req.AssignedPolicyID == "" {
		req.AssignedPolicyID = "azureblob-metadata-discovery"
	}
	if len(req.AllowedOperations) == 0 {
		req.AllowedOperations = []string{
			"metadata_scan",
			"apply_blob_metadata",
			"apply_blob_index_tags",
			"dry_run_delete",
			"dry_run_archive",
			"dry_run_retrieve",
			"dry_run_quarantine",
			"dry_run_migration",
			"archive",
			"retrieve",
			"rehydrate",
			"quarantine",
			"migration",
			"delete",
		}
	}

	now := time.Now().UTC()
	existing, exists := s.getDataSourceLocked(req.DataStoreID)

	ds := DataSource{
		TenantID:          req.TenantID,
		DataStoreID:       req.DataStoreID,
		DisplayName:       req.DisplayName,
		SourceSystem:      req.SourceSystem,
		StorageAccount:    req.StorageAccount,
		Containers:        req.Containers,
		CredentialRef:     req.CredentialRef,
		AssignedPolicyID:  req.AssignedPolicyID,
		ProcessingTier:    req.ProcessingTier,
		ProcessingMode:    req.ProcessingMode,
		ActionMode:        req.ActionMode,
		WritebackEnabled:  req.WritebackEnabled,
		AllowedOperations: req.AllowedOperations,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if exists {
		ds.CreatedAt = existing.CreatedAt
	}

	if err := s.upsertJSON("data_sources", "data_store_id", ds.DataStoreID, ds); err != nil {
		return DataSource{}, err
	}

	_ = s.writeAuditEventLocked("data_source_saved", map[string]any{
		"tenant_id":          ds.TenantID,
		"data_store_id":      ds.DataStoreID,
		"source_system":      ds.SourceSystem,
		"storage_account":    ds.StorageAccount,
		"containers":         ds.Containers,
		"assigned_policy_id": ds.AssignedPolicyID,
	})

	return ds, nil
}

func (s *Store) getDataSourceLocked(dataStoreID string) (DataSource, bool) {
	return scanJSON[DataSource](s.db.QueryRow(`SELECT data FROM data_sources WHERE data_store_id = ?`, dataStoreID))
}

func (s *Store) ListDataSources() []DataSource {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT data FROM data_sources ORDER BY updated_at DESC`)
	if err != nil {
		return []DataSource{}
	}

	items, err := listJSON[DataSource](rows)
	if err != nil {
		return []DataSource{}
	}

	return items
}

func (s *Store) GetDataSource(dataStoreID string) (DataSource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.getDataSourceLocked(dataStoreID)
}

func (s *Store) ListPolicies() []PolicyTemplate {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]PolicyTemplate, 0, len(s.policies))
	for _, p := range s.policies {
		out = append(out, p)
	}
	return out
}

func (s *Store) GetPolicy(policyID string) (PolicyTemplate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.policies[policyID]
	return p, ok
}

func (s *Store) CreateJob(req JobCreateRequest) (Job, []Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.TenantID == "" {
		req.TenantID = "tenant-local-001"
	}
	if req.DataStoreID == "" {
		return Job{}, nil, errors.New("data_store_id is required")
	}
	if req.PolicyID == "" {
		req.PolicyID = "azureblob-metadata-discovery"
	}
	if req.Operation == "" {
		req.Operation = "metadata_scan_and_process"
	}
	if req.ActionMode == "" {
		req.ActionMode = "dry_run"
	}

	ds, ok := s.getDataSourceLocked(req.DataStoreID)
	if !ok {
		return Job{}, nil, fmt.Errorf("data source not found: %s", req.DataStoreID)
	}

	policy, ok := s.policies[req.PolicyID]
	if !ok {
		return Job{}, nil, fmt.Errorf("policy not found: %s", req.PolicyID)
	}

	agentID := req.AgentID
	if agentID == "" {
		agentID = s.findAgentForCapabilityLocked("azure_blob")
	}
	if agentID == "" {
		return Job{}, nil, errors.New("no active Azure Blob capable agent found")
	}

	now := time.Now().UTC()
	idSuffix := now.Format("20060102-150405")

	jobID := "job-" + idSuffix
	leaseID := "lease-" + idSuffix
	executionID := "exec-" + idSuffix

	scope := selectScope(ds, req)

	lease := Lease{
		LeaseID:               leaseID,
		JobID:                 jobID,
		ExecutionID:           executionID,
		TenantID:              req.TenantID,
		AgentID:               agentID,
		DataStoreID:           ds.DataStoreID,
		SourceSystem:          ds.SourceSystem,
		StorageAccount:        ds.StorageAccount,
		Scope:                 scope,
		Operation:             req.Operation,
		CompiledPolicyVersion: policy.CompiledPolicyVersion,
		WorkflowVersion:       policy.WorkflowVersion,
		ProcessingTier:        ds.ProcessingTier,
		ProcessingMode:        ds.ProcessingMode,
		ActionMode:            req.ActionMode,
		WritebackEnabled:      req.WritebackEnabled,
		Checkpoint:            nil,
		Status:                "PENDING",
		CreatedAt:             now,
		ExpiresAt:             now.Add(30 * time.Minute),
	}

	job := Job{
		JobID:       jobID,
		TenantID:    req.TenantID,
		RequestedBy: req.RequestedBy,
		AgentID:     agentID,
		DataStoreID: ds.DataStoreID,
		PolicyID:    req.PolicyID,
		Operation:   req.Operation,
		Status:      "LEASE_CREATED",
		LeaseIDs:    []string{leaseID},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.upsertJSON("jobs", "job_id", job.JobID, job); err != nil {
		return Job{}, nil, err
	}
	if err := s.upsertJSON("leases", "lease_id", lease.LeaseID, lease); err != nil {
		return Job{}, nil, err
	}

	_ = s.writeAuditEventLocked("job_created", map[string]any{
		"job_id":        job.JobID,
		"lease_id":      lease.LeaseID,
		"execution_id":  lease.ExecutionID,
		"tenant_id":     job.TenantID,
		"agent_id":      job.AgentID,
		"data_store_id": job.DataStoreID,
		"policy_id":     job.PolicyID,
		"operation":     job.Operation,
	})

	return job, []Lease{lease}, nil
}

func selectScope(ds DataSource, req JobCreateRequest) []AzureBlobContainerScope {
	if len(req.Containers) == 0 {
		return ds.Containers
	}

	selected := make([]AzureBlobContainerScope, 0, len(req.Containers))
	for _, name := range req.Containers {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		selected = append(selected, AzureBlobContainerScope{
			Container: name,
			Prefixes:  []string{req.Prefix},
		})
	}

	return selected
}

func (s *Store) findAgentForCapabilityLocked(capability string) string {
	rows, err := s.db.Query(`SELECT data FROM agents`)
	if err != nil {
		return ""
	}
	defer rows.Close()

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var agent Agent
		if err := json.Unmarshal([]byte(data), &agent); err != nil {
			continue
		}

		if agent.Status != "active" {
			continue
		}

		for _, c := range agent.Capabilities {
			if c == capability {
				return agent.AgentID
			}
		}
	}

	return ""
}

func (s *Store) ListJobs() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT data FROM jobs ORDER BY updated_at DESC`)
	if err != nil {
		return []Job{}
	}

	items, err := listJSON[Job](rows)
	if err != nil {
		return []Job{}
	}

	return items
}

func (s *Store) NextLease(agentID string) (Lease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT data FROM leases ORDER BY updated_at ASC`)
	if err != nil {
		return Lease{}, false
	}
	defer rows.Close()

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var lease Lease
		if err := json.Unmarshal([]byte(data), &lease); err != nil {
			continue
		}

		if lease.AgentID == agentID && lease.Status == "PENDING" {
			lease.Status = "ASSIGNED"
			lease.LastHeartbeat = time.Now().UTC()
			_ = s.upsertJSON("leases", "lease_id", lease.LeaseID, lease)
			return lease, true
		}
	}

	return Lease{}, false
}

func (s *Store) HeartbeatLease(leaseID string, req LeaseHeartbeatRequest) (Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lease, ok := scanJSON[Lease](s.db.QueryRow(`SELECT data FROM leases WHERE lease_id = ?`, leaseID))
	if !ok {
		return Lease{}, fmt.Errorf("lease not found: %s", leaseID)
	}

	if req.Status != "" {
		lease.Status = req.Status
	}
	lease.LastHeartbeat = time.Now().UTC()

	if err := s.upsertJSON("leases", "lease_id", lease.LeaseID, lease); err != nil {
		return Lease{}, err
	}

	job, ok := scanJSON[Job](s.db.QueryRow(`SELECT data FROM jobs WHERE job_id = ?`, lease.JobID))
	if ok {
		job.Status = "LEASE_" + lease.Status
		job.UpdatedAt = time.Now().UTC()
		_ = s.upsertJSON("jobs", "job_id", job.JobID, job)
	}

	_ = s.writeAuditEventLocked("lease_heartbeat", map[string]any{
		"lease_id":          lease.LeaseID,
		"job_id":            lease.JobID,
		"execution_id":      lease.ExecutionID,
		"agent_id":          lease.AgentID,
		"status":            lease.Status,
		"records_processed": req.RecordsProcessed,
		"message":           req.Message,
	})

	return lease, nil
}

func (s *Store) SaveExecutionStatus(executionID string, status any) error {
	return s.saveEvidence("execution_status", executionID, status)
}

func (s *Store) GetExecutionStatus(executionID string) (any, bool) {
	return s.getEvidence("execution_status", executionID)
}

func (s *Store) SaveManifestSummary(executionID string, summary any) error {
	return s.saveEvidence("manifest_summary", executionID, summary)
}

func (s *Store) GetManifestSummary(executionID string) (any, bool) {
	return s.getEvidence("manifest_summary", executionID)
}

func (s *Store) SaveManifestDetails(executionID string, details []any) error {
	return s.saveEvidence("manifest_details", executionID, details)
}

func (s *Store) GetManifestDetails(executionID string) ([]any, bool) {
	v, ok := s.getEvidence("manifest_details", executionID)
	if !ok {
		return nil, false
	}

	items, ok := v.([]any)
	return items, ok
}

func (s *Store) SaveProvenance(executionID string, records []any) error {
	return s.saveEvidence("provenance", executionID, records)
}

func (s *Store) GetProvenance(executionID string) ([]any, bool) {
	v, ok := s.getEvidence("provenance", executionID)
	if !ok {
		return nil, false
	}

	items, ok := v.([]any)
	return items, ok
}

func (s *Store) SaveActionResults(executionID string, records []any) error {
	return s.saveEvidence("action_results", executionID, records)
}

func (s *Store) GetActionResults(executionID string) ([]any, bool) {
	v, ok := s.getEvidence("action_results", executionID)
	if !ok {
		return nil, false
	}

	items, ok := v.([]any)
	return items, ok
}

func (s *Store) saveEvidence(kind string, executionID string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(executionID) == "" {
		return errors.New("execution_id is required")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
	INSERT INTO evidence (kind, execution_id, data, updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(kind, execution_id) DO UPDATE SET
		data = excluded.data,
		updated_at = excluded.updated_at
`, kind, executionID, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}

	_ = s.writeAuditEventLocked("evidence_uploaded", map[string]any{
		"kind":         kind,
		"execution_id": executionID,
	})

	return nil
}

func (s *Store) getEvidence(kind string, executionID string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data string
	if err := s.db.QueryRow(`
		SELECT data FROM evidence
		WHERE kind = ? AND execution_id = ?
	`, kind, executionID).Scan(&data); err != nil {
		return nil, false
	}

	var out any
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, false
	}

	return out, true
}

func (s *Store) ProvisionAzureBlobQuickstart(req AzureBlobQuickstartRequest) (map[string]any, error) {
	if req.TenantID == "" {
		req.TenantID = "tenant-local-001"
	}
	if req.RequestedBy == "" {
		req.RequestedBy = "local-admin"
	}
	if req.ActivationToken == "" {
		req.ActivationToken = "local-activation-token"
	}
	if req.AgentID == "" {
		req.AgentID = "agentcell-win-001"
	}
	if req.Hostname == "" {
		req.Hostname = "localhost-agent"
	}
	if req.AgentVersion == "" {
		req.AgentVersion = "1.0.0"
	}
	if req.AgentOS == "" {
		req.AgentOS = "windows"
	}
	if req.DataStoreID == "" {
		req.DataStoreID = "azureblob-hulkb01"
	}
	if req.DisplayName == "" {
		req.DisplayName = "Azure Blob - hulkb01"
	}
	if req.StorageAccount == "" {
		req.StorageAccount = "hulk01"
	}
	if len(req.Containers) == 0 {
		req.Containers = []AzureBlobContainerScope{
			{
				Container: "hulkb01",
				Prefixes:  []string{""},
			},
		}
	}
	if req.CredentialRef == "" {
		req.CredentialRef = "secret://" + req.TenantID + "/" + req.DataStoreID
	}
	if req.PolicyID == "" {
		req.PolicyID = "azureblob-approved-tagging"
	}
	if req.ProcessingTier == "" {
		req.ProcessingTier = "tier_2"
	}
	if req.ProcessingMode == "" {
		req.ProcessingMode = "standard"
	}
	if req.ActionMode == "" {
		req.ActionMode = "dry_run"
	}

	agent, err := s.RegisterAgent(AgentRegistrationRequest{
		ActivationToken: req.ActivationToken,
		TenantID:        req.TenantID,
		AgentID:         req.AgentID,
		Hostname:        req.Hostname,
		Version:         req.AgentVersion,
		OS:              req.AgentOS,
		Capabilities: []string{
			"azure_blob",
			"metadata_scan",
			"apply_blob_metadata",
			"apply_blob_index_tags",
			"dry_run_actions",
			"local_execution_store",
		},
	})
	if err != nil {
		return nil, err
	}

	dataSource, err := s.SaveDataSource(DataSourceCreateRequest{
		TenantID:         req.TenantID,
		DataStoreID:      req.DataStoreID,
		DisplayName:      req.DisplayName,
		SourceSystem:     "azure_blob",
		StorageAccount:   req.StorageAccount,
		Containers:       req.Containers,
		CredentialRef:    req.CredentialRef,
		AssignedPolicyID: req.PolicyID,
		ProcessingTier:   req.ProcessingTier,
		ProcessingMode:   req.ProcessingMode,
		ActionMode:       req.ActionMode,
		WritebackEnabled: req.WritebackEnabled,
		AllowedOperations: []string{
			"metadata_scan",
			"apply_blob_metadata",
			"apply_blob_index_tags",
			"dry_run_delete",
			"dry_run_archive",
			"dry_run_retrieve",
			"dry_run_quarantine",
			"dry_run_migration",
			"archive",
			"retrieve",
			"rehydrate",
			"quarantine",
			"migration",
			"delete",
		},
	})
	if err != nil {
		return nil, err
	}

	job, leases, err := s.CreateJob(JobCreateRequest{
		TenantID:         req.TenantID,
		RequestedBy:      req.RequestedBy,
		AgentID:          req.AgentID,
		DataStoreID:      req.DataStoreID,
		PolicyID:         req.PolicyID,
		Operation:        "metadata_scan_and_process",
		Containers:       containerNames(req.Containers),
		Prefix:           "",
		ActionMode:       req.ActionMode,
		WritebackEnabled: req.WritebackEnabled,
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status":      "provisioned",
		"tenant_id":   req.TenantID,
		"agent":       agent,
		"data_source": dataSource,
		"job":         job,
		"leases":      leases,
		"next_step":   "Run the next assigned lease from the Data Agent.",
	}, nil
}

func containerNames(scopes []AzureBlobContainerScope) []string {
	out := []string{}
	for _, scope := range scopes {
		if strings.TrimSpace(scope.Container) != "" {
			out = append(out, scope.Container)
		}
	}
	return out
}

func (s *Store) StorageStatus() (StorageStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := os.Stat(s.dbPath)
	if err != nil {
		return StorageStatus{}, err
	}

	return StorageStatus{
		Status:       "ok",
		DatabasePath: s.dbPath,
		DatabaseSize: info.Size(),
		Agents:       s.countRowsLocked("agents"),
		DataSources:  s.countRowsLocked("data_sources"),
		Jobs:         s.countRowsLocked("jobs"),
		Leases:       s.countRowsLocked("leases"),
		EvidenceRows: s.countRowsLocked("evidence"),
		Approvals:    s.countRowsLocked("approvals"),
	}, nil
}

func (s *Store) countRowsLocked(table string) int {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := s.db.QueryRow(query).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) BackupStorage() (StorageBackupResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Ping(); err != nil {
		return StorageBackupResponse{}, err
	}

	backupDir := filepath.Join(filepath.Dir(s.dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return StorageBackupResponse{}, err
	}

	backupPath := filepath.Join(
		backupDir,
		"everest-platform-"+time.Now().UTC().Format("20060102-150405")+".sqlite",
	)

	if err := copyFile(s.dbPath, backupPath); err != nil {
		return StorageBackupResponse{}, err
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		return StorageBackupResponse{}, err
	}

	_ = s.writeAuditEventLocked("storage_backup", map[string]any{
		"backup_path": backupPath,
		"size_bytes":  info.Size(),
	})

	return StorageBackupResponse{
		Status:     "completed",
		BackupPath: backupPath,
		SizeBytes:  info.Size(),
	}, nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func (s *Store) ResetStorage(req StorageResetRequest) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Confirm != "RESET_EVEREST_PLATFORM_STORAGE" {
		return nil, errors.New("invalid confirmation text")
	}

	if req.Scope == "" {
		req.Scope = "jobs_and_evidence"
	}

	deleted := map[string]int{}

	switch req.Scope {
	case "evidence_only":
		deleted["evidence"] = s.deleteAllLocked("evidence")

	case "jobs_only":
		deleted["jobs"] = s.deleteAllLocked("jobs")
		deleted["leases"] = s.deleteAllLocked("leases")

	case "jobs_and_evidence":
		deleted["jobs"] = s.deleteAllLocked("jobs")
		deleted["leases"] = s.deleteAllLocked("leases")
		deleted["evidence"] = s.deleteAllLocked("evidence")
		deleted["approvals"] = s.deleteAllLocked("approvals")

	default:
		return nil, fmt.Errorf("unsupported reset scope: %s", req.Scope)
	}

	_ = s.writeAuditEventLocked("storage_reset", map[string]any{
		"scope":   req.Scope,
		"reason":  req.Reason,
		"deleted": deleted,
	})

	return map[string]any{
		"status":  "reset_completed",
		"scope":   req.Scope,
		"deleted": deleted,
	}, nil
}

func (s *Store) deleteAllLocked(table string) int {
	count := s.countRowsLocked(table)
	_, _ = s.db.Exec("DELETE FROM " + table)
	return count
}

func (s *Store) writeAuditEventLocked(eventType string, payload any) error {
	eventID := "audit-" + time.Now().UTC().Format("20060102-150405.000000000")

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO audit_events (event_id, event_type, data, created_at)
		VALUES (?, ?, ?, ?)
	`, eventID, eventType, string(data), time.Now().UTC().Format(time.RFC3339Nano))

	return err
}

func (s *Store) DashboardSummary() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	storageInfo, err := os.Stat(s.dbPath)
	dbSize := int64(0)
	if err == nil {
		dbSize = storageInfo.Size()
	}

	latestJob := s.latestJSONLocked("jobs")
	latestLease := s.latestJSONLocked("leases")
	latestExecution := s.latestEvidenceLocked("execution_status")
	latestManifest := s.latestEvidenceLocked("manifest_summary")
	latestExecutionID := firstStringValue(latestExecution, "execution_id")
	if latestExecutionID == "" {
		latestExecutionID = firstStringValue(latestManifest, "execution_id")
	}
	if latestExecutionID == "" {
		latestExecutionID = firstStringValue(latestLease, "execution_id")
	}

	totalObjects := firstIntValue(latestManifest, "total_records", "processed_records", "file_count")
	totalBytes := firstIntValue(latestManifest, "total_size_bytes", "total_bytes")
	criticalCount := firstIntValue(latestManifest, "critical_count", "critical")
	highCount := firstIntValue(latestManifest, "high_count", "high")
	actionCount := firstIntValue(latestManifest, "actions_planned", "action_count", "actions")
	actionMode := firstStringValue(latestLease, "action_mode")
	if actionMode == "" {
		actionMode = firstStringValue(latestManifest, "action_mode")
	}
	evidenceStatus := "pending"
	if latestManifest != nil {
		evidenceStatus = "available"
	}
	approvalCounts := s.approvalCountsLocked()

	return map[string]any{
		"status":        "ok",
		"agents":        s.countRowsLocked("agents"),
		"data_sources":  s.countRowsLocked("data_sources"),
		"jobs":          s.countRowsLocked("jobs"),
		"leases":        s.countRowsLocked("leases"),
		"evidence_rows": s.countRowsLocked("evidence"),
		"storage": map[string]any{
			"database_path":       s.dbPath,
			"database_size_bytes": dbSize,
		},
		"latest_job":              latestJob,
		"latest_lease":            latestLease,
		"latest_execution_status": latestExecution,
		"latest_manifest_summary": latestManifest,
		"latest_execution_id":     latestExecutionID,
		"total_objects":           totalObjects,
		"records_processed":       totalObjects,
		"total_bytes":             totalBytes,
		"critical_count":          criticalCount,
		"critical_risk":           criticalCount,
		"high_count":              highCount,
		"high_risk_count":         highCount,
		"high_risk":               highCount + criticalCount,
		"action_count":            actionCount,
		"actions":                 actionCount,
		"action_mode":             actionMode,
		"evidence_status":         evidenceStatus,
		"approvals":               approvalCounts,
		"pending_approvals":       approvalCounts["pending"],
		"approved_approvals":      approvalCounts["approved"],
		"rejected_approvals":      approvalCounts["rejected"],
		"writeback_policy": map[string]any{
			"real_writeback_supported": true,
			"real_writeback_actions": []string{
				"apply_blob_metadata",
				"apply_blob_index_tags",
			},
			"dry_run_only_actions": []string{
				"delete",
				"archive",
				"retrieve",
				"rehydrate",
				"quarantine",
				"migration",
			},
			"guardian_required":                             true,
			"human_review_required_for_destructive_actions": true,
		},
	}, nil
}

func (s *Store) LatestExecutionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var executionID string
	if err := s.db.QueryRow(`
		SELECT execution_id
		FROM evidence
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&executionID); err == nil && strings.TrimSpace(executionID) != "" {
		return executionID
	}

	latestLease := s.latestJSONLocked("leases")
	return firstStringValue(latestLease, "execution_id")
}

func firstStringValue(source any, keys ...string) string {
	m, ok := source.(map[string]any)
	if !ok {
		return ""
	}

	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	return ""
}

func firstIntValue(source any, keys ...string) int {
	m, ok := source.(map[string]any)
	if !ok {
		return 0
	}

	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}

		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case json.Number:
			i, _ := v.Int64()
			return int(i)
		}
	}

	return 0
}

func (s *Store) latestJSONLocked(table string) any {
	query := fmt.Sprintf("SELECT data FROM %s ORDER BY updated_at DESC LIMIT 1", table)

	var data string
	if err := s.db.QueryRow(query).Scan(&data); err != nil {
		return nil
	}

	var out any
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil
	}

	return out
}

func (s *Store) latestEvidenceLocked(kind string) any {
	var data string
	if err := s.db.QueryRow(`
		SELECT data
		FROM evidence
		WHERE kind = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, kind).Scan(&data); err != nil {
		return nil
	}

	var out any
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil
	}

	return out
}

func (s *Store) countActiveAgentsLocked() int {
	rows, err := s.db.Query(`SELECT data FROM agents`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	active := 0

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var agent Agent
		if err := json.Unmarshal([]byte(data), &agent); err != nil {
			continue
		}

		if agent.Status == "active" {
			active++
		}
	}

	return active
}

func (s *Store) ListAuditEvents(limit int) ([]AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := s.db.Query(`
		SELECT event_id, event_type, data, created_at
		FROM audit_events
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []AuditEvent{}

	for rows.Next() {
		var eventID string
		var eventType string
		var dataRaw string
		var createdAt string

		if err := rows.Scan(&eventID, &eventType, &dataRaw, &createdAt); err != nil {
			return nil, err
		}

		var data any
		if err := json.Unmarshal([]byte(dataRaw), &data); err != nil {
			data = dataRaw
		}

		events = append(events, AuditEvent{
			EventID:   eventID,
			EventType: eventType,
			Data:      data,
			CreatedAt: createdAt,
		})
	}

	return events, rows.Err()
}
