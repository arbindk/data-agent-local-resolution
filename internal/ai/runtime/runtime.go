package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	aicatalog "everest.local/data-agent-policy-resolver/internal/ai/catalog"
	aiproviders "everest.local/data-agent-policy-resolver/internal/ai/providers"
)

const (
	TaskChat             = "chat"
	TaskClassify         = "classification"
	TaskSummarize        = "summarization"
	TaskExplainPolicy    = "policy_explanation"
	TaskRecommendActions = "recommend_actions"
	TaskPrepareHITL      = "hitl_request_prefill"
)

type Request struct {
	RuntimeRequestID  string           `json:"runtime_request_id,omitempty"`
	ProviderID        string           `json:"provider_id"`
	Task              string           `json:"task"`
	Prompt            string           `json:"prompt,omitempty"`
	Model             string           `json:"model,omitempty"`
	RuntimeEndpoint   string           `json:"runtime_endpoint,omitempty"`
	AutomationMode    string           `json:"automation_mode,omitempty"`
	RedactionMode     string           `json:"redaction_mode,omitempty"`
	PromptAudit       bool             `json:"prompt_audit"`
	EvidenceGrounding bool             `json:"evidence_grounding"`
	TimeoutSeconds    int              `json:"timeout_seconds,omitempty"`
	Context           map[string]any   `json:"context,omitempty"`
	Evidence          []map[string]any `json:"evidence,omitempty"`
	Messages          []RuntimeMessage `json:"messages,omitempty"`
}

type RuntimeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Status          string            `json:"status"`
	ProviderID      string            `json:"provider_id"`
	ProviderType    string            `json:"provider_type"`
	Task            string            `json:"task"`
	Model           string            `json:"model,omitempty"`
	RuntimeKind     string            `json:"runtime_kind"`
	RuntimeProtocol string            `json:"runtime_protocol,omitempty"`
	InvocationID    string            `json:"invocation_id"`
	Output          string            `json:"output,omitempty"`
	Recommendations []Recommendation  `json:"recommendations,omitempty"`
	Classifications []Classification  `json:"classifications,omitempty"`
	Summary         string            `json:"summary,omitempty"`
	PromptAudit     PromptAuditRecord `json:"prompt_audit"`
	Controls        []string          `json:"controls"`
	Checks          []Check           `json:"checks,omitempty"`
	Raw             any               `json:"raw,omitempty"`
	StartedAt       string            `json:"started_at"`
	CompletedAt     string            `json:"completed_at"`
	DurationMillis  int64             `json:"duration_ms"`
	Error           string            `json:"error,omitempty"`
	ContractVersion string            `json:"contract_version"`
}

type Recommendation struct {
	RecommendationID string  `json:"recommendation_id"`
	Operation        string  `json:"operation"`
	ObjectID         string  `json:"object_id,omitempty"`
	Path             string  `json:"path,omitempty"`
	RiskLevel        string  `json:"risk_level,omitempty"`
	Confidence       float64 `json:"confidence"`
	RequiresHITL     bool    `json:"requires_hitl"`
	Rationale        string  `json:"rationale"`
	Status           string  `json:"status"`
}

