package portalstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

const Version = "portal_state_v1"

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

type State struct {
	Version          string                          `json:"version"`
	UpdatedAt        string                          `json:"updated_at"`
	AzureSources     []AzureSource                   `json:"azure_sources,omitempty"`
	Containers       []AzureContainer                `json:"containers,omitempty"`
	Blobs            []AzureBlob                     `json:"blobs,omitempty"`
	AIProviders      []AIProvider                    `json:"ai_providers,omitempty"`
	Jobs             []Job                           `json:"jobs,omitempty"`
	Policies         []Policy                        `json:"policies,omitempty"`
	WorkbenchHistory []WorkbenchHistory              `json:"workbench_history,omitempty"`
	EvidenceRecords  []EvidenceRecord                `json:"evidence_records,omitempty"`
	AuditEvents      []AuditEvent                    `json:"audit_events,omitempty"`
	ActionRecords    []ActionRecord                  `json:"action_records,omitempty"`
	DataPassports    []contracts.DataPassport        `json:"data_passports,omitempty"`
	SearchDocuments  []contracts.SearchIndexDocument `json:"search_documents,omitempty"`
}

type AzureSource struct {
	SourceID         string `json:"source_id"`
	DataStoreID      string `json:"data_store_id"`
	DisplayName      string `json:"display_name"`
	ConnectorID      string `json:"connector_id"`
	DeploymentMode   string `json:"deployment_mode"`
	StorageAccount   string `json:"storage_account,omitempty"`
	Container        string `json:"container,omitempty"`
	Prefix           string `json:"prefix,omitempty"`
	ActionMode       string `json:"action_mode,omitempty"`
	WritebackEnabled string `json:"writeback_enabled,omitempty"`
	CredentialSet    bool   `json:"credential_set"`
	LastUsedAt       string `json:"last_used_at,omitempty"`
	UpdatedAt        string `json:"updated_at"`
}

type AzureContainer struct {
	SourceID     string            `json:"source_id"`
	DataStoreID  string            `json:"data_store_id,omitempty"`
	Name         string            `json:"name"`
	LastModified string            `json:"last_modified,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	UpdatedAt    string            `json:"updated_at"`
}

type AzureBlob struct {
	SourceID     string `json:"source_id"`
	DataStoreID  string `json:"data_store_id,omitempty"`
	Container    string `json:"container"`
	Name         string `json:"name"`
	FileID       string `json:"file_id"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	AccessTier   string `json:"access_tier,omitempty"`
	UpdatedAt    string `json:"updated_at"`
}

type AIProvider struct {
	ProviderID      string   `json:"provider_id"`
	DisplayName     string   `json:"display_name,omitempty"`
	Status          string   `json:"status,omitempty"`
	Runtime         string   `json:"runtime,omitempty"`
	DeploymentModes []string `json:"deployment_modes,omitempty"`
	Model           string   `json:"model,omitempty"`
	NetworkMode     string   `json:"network_mode,omitempty"`
	Configured      bool     `json:"configured"`
	CredentialSet   bool     `json:"credential_set"`
	UpdatedAt       string   `json:"updated_at"`
}

type Job struct {
	JobID       string `json:"job_id"`
	DataStoreID string `json:"data_store_id,omitempty"`
	Status      string `json:"status,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

type Policy struct {
	PolicyID string `json:"policy_id"`
	Name     string `json:"name"`
	Effect   string `json:"effect,omitempty"`
}

type WorkbenchHistory struct {
	Operation   string `json:"operation"`
	Status      string `json:"status"`
	SourceID    string `json:"source_id,omitempty"`
	DataStoreID string `json:"data_store_id,omitempty"`
	Container   string `json:"container,omitempty"`
	Blob        string `json:"blob,omitempty"`
	Count       int    `json:"count,omitempty"`
	DryRun      bool   `json:"dry_run,omitempty"`
	Message     string `json:"message,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type EvidenceRecord struct {
	EvidenceID   string         `json:"evidence_id"`
	SubjectType  string         `json:"subject_type"`
	SubjectID    string         `json:"subject_id,omitempty"`
	EventType    string         `json:"event_type"`
	Status       string         `json:"status"`
	Severity     string         `json:"severity,omitempty"`
	SourceSystem string         `json:"source_system,omitempty"`
	DataStoreID  string         `json:"data_store_id,omitempty"`
	ExecutionID  string         `json:"execution_id,omitempty"`
	JobID        string         `json:"job_id,omitempty"`
	Container    string         `json:"container,omitempty"`
	Blob         string         `json:"blob,omitempty"`
	FileID       string         `json:"file_id,omitempty"`
	Operation    string         `json:"operation,omitempty"`
	ProviderID   string         `json:"provider_id,omitempty"`
	Summary      string         `json:"summary"`
	Details      map[string]any `json:"details,omitempty"`
	CreatedAt    string         `json:"created_at"`
}

