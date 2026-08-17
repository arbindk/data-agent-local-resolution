package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	aiproviders "everest.local/data-agent-policy-resolver/internal/ai/providers"
	airuntime "everest.local/data-agent-policy-resolver/internal/ai/runtime"
	connectorcatalog "everest.local/data-agent-policy-resolver/internal/connectors/catalog"
	connectorruntime "everest.local/data-agent-policy-resolver/internal/connectors/runtime"
)

type PluginContractTestRequest struct {
	PluginType      string         `json:"plugin_type"`
	ConnectorID     string         `json:"connector_id,omitempty"`
	ProviderID      string         `json:"provider_id,omitempty"`
	RuntimeEndpoint string         `json:"runtime_endpoint"`
	Model           string         `json:"model,omitempty"`
	TimeoutSeconds  int            `json:"timeout_seconds,omitempty"`
	Scope           map[string]any `json:"scope,omitempty"`
}

type PluginContractTestReport struct {
	Status          string                    `json:"status"`
	PluginType      string                    `json:"plugin_type"`
	SubjectID       string                    `json:"subject_id"`
	RuntimeEndpoint string                    `json:"runtime_endpoint"`
	ReadyForPortal  bool                      `json:"ready_for_portal"`
	ReadyForApply   bool                      `json:"ready_for_apply"`
	Checks          []PluginContractTestCheck `json:"checks"`
	StartedAt       string                    `json:"started_at"`
	CompletedAt     string                    `json:"completed_at"`
	DurationMillis  int64                     `json:"duration_ms"`
	NextSteps       []string                  `json:"next_steps"`
}

type PluginContractTestCheck struct {
	Name     string `json:"name"`
	Result   string `json:"result"`
	Detail   string `json:"detail"`
	Required bool   `json:"required"`
}

func (s *Server) pluginContractTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req PluginContractTestRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req.PluginType = strings.ToLower(strings.TrimSpace(req.PluginType))
	req.RuntimeEndpoint = strings.TrimSpace(req.RuntimeEndpoint)
	if req.TimeoutSeconds <= 0 || req.TimeoutSeconds > 120 {
		req.TimeoutSeconds = 30
	}

	var report PluginContractTestReport
	switch req.PluginType {
	case "connector":
		report = s.runConnectorContractTest(r, req)
	case "ai", "ai_provider":
		report = s.runAIProviderContractTest(r, req)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "plugin_type must be connector or ai_provider"})
		return
	}

	s.runtimeAudit.Add(runtimeAuditRecord(
		"contract_test",
		report.SubjectID,
		report.PluginType,
		report.Status,
		"contract_test",
		"plugin_contract_test_v1",
		true,
		true,
		report.DurationMillis,
		"",
		[]string{"manifest_validation", "runtime_contract_validation", "dry_run_safety"},
	))

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) runConnectorContractTest(r *http.Request, req PluginContractTestRequest) PluginContractTestReport {
	start := time.Now().UTC()
	subjectID := strings.TrimSpace(req.ConnectorID)
	report := PluginContractTestReport{
		Status:          "running",
		PluginType:      "connector",
		SubjectID:       subjectID,
		RuntimeEndpoint: req.RuntimeEndpoint,
		StartedAt:       start.Format(time.RFC3339),
	}

	add := func(name, result, detail string, required bool) {
		report.Checks = append(report.Checks, PluginContractTestCheck{Name: name, Result: result, Detail: detail, Required: required})
	}

	manifest, ok := findConnectorManifest(subjectID)
	if !ok {
		add("manifest", "fail", "Connector manifest was not found in catalog.", true)
		return finishContractReport(report, start)
	}
	add("manifest", "pass", "Connector manifest loaded: "+manifest.DisplayName, true)

	if req.RuntimeEndpoint == "" && manifest.Runtime.Kind != "builtin" {
		add("runtime_endpoint", "fail", "Runtime endpoint is required for customer connector contract test.", true)
		return finishContractReport(report, start)
	}

	tests := []connectorruntime.Request{
		{
			ConnectorID:     subjectID,
			Operation:       connectorruntime.OperationHealth,
			RuntimeEndpoint: req.RuntimeEndpoint,
			TimeoutSeconds:  req.TimeoutSeconds,
			Scope:           req.Scope,
		},
		{
			ConnectorID:     subjectID,
			Operation:       connectorruntime.OperationDiscover,
			RuntimeEndpoint: req.RuntimeEndpoint,
			TimeoutSeconds:  req.TimeoutSeconds,
			Scope:           req.Scope,
		},
		{
			ConnectorID:     subjectID,
			Operation:       connectorruntime.OperationScanMetadata,
			RuntimeEndpoint: req.RuntimeEndpoint,
			TimeoutSeconds:  req.TimeoutSeconds,
			Scope:           req.Scope,
			DataStoreID:     firstScopeString(req.Scope, "data_store_id", "contract-test-datastore"),
		},
		{
			ConnectorID:     subjectID,
			Operation:       connectorruntime.OperationPreviewAction,
			RuntimeEndpoint: req.RuntimeEndpoint,
			TimeoutSeconds:  req.TimeoutSeconds,
			ActionMode:      "dry_run",
			PreviewOnly:     true,
			Action:          map[string]any{"action_id": firstConnectorActionID(manifest)},
			Scope:           req.Scope,
		},
	}

	for _, runtimeReq := range tests {
		resp, err := connectorruntime.Invoke(r.Context(), manifest, runtimeReq)
		if err != nil {
			add(runtimeReq.Operation, "fail", err.Error(), true)
			continue
		}
		checkConnectorRuntimeResponse(&report, runtimeReq.Operation, resp)
	}

	return finishContractReport(report, start)
}

