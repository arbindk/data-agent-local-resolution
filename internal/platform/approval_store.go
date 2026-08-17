package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ApprovalStatusPending  = "PENDING"
	ApprovalStatusApproved = "APPROVED"
	ApprovalStatusRejected = "REJECTED"
	ApprovalStatusExpired  = "EXPIRED"
)

func (s *Store) CreateApproval(req ApprovalCreateRequest) (ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	if strings.TrimSpace(req.TenantID) == "" {
		req.TenantID = "tenant-local-001"
	}
	if strings.TrimSpace(req.RequestedBy) == "" {
		req.RequestedBy = "local-admin"
	}
	if strings.TrimSpace(req.Role) == "" {
		req.Role = "data_steward"
	}
	if strings.TrimSpace(req.Operation) == "" {
		req.Operation = strings.TrimSpace(req.ActionType)
	}
	if strings.TrimSpace(req.Operation) == "" {
		req.Operation = "apply_blob_metadata"
	}
	if strings.TrimSpace(req.DataStoreID) == "" {
		return ApprovalRequest{}, errors.New("data_store_id is required")
	}
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Customer requested governed action from Everest portal."
	}
	if len(req.RequiredApprovers) == 0 {
		req.RequiredApprovers = defaultApproversForOperation(req.Operation)
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}

	gov := s.EvaluateGovernance(GovernanceEvaluateRequest{
		TenantID:         req.TenantID,
		RequestedBy:      req.RequestedBy,
		Role:             req.Role,
		Operation:        req.Operation,
		SourceRegion:     req.SourceRegion,
		TargetRegion:     req.TargetRegion,
		ContractOwner:    req.ContractOwner,
		DataStoreID:      req.DataStoreID,
		PolicyIDs:        req.PolicyIDs,
		ActionMode:       "apply",
		WritebackEnabled: true,
	})

	if len(gov.RequiredApprovers) > 0 {
		req.RequiredApprovers = uniqueAppend(req.RequiredApprovers, gov.RequiredApprovers...)
	}

	approval := ApprovalRequest{
		ApprovalID:         "approval-" + now.Format("20060102-150405.000000000"),
		TenantID:           strings.TrimSpace(req.TenantID),
		RequestedBy:        strings.TrimSpace(req.RequestedBy),
		Role:               normalize(req.Role),
		Operation:          normalize(req.Operation),
		ActionType:         normalize(req.ActionType),
		DataStoreID:        strings.TrimSpace(req.DataStoreID),
		ExecutionID:        strings.TrimSpace(req.ExecutionID),
		SourceRegion:       strings.ToUpper(strings.TrimSpace(req.SourceRegion)),
		TargetRegion:       strings.ToUpper(strings.TrimSpace(req.TargetRegion)),
		ContractOwner:      strings.TrimSpace(req.ContractOwner),
		PolicyIDs:          req.PolicyIDs,
		RequiredApprovers:  uniqueAppend(nil, req.RequiredApprovers...),
		Reason:             strings.TrimSpace(req.Reason),
		Status:             ApprovalStatusPending,
		GovernanceDecision: gov.Decision,
		GovernanceReason:   gov.Reason,
		Decisions:          []ApprovalDecisionRecord{},
		Metadata:           req.Metadata,
		CreatedAt:          now,
		UpdatedAt:          now,
		ExpiresAt:          now.Add(72 * time.Hour),
	}

	if err := s.saveApprovalLocked(approval); err != nil {
		return ApprovalRequest{}, err
	}

	_ = s.writeAuditEventLocked("approval_requested", map[string]any{
		"approval_id":         approval.ApprovalID,
		"tenant_id":           approval.TenantID,
		"data_store_id":       approval.DataStoreID,
		"execution_id":        approval.ExecutionID,
		"operation":           approval.Operation,
		"requested_by":        approval.RequestedBy,
		"required_approvers":  approval.RequiredApprovers,
		"governance_decision": approval.GovernanceDecision,
	})

	return approval, nil
}

