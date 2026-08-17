package platform

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/v1/agents/register", s.registerAgent)
	mux.HandleFunc("/v1/agents", s.listAgents)
	mux.HandleFunc("/v1/agents/", s.agentByID)

	mux.HandleFunc("/v1/datasources", s.dataSources)
	mux.HandleFunc("/v1/policies", s.policies)
	mux.HandleFunc("/v1/governance/context", s.governanceContext)
	mux.HandleFunc("/v1/governance/evaluate", s.governanceEvaluate)
	mux.HandleFunc("/v1/jobs", s.jobs)
	mux.HandleFunc("/v1/dashboard/summary", s.dashboardSummary)
	mux.HandleFunc("/v1/approvals", s.approvals)
	mux.HandleFunc("/v1/approvals/", s.approvalByID)

	mux.HandleFunc("/v1/onboarding/azureblob/quickstart", s.azureBlobQuickstart)

	mux.HandleFunc("/v1/admin/storage/status", s.storageStatus)
	mux.HandleFunc("/v1/admin/storage/backup", s.storageBackup)
	mux.HandleFunc("/v1/admin/storage/reset", s.storageReset)

	mux.HandleFunc("/v1/leases/", s.leaseByID)
	mux.HandleFunc("/v1/executions/", s.executionEvidenceByID)
	mux.HandleFunc("/v1/manifests/summary", s.manifestSummary)
	mux.HandleFunc("/v1/manifests/details", s.manifestDetails)
	mux.HandleFunc("/v1/provenance", s.provenance)
	mux.HandleFunc("/v1/actions/results", s.actionResults)

	mux.HandleFunc("/v1/audit/events", s.auditEvents)
	mux.HandleFunc("/v1/exports/azureblob/", s.azureBlobExport)

	return withCORS(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "everest-platform-api",
		Version: "mvs1-control-plane-skeleton",
	})
}

func (s *Server) registerAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req AgentRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	agent, err := s.store.RegisterAgent(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"registered": true,
		"agent":      agent,
	})
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	agents := s.store.ListAgents()

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(agents),
		"items":  agents,
	})
}

func (s *Server) agentByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/agents/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "agent_id required"})
		return
	}

	agentID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
			return
		}

		agent, ok := s.store.GetAgent(agentID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "agent not found"})
			return
		}

		writeJSON(w, http.StatusOK, agent)
		return
	}

	switch parts[1] {
	case "leases":
		if len(parts) == 3 && parts[2] == "next" {
			s.nextLease(w, r, agentID)
			return
		}

	default:
		writeJSON(w, http.StatusNotFound, APIError{Error: "unknown agent endpoint"})
	}
}

func (s *Server) nextLease(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	lease, ok := s.store.NextLease(agentID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "empty",
			"message": "no pending lease for agent",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "assigned",
		"lease":  lease,
	})
}

func (s *Server) dataSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := s.store.ListDataSources()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})

	case http.MethodPost:
		var req DataSourceCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		ds, err := s.store.SaveDataSource(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "saved",
			"data_source": ds,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) policies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	items := s.store.ListPolicies()

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(items),
		"items":  items,
	})
}

func (s *Server) jobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := s.store.ListJobs()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})

	case http.MethodPost:
		var req JobCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		job, leases, err := s.store.CreateJob(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "created",
			"job":    job,
			"leases": leases,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) leaseByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/leases/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "lease endpoint required"})
		return
	}

	leaseID := parts[0]

	switch parts[1] {
	case "heartbeat":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
			return
		}

		var req LeaseHeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		lease, err := s.store.HeartbeatLease(leaseID, req)
		if err != nil {
			writeJSON(w, http.StatusNotFound, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
			"lease":  lease,
		})

	default:
		writeJSON(w, http.StatusNotFound, APIError{Error: "unknown lease endpoint"})
	}
}

