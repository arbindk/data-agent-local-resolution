package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	airuntime "everest.local/data-agent-policy-resolver/internal/ai/runtime"
	connectorruntime "everest.local/data-agent-policy-resolver/internal/connectors/runtime"
)

func main() {
	infoLog := log.New(os.Stdout, "", log.LstdFlags)
	errorLog := log.New(os.Stderr, "", log.LstdFlags)

	port := strings.TrimSpace(os.Getenv("EVEREST_SAMPLE_RUNTIME_PORT"))
	if port == "" {
		port = "7071"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/connector/healthz", connectorRuntime)
	mux.HandleFunc("/connector", connectorRuntime)
	mux.HandleFunc("/ai", aiRuntime)

	addr := ":" + port
	infoLog.Printf("Everest sample plugin runtime listening on http://localhost%s", addr)
	infoLog.Printf("connector endpoint: http://localhost%s/connector", addr)
	infoLog.Printf("AI endpoint: http://localhost%s/ai", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		errorLog.Fatalf("sample plugin runtime failed: %v", err)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "everest-sample-plugin-runtime",
	})
}

func connectorRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req connectorruntime.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	start := time.Now().UTC()
	resp := connectorruntime.Response{
		Status:          "ok",
		ConnectorID:     firstNonEmpty(req.ConnectorID, "customer-filesystem"),
		Operation:       firstNonEmpty(req.Operation, connectorruntime.OperationHealth),
		RuntimeKind:     "http",
		RuntimeProtocol: "connector_runtime_v1_sample",
		InvocationID:    firstNonEmpty(req.RuntimeRequestID, "sample-connector-"+start.Format("150405.000000000")),
		SafeMode:        true,
		DryRun:          true,
		Applied:         false,
		StartedAt:       start.Format(time.RFC3339),
		ContractVersion: "connector_runtime_v1",
		ExecutionControl: []string{
			"sample_runtime",
			"dry_run_only",
			"no_customer_system_modified",
		},
		Checks: []connectorruntime.Check{
			{Name: "contract", Result: "pass", Detail: "connector_runtime_v1 request accepted."},
			{Name: "safety", Result: "pass", Detail: "Sample connector runtime is dry-run only."},
		},
	}

	switch req.Operation {
	case connectorruntime.OperationHealth, "":
		resp.Checks = append(resp.Checks, connectorruntime.Check{Name: "sample_connector_runtime", Result: "pass", Detail: "Sample connector runtime is reachable."})

	case connectorruntime.OperationDiscover:
		resp.Records = []any{
			map[string]any{"source_id": "sample-share-finance", "name": "Finance Share", "type": "filesystem_share", "path": "\\\\sample\\finance"},
			map[string]any{"source_id": "sample-share-legal", "name": "Legal Share", "type": "filesystem_share", "path": "\\\\sample\\legal"},
		}
		resp.Checks = append(resp.Checks, connectorruntime.Check{Name: "discover_sources", Result: "pass", Detail: "Returned 2 sample filesystem shares."})

	case connectorruntime.OperationScanMetadata:
		resp.Records = sampleConnectorObjects()
		resp.NormalizedBatch = map[string]any{
			"batch_id":       "sample-connector-batch-001",
			"data_store_id":  firstNonEmpty(req.DataStoreID, "sample-filesystem"),
			"source_system":  "customer_filesystem_sample",
			"records_count":  len(resp.Records),
			"contract":       "normalized_batch",
			"runtime_sample": true,
		}
		resp.Checks = append(resp.Checks, connectorruntime.Check{Name: "scan_metadata", Result: "pass", Detail: "Returned 2 sample normalized object records."})

	case connectorruntime.OperationPreviewAction, connectorruntime.OperationExecuteAction:
		actionID := "metadata_scan"
		if req.Action != nil && strings.TrimSpace(fmt.Sprint(req.Action["action_id"])) != "" {
			actionID = strings.TrimSpace(fmt.Sprint(req.Action["action_id"]))
		}
		resp.Status = "planned"
		resp.ActionResult = map[string]any{
			"action_id":      actionID,
			"preview_only":   true,
			"applied":        false,
			"sample_runtime": true,
			"reason":         "Sample connector runtime is dry-run only.",
		}
		resp.Checks = append(resp.Checks, connectorruntime.Check{Name: "action_preview", Result: "pass", Detail: "Action was planned only; no sample system was modified."})

	default:
		resp.Status = "blocked"
		resp.Error = "unsupported sample connector operation: " + req.Operation
	}

	resp.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	resp.DurationMillis = time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, resp)
}

func aiRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req airuntime.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	start := time.Now().UTC()
	resp := airuntime.Response{
		Status:          "ok",
		ProviderID:      firstNonEmpty(req.ProviderID, "customer-private-llm"),
		ProviderType:    "sample_http_ai",
		Task:            firstNonEmpty(req.Task, airuntime.TaskChat),
		Model:           firstNonEmpty(req.Model, "sample-governance-profile"),
		RuntimeKind:     "custom_http",
		RuntimeProtocol: "ai_provider_runtime_v1_sample",
		InvocationID:    firstNonEmpty(req.RuntimeRequestID, "sample-ai-"+start.Format("150405.000000000")),
		StartedAt:       start.Format(time.RFC3339),
		ContractVersion: "ai_provider_runtime_v1",
		PromptAudit: airuntime.PromptAuditRecord{
			Enabled:              req.PromptAudit,
			EvidenceGrounded:     req.EvidenceGrounding,
			RedactionMode:        firstNonEmpty(req.RedactionMode, "metadata_only"),
			CustomerPromptSent:   true,
			CustomerDataBoundary: "sample_local_http_runtime",
		},
		Controls: []string{
			"sample_ai_runtime",
			"structured_json_returned",
			"autonomous_apply_blocked",
			"hitl_required_before_actions",
		},
		Checks: []airuntime.Check{
			{Name: "contract", Result: "pass", Detail: "ai_provider_runtime_v1 request accepted."},
			{Name: "prompt_audit", Result: "pass", Detail: "Prompt audit metadata returned in the response."},
			{Name: "safety", Result: "pass", Detail: "Sample AI runtime cannot execute actions directly."},
		},
	}

	switch req.Task {
	case airuntime.TaskRecommendActions, airuntime.TaskPrepareHITL:
		resp.Output = "Sample AI runtime generated governed action recommendations."
		resp.Recommendations = sampleAIRecommendations(req)
		resp.Checks = append(resp.Checks, airuntime.Check{Name: "recommendations", Result: "pass", Detail: fmt.Sprintf("Returned %d recommendation rows.", len(resp.Recommendations))})
	case airuntime.TaskClassify:
		resp.Output = "Sample AI runtime classified evidence metadata."
		resp.Classifications = sampleAIClassifications(req)
		resp.Checks = append(resp.Checks, airuntime.Check{Name: "classification", Result: "pass", Detail: fmt.Sprintf("Returned %d classification rows.", len(resp.Classifications))})
	case airuntime.TaskSummarize:
		resp.Summary = fmt.Sprintf("Sample AI summary: %d evidence rows reviewed. High-risk evidence should remain behind Guardian and HITL.", len(req.Evidence))
		resp.Output = resp.Summary
		resp.Checks = append(resp.Checks, airuntime.Check{Name: "summarization", Result: "pass", Detail: fmt.Sprintf("Summarized %d evidence rows.", len(req.Evidence))})
	case airuntime.TaskExplainPolicy:
		resp.Output = "Sample AI policy explanation: Everest resolves signed policies, generates evidence locally, applies Guardian decisions, and requires HITL before restricted or destructive actions."
		resp.Checks = append(resp.Checks, airuntime.Check{Name: "policy_explanation", Result: "pass", Detail: "Returned a sample policy explanation."})
	default:
		resp.Output = "Sample AI runtime is reachable and returning ai_provider_runtime_v1 JSON."
		resp.Checks = append(resp.Checks, airuntime.Check{Name: "chat", Result: "pass", Detail: "Returned a sample chat response."})
	}

	resp.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	resp.DurationMillis = time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, resp)
}

func sampleConnectorObjects() []any {
	return []any{
		map[string]any{
			"file_id":               "samplefs:finance:payroll.xlsx",
			"path":                  "\\\\sample\\finance\\payroll.xlsx",
			"name":                  "payroll.xlsx",
			"size_bytes":            248320,
			"classification":        "confidential",
			"risk_level":            "high",
			"permission_risk_score": 86,
			"source_system":         "customer_filesystem_sample",
		},
		map[string]any{
			"file_id":               "samplefs:legal:retention-policy.pdf",
			"path":                  "\\\\sample\\legal\\retention-policy.pdf",
			"name":                  "retention-policy.pdf",
			"size_bytes":            98304,
			"classification":        "internal",
			"risk_level":            "medium",
			"permission_risk_score": 42,
			"source_system":         "customer_filesystem_sample",
		},
	}
}

func sampleAIRecommendations(req airuntime.Request) []airuntime.Recommendation {
	if len(req.Evidence) == 0 {
		return []airuntime.Recommendation{{
			RecommendationID: "sample-ai-rec-empty",
			Operation:        "metadata_scan",
			Confidence:       0.55,
			RequiresHITL:     false,
			Rationale:        "No evidence rows were supplied to the sample AI runtime.",
			Status:           "scan_required",
		}}
	}

	recs := []airuntime.Recommendation{}
	for i, row := range req.Evidence {
		risk := strings.ToLower(fmt.Sprint(row["risk_level"]))
		if risk != "high" && risk != "critical" {
			continue
		}
		recs = append(recs, airuntime.Recommendation{
			RecommendationID: fmt.Sprintf("sample-ai-rec-%03d", i+1),
			Operation:        "quarantine",
			ObjectID:         firstNonEmpty(fmt.Sprint(row["object_id"]), fmt.Sprint(row["file_id"])),
			Path:             fmt.Sprint(row["path"]),
			RiskLevel:        risk,
			Confidence:       0.83,
			RequiresHITL:     true,
			Rationale:        "Sample AI recommends quarantine for high-risk evidence and requires HITL before apply.",
			Status:           "needs_approval",
		})
	}
	return recs
}

func sampleAIClassifications(req airuntime.Request) []airuntime.Classification {
	items := []airuntime.Classification{}
	for _, row := range req.Evidence {
		classification := "internal"
		text := strings.ToLower(fmt.Sprint(row["path"]) + " " + fmt.Sprint(row["classification"]))
		if strings.Contains(text, "confidential") || strings.Contains(text, "payroll") {
			classification = "confidential"
		}
		items = append(items, airuntime.Classification{
			ObjectID:       firstNonEmpty(fmt.Sprint(row["object_id"]), fmt.Sprint(row["file_id"])),
			Path:           fmt.Sprint(row["path"]),
			Classification: classification,
			Confidence:     0.79,
			Rationale:      "Sample metadata classifier.",
		})
	}
	return items
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