type Classification struct {
	ObjectID       string  `json:"object_id,omitempty"`
	Path           string  `json:"path,omitempty"`
	Classification string  `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Rationale      string  `json:"rationale,omitempty"`
}

type PromptAuditRecord struct {
	Enabled              bool   `json:"enabled"`
	EvidenceGrounded     bool   `json:"evidence_grounded"`
	RedactionMode        string `json:"redaction_mode"`
	CustomerPromptSent   bool   `json:"customer_prompt_sent"`
	CustomerDataBoundary string `json:"customer_data_boundary"`
}

type Check struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	Detail string `json:"detail"`
}

func Invoke(ctx context.Context, manifest aicatalog.AIProviderManifest, cfg aiproviders.Config, req Request, credential string) (Response, error) {
	start := time.Now().UTC()
	req = withDefaults(manifest, cfg, req)

	resp := Response{
		Status:          "pending",
		ProviderID:      manifest.ProviderID,
		ProviderType:    manifest.ProviderType,
		Task:            req.Task,
		Model:           req.Model,
		RuntimeKind:     manifest.Runtime.Kind,
		RuntimeProtocol: manifest.Runtime.Protocol,
		InvocationID:    req.RuntimeRequestID,
		StartedAt:       start.Format(time.RFC3339),
		ContractVersion: "ai_provider_runtime_v1",
		PromptAudit: PromptAuditRecord{
			Enabled:              req.PromptAudit,
			EvidenceGrounded:     req.EvidenceGrounding,
			RedactionMode:        req.RedactionMode,
			CustomerPromptSent:   false,
			CustomerDataBoundary: dataBoundary(manifest),
		},
		Controls: []string{
			"prompt_audit_policy_recorded",
			"evidence_grounding_policy_recorded",
			"autonomous_apply_blocked",
			"hitl_required_before_executable_actions",
		},
	}

	if err := validateTask(manifest, req.Task); err != nil {
		return complete(resp, start, "blocked", err.Error()), nil
	}
	if strings.EqualFold(req.AutomationMode, "autonomous_apply") {
		return complete(resp, start, "blocked", "autonomous AI apply is not supported"), nil
	}

	switch manifest.Runtime.Kind {
	case "builtin":
		return localPolicyResponse(manifest, req, resp, start), nil

	case "http", "local_http", "custom_http":
		return invokeHTTP(ctx, manifest, cfg, req, credential, resp, start)

	case "external_process":
		return complete(resp, start, "needs_configuration", "external process AI providers are declared but not enabled in this build"), nil

	default:
		return complete(resp, start, "blocked", "unsupported AI runtime kind: "+manifest.Runtime.Kind), nil
	}
}

func withDefaults(manifest aicatalog.AIProviderManifest, cfg aiproviders.Config, req Request) Request {
	if strings.TrimSpace(req.ProviderID) == "" {
		req.ProviderID = manifest.ProviderID
	}
	if strings.TrimSpace(req.Task) == "" {
		req.Task = TaskChat
	}
	req.Task = strings.ToLower(strings.TrimSpace(req.Task))
	if strings.TrimSpace(req.Model) == "" {
		req.Model = cfg.Model
	}
	if strings.TrimSpace(req.Model) == "" && len(manifest.ModelOptions) > 0 {
		req.Model = manifest.ModelOptions[0].ModelID
	}
	if strings.TrimSpace(req.AutomationMode) == "" {
		req.AutomationMode = cfg.AutomationMode
	}
	if strings.TrimSpace(req.RedactionMode) == "" {
		req.RedactionMode = cfg.RedactionMode
	}
	if strings.TrimSpace(req.RedactionMode) == "" {
		req.RedactionMode = "metadata_only"
	}
	if !req.PromptAudit {
		req.PromptAudit = cfg.PromptAuditEnabled
	}
	if !req.EvidenceGrounding {
		req.EvidenceGrounding = cfg.EvidenceGroundingRequired
	}
	if strings.TrimSpace(req.RuntimeRequestID) == "" {
		req.RuntimeRequestID = "ai-runtime-" + time.Now().UTC().Format("20060102-150405.000000000")
	}
	if req.TimeoutSeconds <= 0 || req.TimeoutSeconds > 120 {
		req.TimeoutSeconds = 45
	}
	return req
}

func validateTask(manifest aicatalog.AIProviderManifest, task string) error {
	caps := manifest.Capabilities
	switch task {
	case TaskChat:
		if caps.Chat || manifest.ProviderType == "local_policy_rules" {
			return nil
		}
	case TaskClassify:
		if caps.Classification {
			return nil
		}
	case TaskSummarize:
		if caps.Summarization {
			return nil
		}
	case TaskExplainPolicy:
		if caps.PolicyExplanation {
			return nil
		}
	case TaskRecommendActions, TaskPrepareHITL:
		if caps.Recommendations || caps.ActionPlanning {
			return nil
		}
	default:
		return fmt.Errorf("unsupported AI runtime task: %s", task)
	}
	return fmt.Errorf("AI provider %s does not advertise task %s", manifest.ProviderID, task)
}

func localPolicyResponse(manifest aicatalog.AIProviderManifest, req Request, resp Response, start time.Time) Response {
	switch req.Task {
	case TaskRecommendActions, TaskPrepareHITL:
		resp.Recommendations = localRecommendations(req)
		resp.Output = fmt.Sprintf("Generated %d evidence-grounded recommendations locally.", len(resp.Recommendations))
	case TaskClassify:
		resp.Classifications = localClassifications(req)
		resp.Output = fmt.Sprintf("Generated %d local classifications from metadata evidence.", len(resp.Classifications))
	case TaskSummarize:
		resp.Summary = localSummary(req)
		resp.Output = resp.Summary
	case TaskExplainPolicy:
		resp.Output = "Everest policy flow: resolve signed policy, run local evidence generation, apply Guardian decisions, then require HITL before restricted or destructive actions."
	default:
		resp.Output = "Everest Local Policy Copilot is ready. It can summarize evidence, explain policy posture, classify metadata, and prepare governed HITL recommendations without external model calls."
	}
	resp.Checks = append(resp.Checks, Check{Name: "local_policy_runtime", Result: "pass", Detail: "No customer prompt was sent outside the environment."})
	return complete(resp, start, "ok", "")
}

func invokeHTTP(ctx context.Context, manifest aicatalog.AIProviderManifest, cfg aiproviders.Config, req Request, credential string, resp Response, start time.Time) (Response, error) {
	endpoint := aiEndpoint(manifest, cfg, req)
	if endpoint == "" {
		return complete(resp, start, "needs_configuration", "endpoint_url is required for AI provider runtime invocation"), nil
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
	httpReq.Header.Set("X-Everest-Contract", "ai_provider_runtime_v1")
	if strings.TrimSpace(credential) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+credential)
		if cfg.AuthScheme == "api_key" {
			httpReq.Header.Set("api-key", credential)
		}
	}

	client := &http.Client{Timeout: time.Duration(req.TimeoutSeconds) * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Checks = append(resp.Checks, Check{Name: "http_ai_runtime", Result: "failed", Detail: err.Error()})
		return complete(resp, start, "failed", err.Error()), nil
	}
	defer httpResp.Body.Close()

	var runtimeResp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&runtimeResp); err != nil {
		reason := "HTTP AI provider did not return ai_provider_runtime_v1 JSON: " + err.Error()
		resp.Checks = append(resp.Checks, Check{Name: "http_ai_runtime", Result: "failed", Detail: reason})
		return complete(resp, start, "failed", reason), nil
	}

	runtimeResp.ProviderID = firstNonEmpty(runtimeResp.ProviderID, resp.ProviderID)
	runtimeResp.ProviderType = firstNonEmpty(runtimeResp.ProviderType, resp.ProviderType)
	runtimeResp.Task = firstNonEmpty(runtimeResp.Task, resp.Task)
	runtimeResp.Model = firstNonEmpty(runtimeResp.Model, resp.Model)
	runtimeResp.RuntimeKind = firstNonEmpty(runtimeResp.RuntimeKind, resp.RuntimeKind)
	runtimeResp.RuntimeProtocol = firstNonEmpty(runtimeResp.RuntimeProtocol, resp.RuntimeProtocol)
	runtimeResp.InvocationID = firstNonEmpty(runtimeResp.InvocationID, resp.InvocationID)
	runtimeResp.ContractVersion = firstNonEmpty(runtimeResp.ContractVersion, "ai_provider_runtime_v1")
	runtimeResp.PromptAudit.Enabled = req.PromptAudit
	runtimeResp.PromptAudit.EvidenceGrounded = req.EvidenceGrounding
	runtimeResp.PromptAudit.RedactionMode = req.RedactionMode
	runtimeResp.PromptAudit.CustomerPromptSent = true
	runtimeResp.PromptAudit.CustomerDataBoundary = dataBoundary(manifest)
	if runtimeResp.Status == "" {
		if httpResp.StatusCode >= 400 {
			runtimeResp.Status = "failed"
		} else {
			runtimeResp.Status = "ok"
		}
	}
	return finish(runtimeResp, start), nil
}

func localRecommendations(req Request) []Recommendation {
	recs := []Recommendation{}
	for i, item := range req.Evidence {
		risk := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["risk_level"])))
		if risk == "" {
			risk = strings.ToLower(strings.TrimSpace(fmt.Sprint(item["risk"])))
		}
		if risk != "critical" && risk != "high" {
			continue
		}
		objectID := firstNonEmpty(fmt.Sprint(item["file_id"]), fmt.Sprint(item["object_id"]))
		path := strings.TrimSpace(fmt.Sprint(item["path"]))
		operation := "quarantine"
		if strings.Contains(strings.ToLower(req.Prompt), "archive") {
			operation = "archive"
		}
		recs = append(recs, Recommendation{
			RecommendationID: "ai-runtime-rec-" + fmt.Sprint(i+1),
			Operation:        operation,
			ObjectID:         objectID,
			Path:             path,
			RiskLevel:        risk,
			Confidence:       0.82,
			RequiresHITL:     true,
			Rationale:        "Local policy runtime selected a governed action because evidence is high risk and requires human approval before apply.",
			Status:           "needs_approval",
		})
	}
	if len(recs) == 0 {
		recs = append(recs, Recommendation{
			RecommendationID: "ai-runtime-rec-empty",
			Operation:        "metadata_scan",
			Confidence:       0.5,
			RequiresHITL:     false,
			Rationale:        "No high-risk evidence was provided to the AI runtime request.",
			Status:           "scan_required",
		})
	}
	return recs
}

func localClassifications(req Request) []Classification {
	items := []Classification{}
	for i, item := range req.Evidence {
		classification := "internal"
		text := strings.ToLower(fmt.Sprint(item["classification"]) + " " + fmt.Sprint(item["path"]) + " " + fmt.Sprint(item["name"]))
		if strings.Contains(text, "confidential") || strings.Contains(text, "secret") {
			classification = "confidential"
		}
		items = append(items, Classification{
			ObjectID:       firstNonEmpty(fmt.Sprint(item["file_id"]), fmt.Sprint(item["object_id"])),
			Path:           strings.TrimSpace(fmt.Sprint(item["path"])),
			Classification: classification,
			Confidence:     0.74,
			Rationale:      "Local metadata-based classification profile.",
		})
		if i >= 24 {
			break
		}
	}
	return items
}

func localSummary(req Request) string {
	count := len(req.Evidence)
	if count == 0 {
		return "No evidence rows were provided. Run or select a governed scan before summarization."
	}
	high := 0
	for _, item := range req.Evidence {
		risk := strings.ToLower(fmt.Sprint(item["risk_level"]) + " " + fmt.Sprint(item["risk"]))
		if strings.Contains(risk, "critical") || strings.Contains(risk, "high") {
			high++
		}
	}
	return fmt.Sprintf("Local evidence summary: %d evidence rows reviewed, %d high-impact rows require Guardian/HITL attention.", count, high)
}

func aiEndpoint(manifest aicatalog.AIProviderManifest, cfg aiproviders.Config, req Request) string {
	if strings.TrimSpace(req.RuntimeEndpoint) != "" {
		return strings.TrimSpace(req.RuntimeEndpoint)
	}
	if strings.TrimSpace(cfg.EndpointURL) != "" {
		return strings.TrimSpace(cfg.EndpointURL)
	}
	if manifest.Runtime.Endpoint != "" && manifest.Runtime.Endpoint != "customer_configured" {
		return strings.TrimSpace(manifest.Runtime.Endpoint)
	}
	return ""
}

func dataBoundary(manifest aicatalog.AIProviderManifest) string {
	if manifest.DataPolicy.CustomerDataLeavesEnvironment {
		return "external_provider"
	}
	if manifest.DataPolicy.SupportsAirGap {
		return "local_or_air_gapped"
	}
	return "customer_private_network"
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
		resp.ContractVersion = "ai_provider_runtime_v1"
	}
	return resp
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && strings.TrimSpace(value) != "<nil>" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
