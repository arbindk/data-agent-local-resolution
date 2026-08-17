package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/connectors/catalog"
)

const (
	OperationHealth        = "health"
	OperationDiscover      = "discover_sources"
	OperationScanMetadata  = "scan_metadata"
	OperationPreviewAction = "preview_action"
	OperationExecuteAction = "execute_action"
)

type Request struct {
	RuntimeRequestID string         `json:"runtime_request_id,omitempty"`
	ConnectorID      string         `json:"connector_id"`
	Operation        string         `json:"operation"`
	ExecutionID      string         `json:"execution_id,omitempty"`
	DataStoreID      string         `json:"data_store_id,omitempty"`
	SourceRegion     string         `json:"source_region,omitempty"`
	TargetRegion     string         `json:"target_region,omitempty"`
	ActionMode       string         `json:"action_mode,omitempty"`
	PreviewOnly      bool           `json:"preview_only,omitempty"`
	GuardianDecision string         `json:"guardian_decision,omitempty"`
	HITLApproved     bool           `json:"hitl_approved,omitempty"`
	Confirmation     string         `json:"confirmation,omitempty"`
	RuntimeEndpoint  string         `json:"runtime_endpoint,omitempty"`
	TimeoutSeconds   int            `json:"timeout_seconds,omitempty"`
	Scope            map[string]any `json:"scope,omitempty"`
	Object           map[string]any `json:"object,omitempty"`
	Action           map[string]any `json:"action,omitempty"`
	Config           map[string]any `json:"config,omitempty"`
}

type Response struct {
	Status           string         `json:"status"`
	ConnectorID      string         `json:"connector_id"`
	Operation        string         `json:"operation"`
	RuntimeKind      string         `json:"runtime_kind"`
	RuntimeProtocol  string         `json:"runtime_protocol,omitempty"`
	InvocationID     string         `json:"invocation_id"`
	SafeMode         bool           `json:"safe_mode"`
	DryRun           bool           `json:"dry_run"`
	Applied          bool           `json:"applied"`
	Records          []any          `json:"records,omitempty"`
	NormalizedBatch  map[string]any `json:"normalized_batch,omitempty"`
	ActionResult     map[string]any `json:"action_result,omitempty"`
	Checks           []Check        `json:"checks,omitempty"`
	Raw              any            `json:"raw,omitempty"`
	StartedAt        string         `json:"started_at"`
	CompletedAt      string         `json:"completed_at"`
	DurationMillis   int64          `json:"duration_ms"`
	Error            string         `json:"error,omitempty"`
	ContractVersion  string         `json:"contract_version"`
	ExecutionControl []string       `json:"execution_control"`
}

type Check struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	Detail string `json:"detail"`
}

func Invoke(ctx context.Context, manifest catalog.ConnectorManifest, req Request) (Response, error) {
	start := time.Now().UTC()
	req = withDefaults(manifest, req)

	resp := Response{
		Status:          "pending",
		ConnectorID:     manifest.ConnectorID,
		Operation:       req.Operation,
		RuntimeKind:     manifest.Runtime.Kind,
		RuntimeProtocol: manifest.Runtime.Protocol,
		InvocationID:    req.RuntimeRequestID,
		SafeMode:        true,
		DryRun:          req.PreviewOnly || req.ActionMode != "apply" || req.Operation == OperationPreviewAction,
		StartedAt:       start.Format(time.RFC3339),
		ContractVersion: "connector_runtime_v1",
		ExecutionControl: []string{
			"guardian_before_apply",
			"hitl_before_restricted_actions",
			"dry_run_default",
			"connector_manifest_validated",
		},
	}

	if err := validateOperation(manifest, req.Operation); err != nil {
		return complete(resp, start, "blocked", err.Error()), nil
	}

	if reason := blockedApplyReason(manifest, req); reason != "" {
		resp.Checks = append(resp.Checks, Check{Name: "everest_safety_gate", Result: "blocked", Detail: reason})
		return complete(resp, start, "blocked", reason), nil
	}

	switch manifest.Runtime.Kind {
	case "builtin":
		resp.Checks = append(resp.Checks, Check{
			Name:   "builtin_runtime",
			Result: "native_endpoint_required",
			Detail: "This connector is implemented by first-party Everest handlers. Use the connector-specific API for execution.",
		})
		return complete(resp, start, "native_runtime", ""), nil

	case "http":
		return invokeHTTP(ctx, manifest, req, resp, start)

	case "external_process":
		return invokeExternalProcess(ctx, manifest, req, resp, start)

	case "external_process_or_http":
		if strings.TrimSpace(req.RuntimeEndpoint) != "" {
			return invokeHTTP(ctx, manifest, req, resp, start)
		}
		return invokeExternalProcess(ctx, manifest, req, resp, start)

	default:
		reason := "unsupported connector runtime kind: " + manifest.Runtime.Kind
		resp.Checks = append(resp.Checks, Check{Name: "runtime_kind", Result: "blocked", Detail: reason})
		return complete(resp, start, "blocked", reason), nil
	}
}

