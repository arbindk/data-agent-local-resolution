package execution

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

type fakePolicyPort struct {
	result *policyresolver.ResolveResult
	err    error
	called bool
}

func (f *fakePolicyPort) ResolveAndActivatePolicy(context.Context, policyresolver.PolicyLease) (*policyresolver.ResolveResult, error) {
	f.called = true
	return f.result, f.err
}

type fakeWorkflowPort struct {
	workflow *WorkflowDefinition
	err      error
	called   bool
}

func (f *fakeWorkflowPort) Resolve(string) (*WorkflowDefinition, error) {
	f.called = true
	return f.workflow, f.err
}

type fakeWorkStorePort struct {
	batch *contracts.NormalizedBatch
	err   error
	saved bool
}

func (f *fakeWorkStorePort) LoadNormalizedBatch(context.Context, string) (*contracts.NormalizedBatch, error) {
	return f.batch, f.err
}

func (f *fakeWorkStorePort) SaveManifest(context.Context, contracts.ManifestSummary, []contracts.ManifestDetail) error {
	f.saved = true
	return nil
}

type fakeGuardianPort struct {
	decision *contracts.GuardianDecision
	err      error
	called   bool
}

func (f *fakeGuardianPort) Decide(context.Context, GuardianEvaluation) (*contracts.GuardianDecision, error) {
	f.called = true
	return f.decision, f.err
}

func TestStartFailsClosedWhenGuardianUnavailable(t *testing.T) {
	guardian := &fakeGuardianPort{err: errors.New("guardian unavailable")}
	orchestrator := testOrchestratorWithGuardian(guardian)

	result, err := orchestrator.Start(context.Background(), testExecutionRequest())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.Status.Status != "FAILED" {
		t.Fatalf("status = %q, want FAILED", result.Status.Status)
	}
	if result.Status.Phase != "GUARDIAN_FAILED" {
		t.Fatalf("phase = %q, want GUARDIAN_FAILED", result.Status.Phase)
	}
	if !strings.Contains(result.Status.Reason, "guardian unavailable") {
		t.Fatalf("reason = %q, want guardian error", result.Status.Reason)
	}
	if len(result.ActionReceipts) != 0 {
		t.Fatalf("expected no action receipts when Guardian is unavailable")
	}
	if !guardian.called {
		t.Fatalf("expected Guardian port to be called")
	}
}

func TestStartFailsClosedWhenPolicyIsNotExecutionReady(t *testing.T) {
	policy := &fakePolicyPort{result: &policyresolver.ResolveResult{
		Status:              "failed_closed",
		DataStoreID:         "finance-azureblob",
		LocalExecutionReady: false,
		Reason:              "stale_policy_artifact",
	}}
	workflow := &fakeWorkflowPort{workflow: testWorkflow()}
	guardian := &fakeGuardianPort{decision: testGuardianDecision(contracts.DecisionAllow, true, false)}
	orchestrator := NewOrchestratorWithPorts(NewManager(), OrchestratorPorts{
		Policy:   policy,
		Workflow: workflow,
		Work:     testWorkStore(),
		Guardian: guardian,
	})

	result, err := orchestrator.Start(context.Background(), testExecutionRequest())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.Status.Status != "FAILED" {
		t.Fatalf("status = %q, want FAILED", result.Status.Status)
	}
	if result.Status.Phase != "POLICY_FAILED" {
		t.Fatalf("phase = %q, want POLICY_FAILED", result.Status.Phase)
	}
	if !strings.Contains(result.Status.Reason, "stale_policy_artifact") {
		t.Fatalf("reason = %q, want stale policy reason", result.Status.Reason)
	}
	if workflow.called {
		t.Fatalf("workflow port should not be called after stale policy")
	}
	if guardian.called {
		t.Fatalf("guardian port should not be called after stale policy")
	}
}

func TestStartBlocksReceiptsForGuardianDenyAndHITLRequired(t *testing.T) {
	tests := []struct {
		name          string
		decision      *contracts.GuardianDecision
		wantApproved  bool
		wantReasonSub string
	}{
		{
			name:          "deny",
			decision:      testGuardianDecision(contracts.DecisionDeny, false, false),
			wantApproved:  false,
			wantReasonSub: "did not return allow",
		},
		{
			name:          "hitl required",
			decision:      testGuardianDecision(contracts.DecisionAllow, true, true),
			wantApproved:  true,
			wantReasonSub: "human approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := testOrchestratorWithGuardian(&fakeGuardianPort{decision: tt.decision})

			result, err := orchestrator.Start(context.Background(), testExecutionRequest())
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			if result.Status.Status != "COMPLETED" {
				t.Fatalf("status = %q, want COMPLETED with blocked receipt evidence", result.Status.Status)
			}
			if result.Status.GuardianApproved != tt.wantApproved {
				t.Fatalf("guardian approved = %v, want %v", result.Status.GuardianApproved, tt.wantApproved)
			}
			if len(result.ActionReceipts) != 1 {
				t.Fatalf("action receipts = %d, want 1", len(result.ActionReceipts))
			}
			receipt := result.ActionReceipts[0]
			if receipt.Status != contracts.ActionStatusBlocked {
				t.Fatalf("receipt status = %q, want blocked", receipt.Status)
			}
			if receipt.Applied {
				t.Fatalf("blocked receipt must not be applied")
			}
			if !strings.Contains(receipt.ResultSummary, tt.wantReasonSub) {
				t.Fatalf("receipt summary = %q, want substring %q", receipt.ResultSummary, tt.wantReasonSub)
			}
		})
	}
}

