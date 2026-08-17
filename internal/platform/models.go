package platform

import "time"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

type AgentRegistrationRequest struct {
	ActivationToken string   `json:"activation_token"`
	TenantID        string   `json:"tenant_id"`
	AgentID         string   `json:"agent_id"`
	Hostname        string   `json:"hostname"`
	Version         string   `json:"version"`
	OS              string   `json:"os"`
	Capabilities    []string `json:"capabilities"`
}

type Agent struct {
	AgentID       string    `json:"agent_id"`
	TenantID      string    `json:"tenant_id"`
	Hostname      string    `json:"hostname"`
	Version       string    `json:"version"`
	OS            string    `json:"os"`
	Capabilities  []string  `json:"capabilities"`
	Status        string    `json:"status"`
	RegisteredAt  time.Time `json:"registered_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

type AzureBlobContainerScope struct {
	Container string   `json:"container"`
	Prefixes  []string `json:"prefixes"`
}

type DataSourceCreateRequest struct {
	TenantID          string                    `json:"tenant_id"`
	DataStoreID       string                    `json:"data_store_id"`
	DisplayName       string                    `json:"display_name"`
	SourceSystem      string                    `json:"source_system"`
	StorageAccount    string                    `json:"storage_account"`
	Containers        []AzureBlobContainerScope `json:"containers"`
	CredentialRef     string                    `json:"credential_ref"`
	AssignedPolicyID  string                    `json:"assigned_policy_id"`
	ProcessingTier    string                    `json:"processing_tier"`
	ProcessingMode    string                    `json:"processing_mode"`
	ActionMode        string                    `json:"action_mode"`
	WritebackEnabled  bool                      `json:"writeback_enabled"`
	AllowedOperations []string                  `json:"allowed_operations"`
}

type DataSource struct {
	TenantID          string                    `json:"tenant_id"`
	DataStoreID       string                    `json:"data_store_id"`
	DisplayName       string                    `json:"display_name"`
	SourceSystem      string                    `json:"source_system"`
	StorageAccount    string                    `json:"storage_account"`
	Containers        []AzureBlobContainerScope `json:"containers"`
	CredentialRef     string                    `json:"credential_ref"`
	AssignedPolicyID  string                    `json:"assigned_policy_id"`
	ProcessingTier    string                    `json:"processing_tier"`
	ProcessingMode    string                    `json:"processing_mode"`
	ActionMode        string                    `json:"action_mode"`
	WritebackEnabled  bool                      `json:"writeback_enabled"`
	AllowedOperations []string                  `json:"allowed_operations"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type PolicyTemplate struct {
	PolicyID              string   `json:"policy_id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	CompiledPolicyVersion string   `json:"compiled_policy_version"`
	WorkflowVersion       string   `json:"workflow_version"`
	AllowedActions        []string `json:"allowed_actions"`
	DryRunOnlyActions     []string `json:"dry_run_only_actions"`
	GuardianRequired      bool     `json:"guardian_required"`
	DefaultPolicy         bool     `json:"default_policy"`
}

type JobCreateRequest struct {
	TenantID         string   `json:"tenant_id"`
	RequestedBy      string   `json:"requested_by"`
	AgentID          string   `json:"agent_id,omitempty"`
	DataStoreID      string   `json:"data_store_id"`
	PolicyID         string   `json:"policy_id"`
	Operation        string   `json:"operation"`
	Containers       []string `json:"containers,omitempty"`
	Prefix           string   `json:"prefix,omitempty"`
	ActionMode       string   `json:"action_mode"`
	WritebackEnabled bool     `json:"writeback_enabled"`
}

type Job struct {
	JobID       string    `json:"job_id"`
	TenantID    string    `json:"tenant_id"`
	RequestedBy string    `json:"requested_by"`
	AgentID     string    `json:"agent_id"`
	DataStoreID string    `json:"data_store_id"`
	PolicyID    string    `json:"policy_id"`
	Operation   string    `json:"operation"`
	Status      string    `json:"status"`
	LeaseIDs    []string  `json:"lease_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Lease struct {
	LeaseID               string                    `json:"lease_id"`
	JobID                 string                    `json:"job_id"`
	ExecutionID           string                    `json:"execution_id"`
	TenantID              string                    `json:"tenant_id"`
	AgentID               string                    `json:"agent_id"`
	DataStoreID           string                    `json:"data_store_id"`
	SourceSystem          string                    `json:"source_system"`
	StorageAccount        string                    `json:"storage_account"`
	Scope                 []AzureBlobContainerScope `json:"scope"`
	Operation             string                    `json:"operation"`
	CompiledPolicyVersion string                    `json:"compiled_policy_version"`
	WorkflowVersion       string                    `json:"workflow_version"`
	ProcessingTier        string                    `json:"processing_tier"`
	ProcessingMode        string                    `json:"processing_mode"`
	ActionMode            string                    `json:"action_mode"`
	WritebackEnabled      bool                      `json:"writeback_enabled"`
	Checkpoint            any                       `json:"checkpoint"`
	Status                string                    `json:"status"`
	CreatedAt             time.Time                 `json:"created_at"`
	ExpiresAt             time.Time                 `json:"expires_at"`
	LastHeartbeat         time.Time                 `json:"last_heartbeat,omitempty"`
}

type LeaseHeartbeatRequest struct {
	AgentID          string `json:"agent_id"`
	Status           string `json:"status"`
	RecordsProcessed int    `json:"records_processed,omitempty"`
	Message          string `json:"message,omitempty"`
}

type APIError struct {
	Error string `json:"error"`
}

type AzureBlobQuickstartRequest struct {
	TenantID         string                    `json:"tenant_id"`
	RequestedBy      string                    `json:"requested_by"`
	ActivationToken  string                    `json:"activation_token"`
	AgentID          string                    `json:"agent_id"`
	Hostname         string                    `json:"hostname"`
	AgentVersion     string                    `json:"agent_version"`
	AgentOS          string                    `json:"agent_os"`
	DataStoreID      string                    `json:"data_store_id"`
	DisplayName      string                    `json:"display_name"`
	StorageAccount   string                    `json:"storage_account"`
	Containers       []AzureBlobContainerScope `json:"containers"`
	CredentialRef    string                    `json:"credential_ref"`
	PolicyID         string                    `json:"policy_id"`
	ProcessingTier   string                    `json:"processing_tier"`
	ProcessingMode   string                    `json:"processing_mode"`
	ActionMode       string                    `json:"action_mode"`
	WritebackEnabled bool                      `json:"writeback_enabled"`
}

type StorageStatus struct {
	Status       string `json:"status"`
	DatabasePath string `json:"database_path"`
	DatabaseSize int64  `json:"database_size_bytes"`
	Agents       int    `json:"agents"`
	DataSources  int    `json:"data_sources"`
	Jobs         int    `json:"jobs"`
	Leases       int    `json:"leases"`
	EvidenceRows int    `json:"evidence_rows"`
	Approvals    int    `json:"approvals"`
}

type StorageResetRequest struct {
	Confirm string `json:"confirm"`
	Scope   string `json:"scope"`
	Reason  string `json:"reason,omitempty"`
}

type StorageBackupResponse struct {
	Status     string `json:"status"`
	BackupPath string `json:"backup_path"`
	SizeBytes  int64  `json:"size_bytes"`
}

type AuditEvent struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Data      any    `json:"data"`
	CreatedAt string `json:"created_at"`
}