func withDefaults(manifest catalog.ConnectorManifest, req Request) Request {
	if strings.TrimSpace(req.ConnectorID) == "" {
		req.ConnectorID = manifest.ConnectorID
	}
	if strings.TrimSpace(req.Operation) == "" {
		req.Operation = OperationHealth
	}
	req.Operation = strings.ToLower(strings.TrimSpace(req.Operation))
	if strings.TrimSpace(req.ActionMode) == "" {
		req.ActionMode = "dry_run"
	}
	req.ActionMode = strings.ToLower(strings.TrimSpace(req.ActionMode))
	if strings.TrimSpace(req.GuardianDecision) == "" {
		req.GuardianDecision = "pending"
	}
	req.GuardianDecision = strings.ToLower(strings.TrimSpace(req.GuardianDecision))
	if strings.TrimSpace(req.RuntimeRequestID) == "" {
		req.RuntimeRequestID = "connector-runtime-" + time.Now().UTC().Format("20060102-150405.000000000")
	}
	if req.TimeoutSeconds <= 0 || req.TimeoutSeconds > 120 {
		req.TimeoutSeconds = 30
	}
	return req
}

func validateOperation(manifest catalog.ConnectorManifest, operation string) error {
	switch operation {
	case OperationHealth, OperationDiscover, OperationScanMetadata, OperationPreviewAction, OperationExecuteAction:
		return nil
	default:
		return fmt.Errorf("unsupported connector runtime operation: %s", operation)
	}
}

func blockedApplyReason(manifest catalog.ConnectorManifest, req Request) string {
	if req.Operation != OperationExecuteAction || req.PreviewOnly || req.ActionMode != "apply" {
		return ""
	}

	actionID := requestedActionID(req)
	action, ok := findAction(manifest, actionID)
	if !ok && actionID != "" {
		return "connector action is not declared in manifest: " + actionID
	}
	if !ok {
		return "execute_action requires action.action_id so Everest can enforce manifest safety controls"
	}

	if action.RequiresGuardian && req.GuardianDecision != "allow" {
		return "Guardian allow is required before connector action apply"
	}
	if action.RequiresHITL && !req.HITLApproved {
		return "Approved HITL request is required before connector action apply"
	}
	if action.Confirmation != "" && strings.ToUpper(strings.TrimSpace(req.Confirmation)) != strings.ToUpper(action.Confirmation) {
		return "Connector action requires confirmation text " + action.Confirmation
	}

	return ""
}

func requestedActionID(req Request) string {
	for _, key := range []string{"action_id", "operation", "action_type"} {
		if v, ok := req.Action[key]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
				return s
			}
		}
		if v, ok := req.Config[key]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func findAction(manifest catalog.ConnectorManifest, actionID string) (catalog.ConnectorAction, bool) {
	actionID = strings.TrimSpace(actionID)
	for _, action := range manifest.Actions {
		if action.ActionID == actionID {
			return action, true
		}
	}
	return catalog.ConnectorAction{}, false
}

