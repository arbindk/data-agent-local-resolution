package platform

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) azureBlobExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	exportType := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/exports/azureblob/"), "/")
	if exportType == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "export type required"})
		return
	}

	executionID := strings.TrimSpace(r.URL.Query().Get("execution_id"))
	if executionID == "" {
		executionID = s.store.LatestExecutionID()
	}
	if executionID == "" {
		writeJSON(w, http.StatusNotFound, APIError{Error: "no execution evidence available"})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	summary, _ := s.store.GetManifestSummary(executionID)
	detailsRaw, _ := s.store.GetManifestDetails(executionID)
	provenanceRaw, _ := s.store.GetProvenance(executionID)
	actionsRaw, _ := s.store.GetActionResults(executionID)
	audit, _ := s.store.ListAuditEvents(200)

	details := exportItems(detailsRaw)
	provenance := exportItems(provenanceRaw)
	actions := exportItems(actionsRaw)

	if exportType == "manifest" && format == "csv" {
		writeManifestCSV(w, executionID, details)
		return
	}

	payload := map[string]any{
		"export_type":  exportType,
		"execution_id": executionID,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}

	switch exportType {
	case "executive-summary":
		payload["summary"] = summary
		payload["top_risk_objects"] = limitItems(details, 10)
		payload["action_posture"] = map[string]any{
			"planned_actions": len(actions),
			"guardian_gate":   "required_before_writeback",
			"real_writeback":  "metadata_and_index_tags_only",
		}

	case "manifest":
		payload["summary"] = summary
		payload["items"] = details

	case "guardian":
		payload["actions"] = actions
		payload["provenance"] = provenance
		payload["controls"] = []string{
			"action_mode_apply_required",
			"writeback_enabled_required",
			"guardian_allow_required",
			"metadata_or_index_tag_action_required",
		}

	case "compliance":
		payload["summary"] = summary
		payload["provenance"] = provenance
		payload["audit_events"] = audit
		payload["policy_precedence"] = defaultPolicyPrecedence()
		payload["jurisdiction_rules"] = defaultJurisdictionRules()

	case "ai-readiness":
		payload["summary"] = summary
		payload["readiness"] = buildAIReadiness(details)
		payload["excluded_objects"] = filterItems(details, func(item map[string]any) bool {
			level := strings.ToLower(stringField(item, "risk_level"))
			return level == "critical" || level == "high"
		})

	default:
		writeJSON(w, http.StatusNotFound, APIError{Error: "unknown Azure Blob export type"})
		return
	}

	writeDownloadJSON(w, "everest-azureblob-"+exportType+"-"+executionID+".json", payload)
}

func writeDownloadJSON(w http.ResponseWriter, filename string, payload any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func writeManifestCSV(w http.ResponseWriter, executionID string, details []map[string]any) {
	var b bytes.Buffer
	writer := csv.NewWriter(&b)

	_ = writer.Write([]string{
		"execution_id",
		"file_id",
		"path",
		"risk_level",
		"classification",
		"action_requested",
		"action_status",
		"guardian_decision",
		"size_bytes",
	})

	for _, item := range details {
		_ = writer.Write([]string{
			executionID,
			stringField(item, "file_id"),
			stringField(item, "path"),
			stringField(item, "risk_level"),
			stringField(item, "classification"),
			stringField(item, "action_requested"),
			stringField(item, "action_status"),
			stringField(item, "guardian_decision"),
			stringField(item, "size_bytes"),
		})
	}

	writer.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="everest-azureblob-manifest-`+executionID+`.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.Bytes())
}

func exportItems(raw []any) []map[string]any {
	items := make([]map[string]any, 0)
	for _, value := range raw {
		if wrapped, ok := value.(map[string]any); ok {
			if nested, ok := wrapped["items"].([]any); ok {
				items = append(items, exportItems(nested)...)
				continue
			}
			items = append(items, wrapped)
		}
	}
	return items
}

func limitItems(items []map[string]any, limit int) []map[string]any {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func filterItems(items []map[string]any, keep func(map[string]any) bool) []map[string]any {
	out := []map[string]any{}
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func buildAIReadiness(details []map[string]any) map[string]any {
	ready := 0
	governed := 0
	excluded := 0

	for _, item := range details {
		level := strings.ToLower(stringField(item, "risk_level"))
		switch level {
		case "critical", "high":
			excluded++
		case "medium":
			governed++
		default:
			ready++
		}
	}

	total := len(details)
	score := 0
	if total > 0 {
		score = int(float64(ready+governed) / float64(total) * 100)
	}

	return map[string]any{
		"ai_ready_objects":        ready,
		"governed_objects":        governed,
		"excluded_sensitive_risk": excluded,
		"quality_score":           score,
		"rule":                    "critical/high risk objects are excluded until remediated or approved",
	}
}

func stringField(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.Trim(strings.TrimSpace(toJSON(v)), `"`), `\"`, `"`))
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
