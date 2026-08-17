package api

import (
	"net/http"

	"everest.local/data-agent-policy-resolver/internal/pluginruntime"
)

func (s *Server) runtimeAuditHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	runtimeType := r.URL.Query().Get("type")
	limit := queryInt(r, "limit", 100)
	items := s.runtimeAudit.List(runtimeType, limit)

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(items),
		"items":  items,
	})
}

func runtimeAuditRecord(
	runtimeType string,
	subjectID string,
	operation string,
	status string,
	runtimeKind string,
	contractVersion string,
	dryRun bool,
	safeMode bool,
	durationMillis int64,
	errText string,
	controls []string,
) pluginruntime.AuditRecord {
	return pluginruntime.AuditRecord{
		RuntimeType:     runtimeType,
		SubjectID:       subjectID,
		Operation:       operation,
		Status:          status,
		RuntimeKind:     runtimeKind,
		ContractVersion: contractVersion,
		DryRun:          dryRun,
		SafeMode:        safeMode,
		DurationMillis:  durationMillis,
		Error:           errText,
		Controls:        controls,
	}
}
