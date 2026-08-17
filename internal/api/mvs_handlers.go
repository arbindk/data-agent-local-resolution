package api

import (
	"net/http"
)

func (s *Server) localStoreTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tables, err := s.store.ListTables(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"store":  "local_execution_store_ephemeral",
		"tables": tables,
	})
}

func (s *Server) normalizedFilesView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	batchID := r.URL.Query().Get("batch_id")
	if batchID == "" {
		batchID = s.azureBlobConfig().BatchID
	}

	limit := queryInt(r, "limit", 25)
	offset := queryInt(r, "offset", 0)

	items, total, err := s.store.ListNormalizedFiles(r.Context(), batchID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"batch_id": batchID,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"items":    items,
	})
}

func (s *Server) quickScanAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	batchID := r.URL.Query().Get("batch_id")
	if batchID == "" {
		batchID = s.azureBlobConfig().BatchID
	}

	data, err := s.store.QuickScanAnalytics(r.Context(), batchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) actionPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	executionID := r.URL.Query().Get("execution_id")
	if executionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id is required"})
		return
	}

	limit := queryInt(r, "limit", 25)
	offset := queryInt(r, "offset", 0)

	items, total, err := s.store.ListActionPlan(r.Context(), executionID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"execution_id": executionID,
		"mode":         "dry_run",
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"items":        items,
	})
}

func (s *Server) guardianDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	executionID := r.URL.Query().Get("execution_id")
	if executionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id is required"})
		return
	}

	limit := queryInt(r, "limit", 25)
	offset := queryInt(r, "offset", 0)

	items, total, err := s.store.ListGuardianDecisions(r.Context(), executionID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"execution_id": executionID,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"items":        items,
	})
}

func (s *Server) currentLeaseView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cfg := s.azureBlobConfig()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "backend_owned",
		"data_store_id":  cfg.DataStoreID,
		"batch_id":       cfg.BatchID,
		"source_system":  "azure_blob",
		"container":      cfg.ContainerName,
		"prefix":         cfg.Prefix,
		"action_mode":    cfg.ActionMode,
		"writeback":      cfg.WritebackEnabled,
		"lease_model":    "In enterprise mode this comes from WorkloadControl. In this Go POC, backend creates the lease from current Azure source + active policy defaults.",
		"recovery_model": "Local store is ephemeral. Recovery source is central lease/checkpoint/source continuation marker.",
	})
}
