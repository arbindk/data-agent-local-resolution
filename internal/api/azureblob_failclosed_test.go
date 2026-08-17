package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/secretstore"
)

type recordingAzureBlobConnector struct {
	called bool
}

func (r *recordingAzureBlobConnector) NewScanner(azureblob.Config) (azureBlobScannerPort, error) {
	r.called = true
	return nil, errors.New("connector should not be constructed when approval is missing")
}

func TestAzureBlobActionExecuteBlocksMissingApprovalBeforeConnectorDispatch(t *testing.T) {
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/approvals" {
			t.Fatalf("unexpected platform path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"count":  0,
			"items":  []any{},
		})
	}))
	defer platform.Close()
	t.Setenv("EVEREST_PLATFORM_URL", platform.URL)

	connector := &recordingAzureBlobConnector{}
	server := &Server{
		cfg: config.Config{
			AzureBlobDataStoreID:        "finance-azureblob",
			AzureBlobContainer:          "finance",
			AzureActionMode:             execution.ActionModeApply,
			AzureWritebackEnabled:       "true",
			ActiveCompiledPolicyVersion: "2026.06.01",
			ActiveCompiledPolicyHash:    "sha256:policy",
			ActiveWorkflowVersion:       "workflow-v1",
			DefaultProcessingTier:       "tier_2",
			DefaultProcessingMode:       "standard",
		},
		orchestrator: execution.NewOrchestratorWithPorts(execution.NewManager(), execution.OrchestratorPorts{}),
		azureConfigStore: azureblob.NewRuntimeConfigStore(azureblob.Config{
			DataStoreID:      "finance-azureblob",
			ContainerName:    "finance",
			ActionMode:       execution.ActionModeApply,
			WritebackEnabled: "true",
		}),
		secrets:            secretstore.NewWithOptions(secretstore.Options{Provider: secretstore.ProviderEnvironment, EnvPrefix: "EVEREST_TEST_SECRET_"}),
		azureBlobConnector: connector,
	}

	payload := AzureBlobActionExecuteRequest{
		ExecutionID:      "exec-missing-grant",
		DecisionID:       "gd-approved",
		FileID:           "azureblob:finance:payroll.csv",
		Operation:        execution.ActionApplyBlobMetadata,
		ActionMode:       execution.ActionModeApply,
		WritebackEnabled: true,
		GuardianDecision: execution.GuardianDecisionAllow,
		Metadata: map[string]string{
			"dd_sensitivity": "confidential",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/connectors/azureblob/actions/execute", bytes.NewReader(body))
	req = req.WithContext(context.Background())
	rr := httptest.NewRecorder()

	server.azureBlobActionExecute(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp AzureBlobActionExecuteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "blocked" {
		t.Fatalf("response status = %q, want blocked", resp.Status)
	}
	if resp.ApprovalEnforcement != "approved_control_plane_approval_required" {
		t.Fatalf("approval enforcement = %q", resp.ApprovalEnforcement)
	}
	if connector.called {
		t.Fatalf("connector should not be constructed until approved HITL grant is verified")
	}
	if resp.ActionReceipt == nil || resp.ActionReceipt.Applied {
		t.Fatalf("expected blocked, unapplied action receipt, got %#v", resp.ActionReceipt)
	}
}
func TestAzureBlobActionExecuteBlocksUnauthorizedApplyRole(t *testing.T) {
	connector := &recordingAzureBlobConnector{}
	server := &Server{
		cfg: config.Config{
			AzureBlobDataStoreID:        "finance-azureblob",
			AzureBlobContainer:          "finance",
			AzureActionMode:             execution.ActionModeApply,
			AzureWritebackEnabled:       "true",
			ActiveCompiledPolicyVersion: "2026.06.01",
			ActiveCompiledPolicyHash:    "sha256:policy",
			ActiveWorkflowVersion:       "workflow-v1",
			DefaultProcessingTier:       "tier_2",
			DefaultProcessingMode:       "standard",
		},
		orchestrator: execution.NewOrchestratorWithPorts(execution.NewManager(), execution.OrchestratorPorts{}),
		azureConfigStore: azureblob.NewRuntimeConfigStore(azureblob.Config{
			DataStoreID:      "finance-azureblob",
			ContainerName:    "finance",
			ActionMode:       execution.ActionModeApply,
			WritebackEnabled: "true",
		}),
		secrets:            secretstore.NewWithOptions(secretstore.Options{Provider: secretstore.ProviderEnvironment, EnvPrefix: "EVEREST_TEST_SECRET_"}),
		azureBlobConnector: connector,
	}

	body, err := json.Marshal(AzureBlobActionExecuteRequest{
		ExecutionID:      "exec-unauthorized",
		DecisionID:       "gd-approved",
		FileID:           "azureblob:finance:payroll.csv",
		Operation:        execution.ActionApplyBlobMetadata,
		ActionMode:       execution.ActionModeApply,
		WritebackEnabled: true,
		GuardianDecision: execution.GuardianDecisionAllow,
		Metadata:         map[string]string{"dd_sensitivity": "confidential"},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/connectors/azureblob/actions/execute", bytes.NewReader(body))
	req.Header.Set("X-Everest-Role", "data_analyst")
	rr := httptest.NewRecorder()

	server.azureBlobActionExecute(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, body = %s", rr.Code, rr.Body.String())
	}
	if connector.called {
		t.Fatalf("connector should not be constructed for unauthorized apply role")
	}
}
