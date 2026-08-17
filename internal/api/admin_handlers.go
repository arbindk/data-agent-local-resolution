// Package api — admin reset & import handlers.
// COPY THIS FILE TO: internal/api/admin_handlers.go
//
// Then register the two routes in server.go (in NewServer, next to the other
// mux.HandleFunc lines, e.g. near "/v1/demo/seed-duckdb"):
//
//     mux.HandleFunc("/v1/admin/reset", s.adminReset)
//     mux.HandleFunc("/v1/admin/import", s.adminImport)
//
// No other edits to existing files are required. The store methods used here
// (portalState.Reset, store.ResetRuntime, scanJobs.ResetAll, scanJobs.ImportJobs)
// are provided in the sibling patch files.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

// ---- helpers --------------------------------------------------------------

// adminAuthorized allows only privileged roles to run destructive admin ops.
func (s *Server) adminAuthorized(r *http.Request) bool {
	role := strings.ToLower(strings.TrimSpace(firstNonBlank(
		r.Header.Get("X-Everest-Role"), r.Header.Get("X-User-Role"))))
	switch role {
	case "admin", "security_officer", "data_steward":
		// data_steward kept for local/dev parity with the portal default.
		return true
	default:
		return false
	}
}

// ---- POST /v1/admin/reset -------------------------------------------------

