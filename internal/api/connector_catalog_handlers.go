package api

import (
	"net/http"
	"os"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/connectors/catalog"
)

func (s *Server) connectorCatalog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := catalog.LoadCatalog(connectorManifestDirs()...)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": "connector_manifest_v1",
			"count":   len(items),
			"items":   items,
			"deployment_modes": []string{
				"cloud_connected",
				"private_network",
				"air_gapped",
				"hybrid",
			},
			"custom_connector_contract": map[string]any{
				"manifest_schema":       "contracts/connector_manifest.schema.json",
				"normalized_contract":   "contracts/normalized_batch.schema.json",
				"action_contract":       "contracts/connector_action.schema.json",
				"runtime_request":       "contracts/connector_runtime_request.schema.json",
				"runtime_response":      "contracts/connector_runtime_response.schema.json",
				"execution_model":       "external_process_or_http",
				"guardian_before_apply": true,
				"ai_autonomous_apply":   false,
			},
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) validateConnectorManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var manifest catalog.ConnectorManifest
	if err := jsonDecode(r, &manifest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := catalog.ValidateManifest(manifest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "invalid",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "valid",
		"connector_id": manifest.ConnectorID,
		"connector":    manifest,
		"next_steps": []string{
			"Package runtime in customer environment.",
			"Expose health/discover/scan/action endpoints or external-process commands.",
			"Emit normalized batch records using the Everest normalized contract.",
			"Route write actions through Guardian and HITL before apply.",
		},
	})
}

func connectorManifestDirs() []string {
	dirs := []string{
		"connectors",
		"demo/connectors",
	}

	if raw := strings.TrimSpace(os.Getenv("EVEREST_CONNECTOR_MANIFEST_DIRS")); raw != "" {
		for _, part := range strings.Split(raw, ";") {
			part = strings.TrimSpace(part)
			if part != "" {
				dirs = append(dirs, part)
			}
		}
	}

	return dirs
}
