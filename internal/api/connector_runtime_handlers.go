package api

import (
	"net/http"
	"strings"

	connectorcatalog "everest.local/data-agent-policy-resolver/internal/connectors/catalog"
	connectorruntime "everest.local/data-agent-policy-resolver/internal/connectors/runtime"
)

func (s *Server) connectorRuntimeInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req connectorruntime.Request
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req.ConnectorID = strings.TrimSpace(req.ConnectorID)
	if req.ConnectorID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "connector_id is required"})
		return
	}

	manifest, ok := findConnectorManifest(req.ConnectorID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "connector manifest not found: " + req.ConnectorID})
		return
	}

	resp, err := connectorruntime.Invoke(r.Context(), manifest, req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.runtimeAudit.Add(runtimeAuditRecord(
		"connector",
		resp.ConnectorID,
		resp.Operation,
		resp.Status,
		resp.RuntimeKind,
		resp.ContractVersion,
		resp.DryRun,
		resp.SafeMode,
		resp.DurationMillis,
		resp.Error,
		resp.ExecutionControl,
	))

	writeJSON(w, http.StatusOK, resp)
}

func findConnectorManifest(connectorID string) (connectorcatalog.ConnectorManifest, bool) {
	items := connectorcatalog.LoadCatalog(connectorManifestDirs()...)
	for _, item := range items {
		if item.ConnectorID == connectorID {
			return item, true
		}
	}
	return connectorcatalog.ConnectorManifest{}, false
}