func (s *Server) adminReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.adminAuthorized(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	var req struct {
		Scopes         []string `json:"scopes"`
		IncludeSecrets bool     `json:"include_secrets"`
		Confirm        string   `json:"confirm"`
	}
	_ = jsonDecode(r, &req)
	if req.Confirm != "RESET" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `confirm must equal "RESET"`})
		return
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"portal_state", "duckdb", "scan_jobs", "async_artifacts"}
	}
	cleared := []string{}
	skipped := []string{}

	for _, sc := range scopes {
		switch strings.ToLower(strings.TrimSpace(sc)) {
		case "portal_state":
			if s.portalState != nil {
				s.portalState.Reset()
				cleared = append(cleared, "portal_state")
			} else {
				skipped = append(skipped, "portal_state")
			}
		case "duckdb":
			if s.store != nil {
				if err := s.store.ResetRuntime(r.Context()); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "duckdb reset failed: " + err.Error(), "cleared": cleared})
					return
				}
				cleared = append(cleared, "duckdb")
			} else {
				skipped = append(skipped, "duckdb")
			}
		case "scan_jobs":
			if s.scanJobs != nil {
				s.scanJobs.ResetAll()
				cleared = append(cleared, "scan_jobs")
			} else {
				skipped = append(skipped, "scan_jobs")
			}
		case "async_artifacts":
			if p := strings.TrimSpace(s.cfg.AzureBlobAsyncResultsPath); p != "" {
				_ = os.RemoveAll(p)
				_ = os.MkdirAll(p, 0o755)
				cleared = append(cleared, "async_artifacts")
			} else {
				skipped = append(skipped, "async_artifacts")
			}
		default:
			skipped = append(skipped, sc)
		}
	}

	secretsNote := ""
	if req.IncludeSecrets {
		if p := strings.TrimSpace(s.cfg.SecretStoreFilePath); p != "" {
			_ = os.Remove(p)
			cleared = append(cleared, "secrets")
			secretsNote = "restart the agent to fully clear in-memory secrets"
		} else {
			skipped = append(skipped, "secrets")
		}
	}

	// The reset itself is provable: first post-reset audit row.
	if s.portalState != nil {
		s.recordAudit(portalstate.AuditEvent{
			EventType: "agent.admin.reset",
			Status:    "ok",
			Operation: "admin_reset",
			Summary:   "Agent runtime data reset via admin endpoint.",
			Details:   map[string]any{"cleared": cleared, "include_secrets": req.IncludeSecrets},
		})
	}

	resp := map[string]any{
		"status":   "ok",
		"cleared":  cleared,
		"skipped":  skipped,
		"reset_at": time.Now().UTC().Format(time.RFC3339),
	}
	if secretsNote != "" {
		resp["secrets_note"] = secretsNote
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---- POST /v1/admin/import ------------------------------------------------

// adminImportPayload mirrors the portal's Backup file (__dd_backup).
type adminImportPayload struct {
	Backup bool   `json:"__dd_backup"`
	Mode   string `json:"mode"` // "replace" (default) | "merge"

	Evidence itemsWrap `json:"evidence"`
	Audit    itemsWrap `json:"audit"`
	Pending  itemsWrap `json:"pending"`
	History  itemsWrap `json:"history"`
	Jobs     itemsWrap `json:"jobs"`
	Options  struct {
		AzureSources []portalstate.AzureSource    `json:"azure_sources"`
		Containers   []portalstate.AzureContainer `json:"containers"`
		Blobs        []portalstate.AzureBlob      `json:"blobs"`
	} `json:"options"`
}

// itemsWrap accepts either {"items":[...]} or a bare [...] array.
type itemsWrap struct {
	Items []json.RawMessage `json:"items"`
}

func (i *itemsWrap) UnmarshalJSON(b []byte) error {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '[' {
		return json.Unmarshal(b, &i.Items)
	}
	var tmp struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	i.Items = tmp.Items
	return nil
}

func decodeEach[T any](raws []json.RawMessage) []T {
	out := make([]T, 0, len(raws))
	for _, raw := range raws {
		var v T
		if err := json.Unmarshal(raw, &v); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func (s *Server) adminImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.adminAuthorized(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	if s.portalState == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "portal state store unavailable"})
		return
	}

	var p adminImportPayload
	if err := jsonDecode(r, &p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid backup: " + err.Error()})
		return
	}
	if !p.Backup {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a DataDynamics backup (missing __dd_backup)"})
		return
	}

	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	if mode == "" {
		mode = "replace"
	}
	if mode == "replace" {
		s.portalState.Reset()
		if s.scanJobs != nil {
			s.scanJobs.ResetAll()
		}
	}

	counts := map[string]int{}

	// Sources first (containers/blobs attach to a source).
	var primary portalstate.AzureSource
	for idx, src := range p.Options.AzureSources {
		s.portalState.UpsertAzureSource(src)
		if idx == 0 {
			primary = src
		}
		counts["sources"]++
	}
	if len(p.Options.Containers) > 0 {
		s.portalState.UpsertAzureContainers(primary, p.Options.Containers)
		counts["containers"] = len(p.Options.Containers)
	}
	if len(p.Options.Blobs) > 0 {
		s.portalState.UpsertAzureBlobs(primary, p.Options.Blobs)
		counts["blobs"] = len(p.Options.Blobs)
	}

	for _, ev := range decodeEach[portalstate.EvidenceRecord](p.Evidence.Items) {
		s.portalState.RecordEvidence(ev)
		counts["evidence"]++
	}
	for _, au := range decodeEach[portalstate.AuditEvent](p.Audit.Items) {
		s.portalState.RecordAudit(au)
		counts["audit"]++
	}
	for _, ac := range decodeEach[portalstate.ActionRecord](p.Pending.Items) {
		s.portalState.UpsertAction(ac)
		counts["actions"]++
	}
	for _, ac := range decodeEach[portalstate.ActionRecord](p.History.Items) {
		s.portalState.UpsertAction(ac)
		counts["actions"]++
	}

	if s.scanJobs != nil && len(p.Jobs.Items) > 0 {
		jobs := decodeEach[controlplane.AzureBlobScanJob](p.Jobs.Items)
		s.scanJobs.ImportJobs(jobs)
		counts["jobs"] = len(jobs)
	}

	s.recordAudit(portalstate.AuditEvent{
		EventType: "agent.admin.import",
		Status:    "ok",
		Operation: "admin_import",
		Summary:   "Agent data imported from a portal backup.",
		Details:   map[string]any{"mode": mode, "imported": counts},
	})

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "mode": mode, "imported": counts})
}