func (s *Server) runAIProviderContractTest(r *http.Request, req PluginContractTestRequest) PluginContractTestReport {
	start := time.Now().UTC()
	subjectID := strings.TrimSpace(req.ProviderID)
	report := PluginContractTestReport{
		Status:          "running",
		PluginType:      "ai_provider",
		SubjectID:       subjectID,
		RuntimeEndpoint: req.RuntimeEndpoint,
		StartedAt:       start.Format(time.RFC3339),
	}

	add := func(name, result, detail string, required bool) {
		report.Checks = append(report.Checks, PluginContractTestCheck{Name: name, Result: result, Detail: detail, Required: required})
	}

	manifest, ok := findAIProviderManifest(subjectID)
	if !ok {
		add("manifest", "fail", "AI provider manifest was not found in catalog.", true)
		return finishContractReport(report, start)
	}
	add("manifest", "pass", "AI provider manifest loaded: "+manifest.DisplayName, true)

	cfg := s.withAIProviderManifestDefaults(aiproviders.Config{
		ProviderID:           subjectID,
		EndpointURL:          req.RuntimeEndpoint,
		Model:                strings.TrimSpace(req.Model),
		AuthScheme:           "none",
		NetworkMode:          "air_gapped",
		AutomationMode:       "recommend_only",
		RedactionMode:        "metadata_only",
		PromptAuditEnabled:   true,
		AllowHITLSubmission:  true,
		AllowAutonomousApply: false,
	})

	if req.RuntimeEndpoint == "" && manifest.Runtime.Kind != "builtin" {
		add("runtime_endpoint", "fail", "Runtime endpoint is required for customer AI provider contract test.", true)
		return finishContractReport(report, start)
	}

	evidence := []map[string]any{{
		"object_id":       "contract-test-object-001",
		"path":            "contract-test/high-risk-file.pdf",
		"risk_level":      "high",
		"classification":  "confidential",
		"guardian_status": "review_required",
	}}
	tests := []airuntime.Request{
		{
			ProviderID:        subjectID,
			Task:              airuntime.TaskChat,
			Prompt:            "Return a short readiness response for Everest contract testing.",
			RuntimeEndpoint:   req.RuntimeEndpoint,
			Model:             cfg.Model,
			PromptAudit:       true,
			EvidenceGrounding: true,
			TimeoutSeconds:    req.TimeoutSeconds,
			Evidence:          evidence,
		},
		{
			ProviderID:        subjectID,
			Task:              airuntime.TaskSummarize,
			Prompt:            "Summarize this evidence.",
			RuntimeEndpoint:   req.RuntimeEndpoint,
			Model:             cfg.Model,
			PromptAudit:       true,
			EvidenceGrounding: true,
			TimeoutSeconds:    req.TimeoutSeconds,
			Evidence:          evidence,
		},
		{
			ProviderID:        subjectID,
			Task:              airuntime.TaskRecommendActions,
			Prompt:            "Recommend safe governed actions.",
			RuntimeEndpoint:   req.RuntimeEndpoint,
			Model:             cfg.Model,
			AutomationMode:    "recommend_only",
			RedactionMode:     "metadata_only",
			PromptAudit:       true,
			EvidenceGrounding: true,
			TimeoutSeconds:    req.TimeoutSeconds,
			Evidence:          evidence,
		},
	}

	for _, runtimeReq := range tests {
		resp, err := airuntime.Invoke(r.Context(), manifest, cfg, runtimeReq, "")
		if err != nil {
			add(runtimeReq.Task, "fail", err.Error(), true)
			continue
		}
		checkAIProviderRuntimeResponse(&report, runtimeReq.Task, resp)
	}

	return finishContractReport(report, start)
}

