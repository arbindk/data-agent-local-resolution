package controlplane

import "time"

type AzureBlobContainerScope struct {
	Container string   `json:"container"`
	Prefixes  []string `json:"prefixes"`
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

type NextLeaseResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Lease   Lease  `json:"lease,omitempty"`
}

type LeaseHeartbeatRequest struct {
	AgentID          string `json:"agent_id"`
	Status           string `json:"status"`
	RecordsProcessed int    `json:"records_processed,omitempty"`
	Message          string `json:"message,omitempty"`
}

type LeaseHeartbeatResponse struct {
	Status string `json:"status"`
	Lease  Lease  `json:"lease"`
}

type ApprovalRequest struct {
	ApprovalID  string `json:"approval_id"`
	Operation   string `json:"operation"`
	ActionType  string `json:"action_type,omitempty"`
	DataStoreID string `json:"data_store_id"`
	ExecutionID string `json:"execution_id,omitempty"`
	Status      string `json:"status"`
}

type ApprovalCreateRequest struct {
	TenantID      string         `json:"tenant_id"`
	RequestedBy   string         `json:"requested_by"`
	Role          string         `json:"role"`
	Operation     string         `json:"operation"`
	ActionType    string         `json:"action_type,omitempty"`
	DataStoreID   string         `json:"data_store_id"`
	ExecutionID   string         `json:"execution_id,omitempty"`
	SourceRegion  string         `json:"source_region,omitempty"`
	TargetRegion  string         `json:"target_region,omitempty"`
	ContractOwner string         `json:"contract_owner,omitempty"`
	PolicyIDs     []string       `json:"policy_ids,omitempty"`
	Reason        string         `json:"reason"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ApprovalCreateResponse struct {
	Status   string          `json:"status"`
	Approval ApprovalRequest `json:"approval"`
}

type ApprovalListResponse struct {
	Status string            `json:"status"`
	Count  int               `json:"count"`
	Items  []ApprovalRequest `json:"items"`
}