func TestEvaluateIntentFailsClosedWhenGuardianUnavailable(t *testing.T) {
	orchestrator := NewOrchestratorWithPorts(NewManager(), OrchestratorPorts{
		Guardian: &fakeGuardianPort{err: errors.New("guardian offline")},
	})

	result, err := orchestrator.EvaluateIntent(context.Background(), testIntentEnvelope())
	if err != nil {
		t.Fatalf("EvaluateIntent() error = %v", err)
	}
	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if result.GuardianDecision != nil {
		t.Fatalf("expected no Guardian decision when port is unavailable")
	}
	if len(result.ComponentTrace) == 0 || result.ComponentTrace[len(result.ComponentTrace)-1].Status != "failed" {
		t.Fatalf("expected failed Guardian trace, got %#v", result.ComponentTrace)
	}
}

func testOrchestratorWithGuardian(guardian GuardianPort) *Orchestrator {
	return NewOrchestratorWithPorts(NewManager(), OrchestratorPorts{
		Policy:   testPolicyPort(),
		Workflow: &fakeWorkflowPort{workflow: testWorkflow()},
		Work:     testWorkStore(),
		Guardian: guardian,
	})
}

func testPolicyPort() *fakePolicyPort {
	return &fakePolicyPort{result: &policyresolver.ResolveResult{
		Status:              "activated",
		DataStoreID:         "finance-azureblob",
		LocalExecutionReady: true,
		PolicyContext:       testPolicyContext(),
	}}
}

func testPolicyContext() *policyresolver.PolicyContext {
	return &policyresolver.PolicyContext{
		DataStoreID:             "finance-azureblob",
		CompiledPolicyVersion:   "2026.06.01",
		SourcePolicyID:          "finance-policy",
		SourcePolicyVersion:     "2026.06.01",
		ArtifactHash:            "sha256:policy",
		SignatureVerified:       true,
		ArtifactHashVerified:    true,
		LocalExecutionConfirmed: true,
		ActivatedAt:             time.Now().UTC(),
	}
}

func testWorkflow() *WorkflowDefinition {
	return &WorkflowDefinition{
		WorkflowID:      "workflow-mvs",
		WorkflowVersion: "workflow-v1",
		Steps: []WorkflowStep{
			{Name: "metadata_scan", Owner: "agent-core", Required: true},
		},
	}
}

func testWorkStore() *fakeWorkStorePort {
	return &fakeWorkStorePort{batch: &contracts.NormalizedBatch{
		BatchID:     "batch-ports",
		DataStoreID: "finance-azureblob",
		Records: []contracts.NormalizedFileRecord{
			{
				FileID:              "azureblob:finance:payroll.csv",
				DataStoreID:         "finance-azureblob",
				Path:                "payroll.csv",
				Name:                "payroll.csv",
				ContentType:         "text/csv",
				PermissionRiskScore: 92,
				ScanTimestamp:       time.Now().UTC(),
			},
		},
	}}
}

func testExecutionRequest() ExecutionStartRequest {
	return ExecutionStartRequest{
		ExecutionID: "exec-ports",
		Lease: ExecutionLease{
			LeaseID:               "lease-ports",
			JobID:                 "job-ports",
			DataStoreID:           "finance-azureblob",
			CompiledPolicyVersion: "2026.06.01",
			CompiledArtifactHash:  "sha256:policy",
			WorkflowVersion:       "workflow-v1",
		},
		ExecutionState: ExecutionState{
			DataStoreID:    "finance-azureblob",
			ProcessingTier: "tier_2",
			ProcessingMode: "standard",
			Settings: ExecutionSettings{
				EnableContentIntelligence: true,
				EnableGuardianEvaluation:  true,
				EnableWriteBackActions:    true,
			},
			OperationalConfig: ExecutionOperationalConfig{
				LocalExecutionRequired:       true,
				GuardianBeforeActionRequired: true,
				ActionMode:                   ActionModeApply,
			},
		},
		BatchID: "batch-ports",
	}
}

func testGuardianDecision(decision string, approved bool, hitl bool) *contracts.GuardianDecision {
	return &contracts.GuardianDecision{
		DecisionID:              "gd-test",
		DecisionRequestID:       "gdr-test",
		RequestID:               "req-test",
		Decision:                decision,
		Approved:                approved,
		PolicyBundleVersion:     "2026.06.01",
		PolicyBundleHash:        "sha256:policy",
		RulesApplied:            []string{"test_rule"},
		Obligations:             []contracts.GuardianObligation{{ID: contracts.ObligationRecordAudit, Type: "audit"}},
		HITLRequired:            hitl,
		HumanReviewRequired:     hitl,
		Reasoning:               "test decision",
		EvaluatedAt:             time.Now().UTC(),
		LocalExecutionConfirmed: true,
	}
}

func testIntentEnvelope() contracts.ActionRequestEnvelope {
	return contracts.ActionRequestEnvelope{
		RequestID:   "req-intent",
		TenantID:    "tenant-a",
		Actor:       contracts.ActorContext{ID: "owner@example.com", Role: "data_steward"},
		SourceID:    "finance-azureblob",
		ConnectorID: "azure_blob",
		Operation:   ActionApplyBlobMetadata,
		Scope: contracts.DataScope{
			SourceID:     "finance-azureblob",
			Container:    "finance",
			ZeroCopyMode: "reference_stream_range_or_bounded_preview",
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
		CorrelationID: "corr-intent",
	}
}
