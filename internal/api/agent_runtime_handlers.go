package api

import (
	"encoding/json"
	"net/http"

	"everest.local/data-agent-policy-resolver/internal/agentruntime"
)

func (s *Server) agentRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	payload := map[string]any{}
	if data, err := json.Marshal(agentruntime.CurrentStatus()); err == nil {
		_ = json.Unmarshal(data, &payload)
	}
	payload["secret_provider"] = s.secrets.Status()
	payload["authorization"] = map[string]any{
		"mode": firstNonBlank(s.cfg.AuthorizationMode, "audit"),
		"headers": []string{
			"X-Everest-Actor",
			"X-Everest-Role",
			"X-Everest-Tenant",
		},
	}
	if s.searchIndex != nil {
		payload["search_index"] = s.searchIndex.Status()
		payload["elasticsearch_index"] = s.searchIndex.Status()
	}
	writeJSON(w, http.StatusOK, payload)
}