type ApprovalCreateRequest struct {
	TenantID          string         `json:"tenant_id"`
	RequestedBy       string         `json:"requested_by"`
	Role              string         `json:"role"`
	Operation         string         `json:"operation"`
	ActionType        string         `json:"action_type,omitempty"`
	DataStoreID       string         `json:"data_store_id"`
	ExecutionID       string         `json:"execution_id,omitempty"`
	SourceRegion      string         `json:"source_region,omitempty"`
	TargetRegion      string         `json:"target_region,omitempty"`
	ContractOwner     string         `json:"contract_owner,omitempty"`
	PolicyIDs         []string       `json:"policy_ids,omitempty"`
	RequiredApprovers []string       `json:"required_approvers,omitempty"`
	Reason            string         `json:"reason"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type ApprovalDecisionRequest struct {
	Actor    string `json:"actor"`
	Role     string `json:"role"`
	Decision string `json:"decision"`
	Comment  string `json:"comment,omitempty"`
}

type ApprovalDecisionRecord struct {
	Actor     string    `json:"actor"`
	Role      string    `json:"role"`
	Decision  string    `json:"decision"`
	Comment   string    `json:"comment,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}

type ApprovalRequest struct {
	ApprovalID         string                   `json:"approval_id"`
	TenantID           string                   `json:"tenant_id"`
	RequestedBy        string                   `json:"requested_by"`
	Role               string                   `json:"role"`
	Operation          string                   `json:"operation"`
	ActionType         string                   `json:"action_type,omitempty"`
	DataStoreID        string                   `json:"data_store_id"`
	ExecutionID        string                   `json:"execution_id,omitempty"`
	SourceRegion       string                   `json:"source_region,omitempty"`
	TargetRegion       string                   `json:"target_region,omitempty"`
	ContractOwner      string                   `json:"contract_owner,omitempty"`
	PolicyIDs          []string                 `json:"policy_ids,omitempty"`
	RequiredApprovers  []string                 `json:"required_approvers"`
	Reason             string                   `json:"reason"`
	Status             string                   `json:"status"`
	GovernanceDecision string                   `json:"governance_decision,omitempty"`
	GovernanceReason   string                   `json:"governance_reason,omitempty"`
	Decisions          []ApprovalDecisionRecord `json:"decisions"`
	Metadata           map[string]any           `json:"metadata,omitempty"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	ExpiresAt          time.Time                `json:"expires_at"`
}