func (s *Server) azureBlobQuickstart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req AzureBlobQuickstartRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	result, err := s.store.ProvisionAzureBlobQuickstart(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) storageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	status, err := s.store.StorageStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) storageBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	result, err := s.store.BackupStorage()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) storageReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req StorageResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	result, err := s.store.ResetStorage(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	summary, err := s.store.DashboardSummary()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		switch origin {
		case "http://localhost:8080", "http://127.0.0.1:8080":
			w.Header().Set("Access-Control-Allow-Origin", origin)
		default:
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		}

		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) auditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := s.store.ListAuditEvents(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(events),
		"items":  events,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) executionEvidenceByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/executions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "execution endpoint required"})
		return
	}

	executionID := parts[0]

	switch parts[1] {
	case "status":
		if r.Method == http.MethodPost {
			var payload any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
				return
			}

			if err := s.store.SaveExecutionStatus(executionID, payload); err != nil {
				writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"status":       "saved",
				"execution_id": executionID,
			})
			return
		}

		if r.Method == http.MethodGet {
			value, ok := s.store.GetExecutionStatus(executionID)
			if !ok {
				writeJSON(w, http.StatusNotFound, APIError{Error: "execution status not found"})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"status":       "ok",
				"execution_id": executionID,
				"data":         value,
			})
			return
		}

		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})

	default:
		writeJSON(w, http.StatusNotFound, APIError{Error: "unknown execution evidence endpoint"})
	}
}

func (s *Server) manifestSummary(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		executionID := stringValue(payload, "execution_id")
		if executionID == "" {
			writeJSON(w, http.StatusBadRequest, APIError{Error: "execution_id is required"})
			return
		}

		if err := s.store.SaveManifestSummary(executionID, payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "saved",
			"execution_id": executionID,
		})

	case http.MethodGet:
		executionID := r.URL.Query().Get("execution_id")
		value, ok := s.store.GetManifestSummary(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "manifest summary not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"execution_id": executionID,
			"data":         value,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) manifestDetails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		executionID := stringValue(payload, "execution_id")
		if executionID == "" {
			writeJSON(w, http.StatusBadRequest, APIError{Error: "execution_id is required"})
			return
		}

		items := anySlice(payload["items"])

		if err := s.store.SaveManifestDetails(executionID, items); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "saved",
			"execution_id": executionID,
			"count":        len(items),
		})

	case http.MethodGet:
		executionID := r.URL.Query().Get("execution_id")
		items, ok := s.store.GetManifestDetails(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "manifest details not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"execution_id": executionID,
			"count":        len(items),
			"items":        items,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) provenance(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		executionID := stringValue(payload, "execution_id")
		if executionID == "" {
			writeJSON(w, http.StatusBadRequest, APIError{Error: "execution_id is required"})
			return
		}

		items := anySlice(payload["items"])

		if err := s.store.SaveProvenance(executionID, items); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "saved",
			"execution_id": executionID,
			"count":        len(items),
		})

	case http.MethodGet:
		executionID := r.URL.Query().Get("execution_id")
		items, ok := s.store.GetProvenance(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "provenance not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"execution_id": executionID,
			"count":        len(items),
			"items":        items,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) actionResults(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		executionID := stringValue(payload, "execution_id")
		if executionID == "" {
			writeJSON(w, http.StatusBadRequest, APIError{Error: "execution_id is required"})
			return
		}

		items := anySlice(payload["items"])

		if err := s.store.SaveActionResults(executionID, items); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "saved",
			"execution_id": executionID,
			"count":        len(items),
		})

	case http.MethodGet:
		executionID := r.URL.Query().Get("execution_id")
		items, ok := s.store.GetActionResults(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "action results not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"execution_id": executionID,
			"count":        len(items),
			"items":        items,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func stringValue(m map[string]any, key string) string {
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}

	s, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func anySlice(value any) []any {
	if value == nil {
		return []any{}
	}

	if wrapped, ok := value.(map[string]any); ok {
		if items, ok := wrapped["items"]; ok {
			return anySlice(items)
		}
	}

	items, ok := value.([]any)
	if ok {
		return items
	}

	return []any{value}
}
