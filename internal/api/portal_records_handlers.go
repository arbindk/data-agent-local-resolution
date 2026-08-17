package api

import (
	"net/http"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) portalEvidence(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := []portalstate.EvidenceRecord{}
		if s.portalState != nil {
			items = s.portalState.ListEvidence(queryInt(r, "limit", 100))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})
	case http.MethodPost:
		var record portalstate.EvidenceRecord
		if err := jsonDecode(r, &record); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}
		recorded := s.recordEvidence(record)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"evidence": recorded,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) portalAuditEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := []portalstate.AuditEvent{}
		if s.portalState != nil {
			items = s.portalState.ListAudit(queryInt(r, "limit", 100))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})
	case http.MethodPost:
		var event portalstate.AuditEvent
		if err := jsonDecode(r, &event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}
		recorded := s.recordAudit(event)
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"event":  recorded,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) portalActionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	items := []portalstate.ActionRecord{}
	if s.portalState != nil {
		items = s.portalState.ListActions(queryInt(r, "limit", 100))
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" && strings.HasSuffix(r.URL.Path, "/pending") {
		status = "pending_approval"
	}
	if status != "" {
		filtered := make([]portalstate.ActionRecord, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(item.Status, status) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(items),
		"items":  items,
	})
}