type AuditEvent struct {
	EventID     string         `json:"event_id"`
	EventType   string         `json:"event_type"`
	Actor       string         `json:"actor,omitempty"`
	Role        string         `json:"role,omitempty"`
	Status      string         `json:"status"`
	TargetType  string         `json:"target_type,omitempty"`
	TargetID    string         `json:"target_id,omitempty"`
	SourceID    string         `json:"source_id,omitempty"`
	DataStoreID string         `json:"data_store_id,omitempty"`
	ExecutionID string         `json:"execution_id,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	DryRun      bool           `json:"dry_run,omitempty"`
	Summary     string         `json:"summary"`
	Details     map[string]any `json:"details,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

type ActionRecord struct {
	ActionID            string         `json:"action_id"`
	RecommendationID    string         `json:"recommendation_id,omitempty"`
	ApprovalID          string         `json:"approval_id,omitempty"`
	ExecutionID         string         `json:"execution_id,omitempty"`
	DecisionID          string         `json:"decision_id,omitempty"`
	FileID              string         `json:"file_id,omitempty"`
	BlobPath            string         `json:"blob_path,omitempty"`
	Container           string         `json:"container,omitempty"`
	Operation           string         `json:"operation"`
	Status              string         `json:"status"`
	RequestedBy         string         `json:"requested_by,omitempty"`
	ProviderID          string         `json:"provider_id,omitempty"`
	ActionMode          string         `json:"action_mode,omitempty"`
	WritebackEnabled    bool           `json:"writeback_enabled,omitempty"`
	GuardianDecision    string         `json:"guardian_decision,omitempty"`
	ApprovalEnforcement string         `json:"approval_enforcement,omitempty"`
	GuardianEnforcement string         `json:"guardian_enforcement,omitempty"`
	Reason              string         `json:"reason,omitempty"`
	Details             map[string]any `json:"details,omitempty"`
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
}

type Options struct {
	Status           string                          `json:"status"`
	Version          string                          `json:"version"`
	UpdatedAt        string                          `json:"updated_at"`
	AzureSources     []AzureSource                   `json:"azure_sources"`
	DataSources      []AzureSource                   `json:"data_sources"`
	Containers       []AzureContainer                `json:"containers"`
	Blobs            []AzureBlob                     `json:"blobs"`
	AIProviders      []AIProvider                    `json:"ai_providers"`
	Jobs             []Job                           `json:"jobs"`
	Policies         []Policy                        `json:"policies"`
	WorkbenchHistory []WorkbenchHistory              `json:"workbench_history"`
	EvidenceRecords  []EvidenceRecord                `json:"evidence_records"`
	AuditEvents      []AuditEvent                    `json:"audit_events"`
	ActionRecords    []ActionRecord                  `json:"action_records"`
	DataPassports    []contracts.DataPassport        `json:"data_passports"`
	SearchDocuments  []contracts.SearchIndexDocument `json:"search_documents"`
	DataStoreIDs     []string                        `json:"data_store_ids"`
	ContainerNames   []string                        `json:"container_names"`
	BlobNames        []string                        `json:"blob_names"`
	FileIDs          []string                        `json:"file_ids"`
	NextSteps        []string                        `json:"next_steps"`
}

func New(path string) *Store {
	s := &Store{
		path: path,
		state: State{
			Version:   Version,
			UpdatedAt: now(),
			Policies: []Policy{
				{PolicyID: "regional-data-residency", Name: "Regional Data Residency", Effect: "block_or_hitl"},
				{PolicyID: "destructive-action-safety", Name: "Destructive Action Safety", Effect: "hitl_required"},
				{PolicyID: "azureblob-approved-tagging", Name: "Azure Blob Approved Tagging", Effect: "allow_safe_tags"},
			},
		},
	}
	s.load()
	return s
}

func (s *Store) Options() Options {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := s.cloneLocked()
	dataStoreIDs := make([]string, 0, len(state.AzureSources))
	containerNames := make([]string, 0, len(state.Containers))
	blobNames := make([]string, 0, len(state.Blobs))
	fileIDs := make([]string, 0, len(state.Blobs))

	for _, source := range state.AzureSources {
		dataStoreIDs = append(dataStoreIDs, source.DataStoreID)
	}
	for _, item := range state.Containers {
		containerNames = append(containerNames, item.Name)
	}
	for _, item := range state.Blobs {
		blobNames = append(blobNames, item.Name)
		fileIDs = append(fileIDs, item.FileID)
	}

	nextSteps := []string{
		"Save or load an Azure Blob source.",
		"Discover containers to populate container dropdowns.",
		"List blobs to populate file/object dropdowns.",
		"Run a scan or AI recommendation after selecting an existing object.",
	}
	if len(state.AzureSources) > 0 && len(state.Containers) == 0 {
		nextSteps = []string{"Discover containers for the saved source."}
	}
	if len(state.Containers) > 0 && len(state.Blobs) == 0 {
		nextSteps = []string{"Select a container and list blobs to populate object suggestions."}
	}
	if len(state.Blobs) > 0 {
		nextSteps = []string{"Select an existing object, then run DSPM, AI recommendation, or a dry-run action."}
	}
	if state.AzureSources == nil {
		state.AzureSources = []AzureSource{}
	}
	if state.Containers == nil {
		state.Containers = []AzureContainer{}
	}
	if state.Blobs == nil {
		state.Blobs = []AzureBlob{}
	}
	if state.AIProviders == nil {
		state.AIProviders = []AIProvider{}
	}
	if state.Jobs == nil {
		state.Jobs = []Job{}
	}
	if state.Policies == nil {
		state.Policies = []Policy{}
	}
	if state.WorkbenchHistory == nil {
		state.WorkbenchHistory = []WorkbenchHistory{}
	}
	if state.EvidenceRecords == nil {
		state.EvidenceRecords = []EvidenceRecord{}
	}
	if state.AuditEvents == nil {
		state.AuditEvents = []AuditEvent{}
	}
	if state.ActionRecords == nil {
		state.ActionRecords = []ActionRecord{}
	}
	if state.DataPassports == nil {
		state.DataPassports = []contracts.DataPassport{}
	}
	if state.SearchDocuments == nil {
		state.SearchDocuments = []contracts.SearchIndexDocument{}
	}

	return Options{
		Status:           "ok",
		Version:          state.Version,
		UpdatedAt:        state.UpdatedAt,
		AzureSources:     state.AzureSources,
		DataSources:      state.AzureSources,
		Containers:       state.Containers,
		Blobs:            state.Blobs,
		AIProviders:      state.AIProviders,
		Jobs:             state.Jobs,
		Policies:         state.Policies,
		WorkbenchHistory: state.WorkbenchHistory,
		EvidenceRecords:  state.EvidenceRecords,
		AuditEvents:      state.AuditEvents,
		ActionRecords:    state.ActionRecords,
		DataPassports:    state.DataPassports,
		SearchDocuments:  state.SearchDocuments,
		DataStoreIDs:     uniqueSorted(dataStoreIDs),
		ContainerNames:   uniqueSorted(containerNames),
		BlobNames:        uniqueSorted(blobNames),
		FileIDs:          uniqueSorted(fileIDs),
		NextSteps:        nextSteps,
	}
}

func (s *Store) ListEvidence(limit int) []EvidenceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items := append([]EvidenceRecord(nil), s.state.EvidenceRecords...)
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *Store) ListAudit(limit int) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items := append([]AuditEvent(nil), s.state.AuditEvents...)
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *Store) ListActions(limit int) []ActionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items := append([]ActionRecord(nil), s.state.ActionRecords...)
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *Store) RecordEvidence(record EvidenceRecord) EvidenceRecord {
	record.EvidenceID = clean(firstNonEmpty(record.EvidenceID, "ev-"+time.Now().UTC().Format("20060102150405.000000000")))
	record.CreatedAt = firstNonEmpty(record.CreatedAt, now())
	record.Status = clean(firstNonEmpty(record.Status, "recorded"))
	record.SourceSystem = clean(firstNonEmpty(record.SourceSystem, "azure_blob"))
	if record.Summary == "" {
		record.Summary = record.EventType + " " + record.Status
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.EvidenceRecords = append([]EvidenceRecord{record}, s.state.EvidenceRecords...)
	if len(s.state.EvidenceRecords) > 1000 {
		s.state.EvidenceRecords = s.state.EvidenceRecords[:1000]
	}
	s.saveLocked()
	return record
}

func (s *Store) RecordAudit(event AuditEvent) AuditEvent {
	event.EventID = clean(firstNonEmpty(event.EventID, "audit-"+time.Now().UTC().Format("20060102150405.000000000")))
	event.CreatedAt = firstNonEmpty(event.CreatedAt, now())
	event.Status = clean(firstNonEmpty(event.Status, "recorded"))
	if event.Summary == "" {
		event.Summary = event.EventType + " " + event.Status
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.AuditEvents = append([]AuditEvent{event}, s.state.AuditEvents...)
	if len(s.state.AuditEvents) > 1000 {
		s.state.AuditEvents = s.state.AuditEvents[:1000]
	}
	s.saveLocked()
	return event
}

func (s *Store) UpsertAction(action ActionRecord) ActionRecord {
	action.ActionID = clean(firstNonEmpty(action.ActionID, "act-"+time.Now().UTC().Format("20060102150405.000000000")))
	action.Status = clean(firstNonEmpty(action.Status, "draft"))
	action.UpdatedAt = now()
	if action.CreatedAt == "" {
		action.CreatedAt = action.UpdatedAt
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.state.ActionRecords {
		if existing.ActionID == action.ActionID {
			action.CreatedAt = firstNonEmpty(existing.CreatedAt, action.CreatedAt)
			s.state.ActionRecords[i] = action
			s.saveLocked()
			return action
		}
	}

	s.state.ActionRecords = append([]ActionRecord{action}, s.state.ActionRecords...)
	if len(s.state.ActionRecords) > 1000 {
		s.state.ActionRecords = s.state.ActionRecords[:1000]
	}
	s.saveLocked()
	return action
}

func (s *Store) UpsertAzureSource(source AzureSource) {
	source.SourceID = clean(firstNonEmpty(source.SourceID, source.DataStoreID, source.StorageAccount, "azureblob-local"))
	source.DataStoreID = clean(firstNonEmpty(source.DataStoreID, source.SourceID))
	source.ConnectorID = clean(firstNonEmpty(source.ConnectorID, "azureblob"))
	source.DeploymentMode = clean(firstNonEmpty(source.DeploymentMode, "cloud_connected"))
	source.UpdatedAt = now()
	if source.LastUsedAt == "" {
		source.LastUsedAt = source.UpdatedAt
	}
	if source.DisplayName == "" {
		source.DisplayName = displayName(source)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.state.AzureSources {
		if existing.SourceID == source.SourceID || existing.DataStoreID == source.DataStoreID {
			source.CredentialSet = source.CredentialSet || existing.CredentialSet
			s.state.AzureSources[i] = mergeAzureSource(existing, source)
			s.saveLocked()
			return
		}
	}

	s.state.AzureSources = append(s.state.AzureSources, source)
	sort.Slice(s.state.AzureSources, func(i, j int) bool {
		return s.state.AzureSources[i].DisplayName < s.state.AzureSources[j].DisplayName
	})
	s.saveLocked()
}

func (s *Store) RemoveAzureSource(sourceID string) {
	sourceID = clean(sourceID)
	if sourceID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sources := s.state.AzureSources[:0]
	for _, source := range s.state.AzureSources {
		if source.SourceID == sourceID || source.DataStoreID == sourceID {
			continue
		}
		sources = append(sources, source)
	}
	s.state.AzureSources = sources

	containers := s.state.Containers[:0]
	for _, item := range s.state.Containers {
		if item.SourceID == sourceID || item.DataStoreID == sourceID {
			continue
		}
		containers = append(containers, item)
	}
	s.state.Containers = containers

	blobs := s.state.Blobs[:0]
	for _, item := range s.state.Blobs {
		if item.SourceID == sourceID || item.DataStoreID == sourceID {
			continue
		}
		blobs = append(blobs, item)
	}
	s.state.Blobs = blobs

	s.saveLocked()
}

func (s *Store) UpsertAzureContainers(source AzureSource, containers []AzureContainer) {
	if len(containers) == 0 {
		return
	}
	source.SourceID = clean(firstNonEmpty(source.SourceID, source.DataStoreID, "azureblob-local"))
	source.DataStoreID = clean(firstNonEmpty(source.DataStoreID, source.SourceID))
	now := now()

	s.UpsertAzureSource(source)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range containers {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			continue
		}
		item.SourceID = source.SourceID
		item.DataStoreID = source.DataStoreID
		item.UpdatedAt = now
		replaced := false
		for i, existing := range s.state.Containers {
			if existing.SourceID == item.SourceID && existing.Name == item.Name {
				s.state.Containers[i] = item
				replaced = true
				break
			}
		}
		if !replaced {
			s.state.Containers = append(s.state.Containers, item)
		}
	}

	sort.Slice(s.state.Containers, func(i, j int) bool {
		if s.state.Containers[i].SourceID != s.state.Containers[j].SourceID {
			return s.state.Containers[i].SourceID < s.state.Containers[j].SourceID
		}
		return s.state.Containers[i].Name < s.state.Containers[j].Name
	})
	s.saveLocked()
}

func (s *Store) UpsertAzureBlobs(source AzureSource, blobs []AzureBlob) {
	if len(blobs) == 0 {
		return
	}
	source.SourceID = clean(firstNonEmpty(source.SourceID, source.DataStoreID, "azureblob-local"))
	source.DataStoreID = clean(firstNonEmpty(source.DataStoreID, source.SourceID))
	now := now()

	s.UpsertAzureSource(source)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range blobs {
		item.Container = strings.TrimSpace(item.Container)
		item.Name = strings.Trim(strings.TrimSpace(item.Name), "/")
		if item.Container == "" || item.Name == "" {
			continue
		}
		item.SourceID = source.SourceID
		item.DataStoreID = source.DataStoreID
		item.FileID = firstNonEmpty(item.FileID, "azureblob:"+item.Container+":"+item.Name)
		item.UpdatedAt = now
		replaced := false
		for i, existing := range s.state.Blobs {
			if existing.SourceID == item.SourceID && existing.Container == item.Container && existing.Name == item.Name {
				s.state.Blobs[i] = item
				replaced = true
				break
			}
		}
		if !replaced {
			s.state.Blobs = append(s.state.Blobs, item)
		}
	}

	sort.Slice(s.state.Blobs, func(i, j int) bool {
		if s.state.Blobs[i].Container != s.state.Blobs[j].Container {
			return s.state.Blobs[i].Container < s.state.Blobs[j].Container
		}
		return s.state.Blobs[i].Name < s.state.Blobs[j].Name
	})
	if len(s.state.Blobs) > 2000 {
		s.state.Blobs = s.state.Blobs[len(s.state.Blobs)-2000:]
	}
	s.saveLocked()
}

func (s *Store) UpsertAIProvider(provider AIProvider) {
	provider.ProviderID = clean(provider.ProviderID)
	if provider.ProviderID == "" {
		return
	}
	provider.UpdatedAt = now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.state.AIProviders {
		if existing.ProviderID == provider.ProviderID {
			if provider.DisplayName == "" {
				provider.DisplayName = existing.DisplayName
			}
			if provider.Status == "" {
				provider.Status = existing.Status
			}
			if provider.Runtime == "" {
				provider.Runtime = existing.Runtime
			}
			if len(provider.DeploymentModes) == 0 {
				provider.DeploymentModes = existing.DeploymentModes
			}
			provider.Configured = provider.Configured || existing.Configured
			provider.CredentialSet = provider.CredentialSet || existing.CredentialSet
			s.state.AIProviders[i] = provider
			s.saveLocked()
			return
		}
	}

	s.state.AIProviders = append(s.state.AIProviders, provider)
	sort.Slice(s.state.AIProviders, func(i, j int) bool {
		return s.state.AIProviders[i].ProviderID < s.state.AIProviders[j].ProviderID
	})
	s.saveLocked()
}

func (s *Store) UpsertJob(job Job) {
	job.JobID = clean(job.JobID)
	if job.JobID == "" {
		return
	}
	job.UpdatedAt = now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.state.Jobs {
		if existing.JobID == job.JobID {
			s.state.Jobs[i] = job
			s.saveLocked()
			return
		}
	}
	s.state.Jobs = append(s.state.Jobs, job)
	s.saveLocked()
}

func (s *Store) RecordWorkbench(item WorkbenchHistory) {
	item.CreatedAt = firstNonEmpty(item.CreatedAt, now())

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.WorkbenchHistory = append([]WorkbenchHistory{item}, s.state.WorkbenchHistory...)
	if len(s.state.WorkbenchHistory) > 100 {
		s.state.WorkbenchHistory = s.state.WorkbenchHistory[:100]
	}
	s.saveLocked()
}

func (s *Store) load() {
	if s.path == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if state.Version == "" {
		state.Version = Version
	}
	if state.UpdatedAt == "" {
		state.UpdatedAt = now()
	}
	if len(state.Policies) == 0 {
		state.Policies = s.state.Policies
	}
	s.state = state
}

func (s *Store) saveLocked() {
	if s.path == "" {
		return
	}
	s.state.Version = Version
	s.state.UpdatedAt = now()
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0600)
}

func (s *Store) cloneLocked() State {
	data, err := json.Marshal(s.state)
	if err != nil {
		return s.state
	}
	var out State
	if err := json.Unmarshal(data, &out); err != nil {
		return s.state
	}
	return out
}

func mergeAzureSource(existing AzureSource, next AzureSource) AzureSource {
	if next.DisplayName == "" {
		next.DisplayName = existing.DisplayName
	}
	if next.StorageAccount == "" {
		next.StorageAccount = existing.StorageAccount
	}
	if next.Container == "" {
		next.Container = existing.Container
	}
	if next.Prefix == "" {
		next.Prefix = existing.Prefix
	}
	if next.ActionMode == "" {
		next.ActionMode = existing.ActionMode
	}
	if next.WritebackEnabled == "" {
		next.WritebackEnabled = existing.WritebackEnabled
	}
	if next.DeploymentMode == "" {
		next.DeploymentMode = existing.DeploymentMode
	}
	if next.LastUsedAt == "" {
		next.LastUsedAt = existing.LastUsedAt
	}
	return next
}

func displayName(source AzureSource) string {
	if source.StorageAccount != "" && source.Container != "" {
		return source.StorageAccount + " / " + source.Container
	}
	if source.Container != "" {
		return "Azure Blob / " + source.Container
	}
	return firstNonEmpty(source.DataStoreID, source.SourceID, "Azure Blob source")
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func clean(value string) string {
	return strings.TrimSpace(value)
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