func invokeHTTP(ctx context.Context, manifest catalog.ConnectorManifest, req Request, resp Response, start time.Time) (Response, error) {
	endpoint := runtimeEndpoint(manifest, req)
	if endpoint == "" {
		reason := "runtime_endpoint is required for HTTP connector runtime invocation"
		resp.Checks = append(resp.Checks, Check{Name: "runtime_endpoint", Result: "blocked", Detail: reason})
		return complete(resp, start, "needs_configuration", reason), nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return complete(resp, start, "failed", err.Error()), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return complete(resp, start, "failed", err.Error()), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Everest-Contract", "connector_runtime_v1")

	client := &http.Client{Timeout: time.Duration(req.TimeoutSeconds) * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Checks = append(resp.Checks, Check{Name: "http_runtime", Result: "failed", Detail: err.Error()})
		return complete(resp, start, "failed", err.Error()), nil
	}
	defer httpResp.Body.Close()

	var runtimeResp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&runtimeResp); err != nil {
		reason := "HTTP connector runtime did not return connector_runtime_v1 JSON: " + err.Error()
		resp.Checks = append(resp.Checks, Check{Name: "http_runtime", Result: "failed", Detail: reason})
		return complete(resp, start, "failed", reason), nil
	}

	runtimeResp.ConnectorID = firstNonEmpty(runtimeResp.ConnectorID, resp.ConnectorID)
	runtimeResp.Operation = firstNonEmpty(runtimeResp.Operation, resp.Operation)
	runtimeResp.RuntimeKind = firstNonEmpty(runtimeResp.RuntimeKind, resp.RuntimeKind)
	runtimeResp.RuntimeProtocol = firstNonEmpty(runtimeResp.RuntimeProtocol, resp.RuntimeProtocol)
	runtimeResp.InvocationID = firstNonEmpty(runtimeResp.InvocationID, resp.InvocationID)
	runtimeResp.ContractVersion = firstNonEmpty(runtimeResp.ContractVersion, "connector_runtime_v1")
	runtimeResp.SafeMode = true
	runtimeResp.DryRun = resp.DryRun
	if httpResp.StatusCode >= 400 && runtimeResp.Status == "" {
		runtimeResp.Status = "failed"
	}
	if runtimeResp.Status == "" {
		runtimeResp.Status = "ok"
	}
	return finish(runtimeResp, start), nil
}

func invokeExternalProcess(ctx context.Context, manifest catalog.ConnectorManifest, req Request, resp Response, start time.Time) (Response, error) {
	entrypoint := strings.TrimSpace(manifest.Runtime.Entrypoint)
	if entrypoint == "" {
		reason := "runtime.entrypoint is required for external process connector invocation"
		resp.Checks = append(resp.Checks, Check{Name: "runtime_entrypoint", Result: "blocked", Detail: reason})
		return complete(resp, start, "needs_configuration", reason), nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return complete(resp, start, "failed", err.Error()), nil
	}

	cmd := exec.CommandContext(ctx, entrypoint)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		resp.Checks = append(resp.Checks, Check{Name: "external_process", Result: "failed", Detail: detail})
		return complete(resp, start, "failed", detail), nil
	}

	var runtimeResp Response
	if err := json.Unmarshal(stdout.Bytes(), &runtimeResp); err != nil {
		resp.Raw = strings.TrimSpace(stdout.String())
		resp.Checks = append(resp.Checks, Check{Name: "external_process", Result: "contract_warning", Detail: "Runtime completed but did not return connector_runtime_v1 JSON."})
		return complete(resp, start, "ok", ""), nil
	}

	runtimeResp.ConnectorID = firstNonEmpty(runtimeResp.ConnectorID, resp.ConnectorID)
	runtimeResp.Operation = firstNonEmpty(runtimeResp.Operation, resp.Operation)
	runtimeResp.RuntimeKind = firstNonEmpty(runtimeResp.RuntimeKind, resp.RuntimeKind)
	runtimeResp.RuntimeProtocol = firstNonEmpty(runtimeResp.RuntimeProtocol, resp.RuntimeProtocol)
	runtimeResp.InvocationID = firstNonEmpty(runtimeResp.InvocationID, resp.InvocationID)
	runtimeResp.ContractVersion = firstNonEmpty(runtimeResp.ContractVersion, "connector_runtime_v1")
	runtimeResp.SafeMode = true
	runtimeResp.DryRun = resp.DryRun
	if runtimeResp.Status == "" {
		runtimeResp.Status = "ok"
	}
	return finish(runtimeResp, start), nil
}

func runtimeEndpoint(manifest catalog.ConnectorManifest, req Request) string {
	base := strings.TrimRight(strings.TrimSpace(req.RuntimeEndpoint), "/")
	if base == "" {
		return ""
	}
	if req.Operation == OperationHealth && manifest.Runtime.HealthCheck != "" {
		return base + "/" + strings.TrimLeft(manifest.Runtime.HealthCheck, "/")
	}
	return base
}

func complete(resp Response, start time.Time, status string, errText string) Response {
	resp.Status = status
	resp.Error = errText
	return finish(resp, start)
}

func finish(resp Response, start time.Time) Response {
	completed := time.Now().UTC()
	resp.CompletedAt = completed.Format(time.RFC3339)
	resp.DurationMillis = completed.Sub(start).Milliseconds()
	if resp.StartedAt == "" {
		resp.StartedAt = start.Format(time.RFC3339)
	}
	if resp.ContractVersion == "" {
		resp.ContractVersion = "connector_runtime_v1"
	}
	return resp
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
