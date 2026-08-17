package execution

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type failingSearchIndexPort struct {
	called bool
}

func (f *failingSearchIndexPort) Publish(context.Context, contracts.SearchIndexBatch) (*contracts.SearchIndexReceipt, error) {
	f.called = true
	return &contracts.SearchIndexReceipt{
		Status:          contracts.SearchIndexStatusQueued,
		Mode:            contracts.SearchIndexModeElastic,
		Backend:         "test_elastic",
		DocumentsQueued: 1,
		PublishedAt:     time.Now().UTC(),
	}, errors.New("elastic unavailable")
}

func TestStartDegradesWhenSearchIndexPublishFails(t *testing.T) {
	search := &failingSearchIndexPort{}
	req := testExecutionRequest()
	req.ExecutionState.Settings.EnableElasticsearchIndex = true

	orchestrator := NewOrchestratorWithPorts(NewManager(), OrchestratorPorts{
		Policy:   testPolicyPort(),
		Workflow: &fakeWorkflowPort{workflow: testWorkflow()},
		Work:     testWorkStore(),
		Guardian: &fakeGuardianPort{decision: testGuardianDecision(contracts.DecisionAllow, true, false)},
		Search:   search,
	})

	result, err := orchestrator.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.Status.Status != "COMPLETED" {
		t.Fatalf("status = %q, want COMPLETED", result.Status.Status)
	}
	if !search.called {
		t.Fatalf("expected SearchIndexPort to be called")
	}
	if !result.Status.Degraded {
		t.Fatalf("expected degraded status when index publish fails")
	}
	if !strings.Contains(result.Status.Reason, "search index publish queued") {
		t.Fatalf("reason = %q, want queued index failure", result.Status.Reason)
	}
}
