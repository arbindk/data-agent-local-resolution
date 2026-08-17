package platform

import (
	"path/filepath"
	"testing"
)

func TestApprovalLifecycle(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "platform.sqlite"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.db.Close()

	approval, err := store.CreateApproval(ApprovalCreateRequest{
		TenantID:      "tenant-local-001",
		RequestedBy:   "local-admin",
		Role:          "data_steward",
		Operation:     "migration",
		DataStoreID:   "azureblob-demo",
		ExecutionID:   "exec-001",
		SourceRegion:  "IN",
		TargetRegion:  "AE",
		ContractOwner: "owner@example.com",
		Reason:        "test migration approval",
	})
	if err != nil {
		t.Fatalf("CreateApproval() error = %v", err)
	}
	if approval.Status != ApprovalStatusPending {
		t.Fatalf("Status = %q, want %q", approval.Status, ApprovalStatusPending)
	}
	if approval.GovernanceDecision != "block" {
		t.Fatalf("GovernanceDecision = %q, want block", approval.GovernanceDecision)
	}

	updated, err := store.DecideApproval(approval.ApprovalID, ApprovalDecisionRequest{
		Actor:    "security@example.com",
		Role:     "security_officer",
		Decision: "approve",
		Comment:  "approved in test",
	})
	if err != nil {
		t.Fatalf("DecideApproval() error = %v", err)
	}
	if updated.Status != ApprovalStatusApproved {
		t.Fatalf("Status after decision = %q, want %q", updated.Status, ApprovalStatusApproved)
	}

	items, err := store.ListApprovals(ApprovalStatusApproved, "exec-001", 10)
	if err != nil {
		t.Fatalf("ListApprovals() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListApprovals() returned %d items, want 1", len(items))
	}
}