func checkConnectorRuntimeResponse(report *PluginContractTestReport, operation string, resp connectorruntime.Response) {
	add := func(name, result, detail string, required bool) {
		report.Checks = append(report.Checks, PluginContractTestCheck{Name: name, Result: result, Detail: detail, Required: required})
	}

	if resp.Status == "ok" || resp.Status == "planned" || resp.Status == "native_runtime" {
		add(operation+".status", "pass", "Runtime returned status "+resp.Status+".", true)
	} else {
		add(operation+".status", "fail", "Runtime returned status "+resp.Status+": "+resp.Error, true)
	}
	if resp.ContractVersion == "connector_runtime_v1" {
		add(operation+".contract_version", "pass", "connector_runtime_v1 confirmed.", true)
	} else {
		add(operation+".contract_version", "fail", "Expected connector_runtime_v1, got "+resp.ContractVersion+".", true)
	}
	if resp.SafeMode {
		add(operation+".safe_mode", "pass", "Runtime response kept safe_mode=true.", true)
	} else {
		add(operation+".safe_mode", "fail", "Runtime response did not set safe_mode=true.", true)
	}
	if operation == connectorruntime.OperationPreviewAction && resp.DryRun && !resp.Applied {
		add(operation+".dry_run", "pass", "Preview action stayed dry-run and did not apply.", true)
	}
	if operation == connectorruntime.OperationDiscover && len(resp.Records) > 0 {
		add(operation+".records", "pass", "Discover returned records.", false)
	}
	if operation == connectorruntime.OperationScanMetadata && (len(resp.Records) > 0 || len(resp.NormalizedBatch) > 0) {
		add(operation+".scan_output", "pass", "Scan returned records or normalized batch.", false)
	}
}

func checkAIProviderRuntimeResponse(report *PluginContractTestReport, task string, resp airuntime.Response) {
	add := func(name, result, detail string, required bool) {
		report.Checks = append(report.Checks, PluginContractTestCheck{Name: name, Result: result, Detail: detail, Required: required})
	}

	if resp.Status == "ok" {
		add(task+".status", "pass", "Runtime returned status ok.", true)
	} else {
		add(task+".status", "fail", "Runtime returned status "+resp.Status+": "+resp.Error, true)
	}
	if resp.ContractVersion == "ai_provider_runtime_v1" {
		add(task+".contract_version", "pass", "ai_provider_runtime_v1 confirmed.", true)
	} else {
		add(task+".contract_version", "fail", "Expected ai_provider_runtime_v1, got "+resp.ContractVersion+".", true)
	}
	if resp.PromptAudit.Enabled {
		add(task+".prompt_audit", "pass", "Prompt audit metadata returned.", true)
	} else {
		add(task+".prompt_audit", "fail", "Prompt audit metadata was not enabled.", true)
	}
	if containsString(resp.Controls, "autonomous_apply_blocked") {
		add(task+".autonomous_apply", "pass", "Runtime declares autonomous apply blocked.", true)
	} else {
		add(task+".autonomous_apply", "fail", "Runtime response did not declare autonomous apply blocked.", true)
	}
	if task == airuntime.TaskSummarize && (resp.Output != "" || resp.Summary != "") {
		add(task+".output", "pass", "Summarization returned output.", false)
	}
	if task == airuntime.TaskRecommendActions && len(resp.Recommendations) > 0 {
		add(task+".recommendations", "pass", "Recommendation task returned recommendation rows.", false)
	}
}

func finishContractReport(report PluginContractTestReport, start time.Time) PluginContractTestReport {
	completed := time.Now().UTC()
	report.CompletedAt = completed.Format(time.RFC3339)
	report.DurationMillis = completed.Sub(start).Milliseconds()

	failedRequired := false
	for _, check := range report.Checks {
		if check.Required && check.Result == "fail" {
			failedRequired = true
			break
		}
	}

	report.ReadyForPortal = !failedRequired
	report.ReadyForApply = false
	if report.ReadyForPortal {
		report.Status = "ready_for_portal"
		report.NextSteps = []string{
			"Review runtime audit history.",
			"Run customer-environment tests with representative sources and evidence.",
			"Keep apply disabled until Guardian and HITL approval verification is configured.",
		}
	} else {
		report.Status = "needs_work"
		report.NextSteps = []string{
			"Fix failed required checks.",
			"Confirm manifest ID and runtime endpoint.",
			"Verify the runtime returns the expected contract_version and safety controls.",
		}
	}
	return report
}

func firstConnectorActionID(manifest connectorcatalog.ConnectorManifest) string {
	if len(manifest.Actions) > 0 && strings.TrimSpace(manifest.Actions[0].ActionID) != "" {
		return manifest.Actions[0].ActionID
	}
	return "metadata_scan"
}

func firstScopeString(scope map[string]any, key string, fallback string) string {
	if scope == nil {
		return fallback
	}
	if value, ok := scope[key]; ok && strings.TrimSpace(valueString(value)) != "" {
		return strings.TrimSpace(valueString(value))
	}
	return fallback
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
