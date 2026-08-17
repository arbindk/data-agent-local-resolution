package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

func TestBuildActionRequestEnvelopeDefaultsToDACContract(t *testing.T) {
	req := ExecutionStartRequest{
		ExecutionID: "exec-001",
		Lease: ExecutionLease{
			LeaseID:               "lease-001",
			JobID:                 "job-001",
			DataStoreID:           "finance-azureblob",
			CompiledPolicyVersion: "2026.06.01",
			CompiledArtifactHash:  "sha256:policy",
			WorkflowVersion:       "workflow-v1",
		},
		ExecutionState: ExecutionState{
			DataStoreID:     "finance-azureblob",
			ProcessingTier:  "tier_2",
			ProcessingMode:  "standard",
			PolicyVersion:   "2026.06.01",
			WorkflowVersion: "workflow-v1",
			Settings: ExecutionSettings{
				EnableContentIntelligence: true,
				EnableGuardianEvaluation:  true,
			},
			OperationalConfig: ExecutionOperationalConfig{
				ActionMode:                   ActionModeDryRun,
				LocalExecutionRequired:       true,
				GuardianBeforeActionRequired: true,
			},
		},
		BatchID: "batch-001",
	}

	envelope := buildActionRequestEnvelope(req, time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))

	if err := envelope.Validate(); err != nil {
		t.Fatalf("expected generated envelope to be valid, got %v", err)
	}
	if envelope.ConnectorID != "azure_blob" {
		t.Fatalf("expected azure_blob connector inference, got %s", envelope.ConnectorID)
	}
	if envelope.Operation != "quick_scan" {
		t.Fatalf("expected quick_scan operation, got %s", envelope.Operation)
	}
	if !envelope.RuntimeContext.ZeroCopyRequired {
		t.Fatalf("expected zero-copy requirement to be enabled")
	}
	if envelope.PolicyContext.PolicyBundleHash != "sha256:policy" {
		t.Fatalf("expected policy hash to be attached, got %s", envelope.PolicyContext.PolicyBundleHash)
	}
}

func TestBuildActionReceiptsCapturesGuardianGate(t *testing.T) {
	req := ExecutionStartRequest{
		ExecutionID: "exec-002",
		ExecutionState: ExecutionState{
			Settings: ExecutionSettings{
				EnableWriteBackActions: true,
			},
			OperationalConfig: ExecutionOperationalConfig{
				ActionMode: ActionModeApply,
			},
		},
	}
	envelope := contracts.ActionRequestEnvelope{
		RequestID:   "req-002",
		ConnectorID: "azure_blob",
		RuntimeContext: contracts.RuntimeContext{
			ActionMode: ActionModeApply,
		},
	}
	decision := &contracts.GuardianDecision{
		DecisionID: "gd-002",
		Approved:   true,
		Obligations: []contracts.GuardianObligation{
			{ID: contracts.ObligationRecordAudit, Type: "audit"},
			{ID: contracts.ObligationAttachPolicyHash, Type: "policy"},
		},
	}
	batch := &contracts.NormalizedBatch{
		BatchID:     "batch-002",
		DataStoreID: "finance-azureblob",
		Records: []contracts.NormalizedFileRecord{
			{FileID: "azureblob:finance/payroll.csv", DataStoreID: "finance-azureblob", Path: "payroll.csv", PermissionRiskScore: 92},
		},
	}

	receipts := buildActionReceipts(req, envelope, decision, batch, TierBehavior{WriteBackPlanned: true})

	if len(receipts) != 1 {
		t.Fatalf("expected one planned receipt, got %d", len(receipts))
	}
	receipt := receipts[0]
	if receipt.DecisionID != "gd-002" {
		t.Fatalf("expected decision id gd-002, got %s", receipt.DecisionID)
	}
	if receipt.DryRun {
		t.Fatalf("expected apply mode receipt to be non-dry-run")
	}
	if receipt.Applied {
		t.Fatalf("expected orchestrator receipt to remain planned until Action Executor completes")
	}
	if receipt.Status != contracts.ActionStatusPlanned {
		t.Fatalf("expected planned status, got %s", receipt.Status)
	}
	if !strings.HasPrefix(receipt.ReceiptHash, "sha256:") {
		t.Fatalf("expected receipt hash, got %s", receipt.ReceiptHash)
	}
	if len(receipt.ObjectRefs) != 1 {
		t.Fatalf("expected sampled object ref")
	}
}

func TestEvaluateIntentProducesGuardianDecisionAndReceipt(t *testing.T) {
	orchestrator := NewOrchestrator(nil, nil, nil, nil)
	envelope := contracts.ActionRequestEnvelope{
		RequestID:   "req-workbench-001",
		TenantID:    "tenant-a",
		Actor:       contracts.ActorContext{ID: "owner@example.com", Role: "data_steward"},
		SourceID:    "finance-azureblob",
		ConnectorID: "azure_blob",
		Operation:   "set_metadata",
		Scope: contracts.DataScope{
			SourceID:     "finance-azureblob",
			Container:    "finance",
			ZeroCopyMode: "reference_stream_range_or_bounded_preview",
			ObjectRefs: []contracts.ObjectRef{
				{URI: "azureblob:finance:payroll.csv", ConnectorID: "azure_blob", SourceID: "finance-azureblob", Container: "finance", Path: "payroll.csv"},
			},
		},
		PolicyContext: contracts.PolicyContextRef{
			PolicyID:            "finance-policy",
			PolicyBundleVersion: "2026.06.01",
			PolicyBundleHash:    "sha256:policy",
		},
		RuntimeContext: contracts.RuntimeContext{
			ActionMode:             ActionModeApply,
			LocalExecutionRequired: true,
			ZeroCopyRequired:       true,
		},
		CorrelationID: "corr-workbench-001",
	}

	result, err := orchestrator.EvaluateIntent(context.Background(), envelope)
	if err != nil {
		t.Fatalf("expected evaluate intent to succeed, got %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %s", result.Status)
	}
	if result.GuardianDecision == nil || result.GuardianDecision.Decision != contracts.DecisionAllow {
		t.Fatalf("expected allow Guardian decision, got %#v", result.GuardianDecision)
	}
	if result.ActionReceipt == nil {
		t.Fatalf("expected mutation intent to include action receipt")
	}
	if result.ActionReceipt.DecisionID != result.GuardianDecision.DecisionID {
		t.Fatalf("expected receipt to reference Guardian decision")
	}
	if len(result.ComponentTrace) < 3 {
		t.Fatalf("expected component trace to describe orchestrator path")
	}
}