func (s *Store) ListApprovals(status string, executionID string, limit int) ([]ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	status = strings.ToUpper(strings.TrimSpace(status))
	executionID = strings.TrimSpace(executionID)

	rows, err := s.db.Query(`
		SELECT data
		FROM approvals
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ApprovalRequest{}

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}

		var item ApprovalRequest
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, err
		}

		item = expireApprovalIfNeeded(item, time.Now().UTC())
		if status != "" && item.Status != status {
			continue
		}
		if executionID != "" && item.ExecutionID != executionID {
			continue
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *Store) GetApproval(approvalID string) (ApprovalRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	approval, ok := s.getApprovalLocked(approvalID)
	if !ok {
		return ApprovalRequest{}, false
	}

	approval = expireApprovalIfNeeded(approval, time.Now().UTC())
	if approval.Status == ApprovalStatusExpired {
		_ = s.saveApprovalLocked(approval)
	}

	return approval, true
}

func (s *Store) DecideApproval(approvalID string, req ApprovalDecisionRequest) (ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	approval, ok := s.getApprovalLocked(approvalID)
	if !ok {
		return ApprovalRequest{}, fmt.Errorf("approval not found: %s", approvalID)
	}

	now := time.Now().UTC()
	approval = expireApprovalIfNeeded(approval, now)
	if approval.Status != ApprovalStatusPending {
		return ApprovalRequest{}, fmt.Errorf("approval is already %s", approval.Status)
	}

	decision := normalize(req.Decision)
	if decision == "" {
		return ApprovalRequest{}, errors.New("decision is required")
	}
	if decision != "approve" && decision != "approved" && decision != "reject" && decision != "rejected" {
		return ApprovalRequest{}, errors.New("decision must be approve or reject")
	}

	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "local-admin"
	}
	role := normalize(req.Role)
	if role == "" {
		role = "security_officer"
	}

	record := ApprovalDecisionRecord{
		Actor:     actor,
		Role:      role,
		Decision:  decision,
		Comment:   strings.TrimSpace(req.Comment),
		DecidedAt: now,
	}

	approval.Decisions = append(approval.Decisions, record)
	approval.UpdatedAt = now
	if decision == "approve" || decision == "approved" {
		approval.Status = ApprovalStatusApproved
	} else {
		approval.Status = ApprovalStatusRejected
	}

	if err := s.saveApprovalLocked(approval); err != nil {
		return ApprovalRequest{}, err
	}

	_ = s.writeAuditEventLocked("approval_decided", map[string]any{
		"approval_id":   approval.ApprovalID,
		"status":        approval.Status,
		"actor":         actor,
		"role":          role,
		"operation":     approval.Operation,
		"data_store_id": approval.DataStoreID,
		"execution_id":  approval.ExecutionID,
	})

	return approval, nil
}

func (s *Store) ApprovalCounts() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.approvalCountsLocked()
}

func (s *Store) approvalCountsLocked() map[string]int {

	counts := map[string]int{
		"pending":  0,
		"approved": 0,
		"rejected": 0,
		"expired":  0,
		"total":    0,
	}

	rows, err := s.db.Query(`SELECT data FROM approvals`)
	if err != nil {
		return counts
	}
	defer rows.Close()

	now := time.Now().UTC()
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		var approval ApprovalRequest
		if err := json.Unmarshal([]byte(data), &approval); err != nil {
			continue
		}

		approval = expireApprovalIfNeeded(approval, now)
		counts["total"]++
		switch approval.Status {
		case ApprovalStatusApproved:
			counts["approved"]++
		case ApprovalStatusRejected:
			counts["rejected"]++
		case ApprovalStatusExpired:
			counts["expired"]++
		default:
			counts["pending"]++
		}
	}

	return counts
}

func (s *Store) getApprovalLocked(approvalID string) (ApprovalRequest, bool) {
	approvalID = strings.TrimSpace(approvalID)
	if approvalID == "" {
		return ApprovalRequest{}, false
	}

	var data string
	err := s.db.QueryRow(`SELECT data FROM approvals WHERE approval_id = ?`, approvalID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return ApprovalRequest{}, false
		}
		return ApprovalRequest{}, false
	}

	var approval ApprovalRequest
	if err := json.Unmarshal([]byte(data), &approval); err != nil {
		return ApprovalRequest{}, false
	}

	return approval, true
}

func (s *Store) saveApprovalLocked(approval ApprovalRequest) error {
	data, err := json.Marshal(approval)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO approvals (
			approval_id,
			status,
			tenant_id,
			data_store_id,
			execution_id,
			operation,
			data,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(approval_id) DO UPDATE SET
			status = excluded.status,
			tenant_id = excluded.tenant_id,
			data_store_id = excluded.data_store_id,
			execution_id = excluded.execution_id,
			operation = excluded.operation,
			data = excluded.data,
			updated_at = excluded.updated_at
	`,
		approval.ApprovalID,
		approval.Status,
		approval.TenantID,
		approval.DataStoreID,
		approval.ExecutionID,
		approval.Operation,
		string(data),
		approval.UpdatedAt.Format(time.RFC3339Nano),
	)

	return err
}

func expireApprovalIfNeeded(approval ApprovalRequest, now time.Time) ApprovalRequest {
	if approval.Status == ApprovalStatusPending && !approval.ExpiresAt.IsZero() && now.After(approval.ExpiresAt) {
		approval.Status = ApprovalStatusExpired
		approval.UpdatedAt = now
	}
	return approval
}

func defaultApproversForOperation(operation string) []string {
	operation = normalize(operation)

	switch {
	case operation == "apply_blob_metadata" || operation == "apply_blob_index_tags":
		return []string{"data_steward", "security_officer"}
	case operation == "migration":
		return []string{"contract_owner", "security_officer"}
	case isDestructiveOperation(operation):
		return []string{"contract_owner", "security_officer"}
	default:
		return []string{"security_officer"}
	}
}
