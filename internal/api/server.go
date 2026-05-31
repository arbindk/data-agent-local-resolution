package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

type Server struct {
	resolver     *policyresolver.Resolver
	orchestrator *execution.Orchestrator
}

func NewServer(resolver *policyresolver.Resolver, orchestrator *execution.Orchestrator) *Server {
	return &Server{
		resolver:     resolver,
		orchestrator: orchestrator,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.health)

	mux.HandleFunc("/v1/policy/resolve", s.resolvePolicy)
	mux.HandleFunc("/v1/policy/active/", s.getActivePolicy)
	mux.HandleFunc("/v1/policy/simulate/git-unavailable", s.gitUnavailable)
	mux.HandleFunc("/v1/policy/simulate/git-available", s.gitAvailable)

	mux.HandleFunc("/v1/execution/start", s.startExecution)
	mux.HandleFunc("/v1/execution/", s.executionByID)
	mux.HandleFunc("/v1/cache/status", s.cacheStatus)

	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "data-agent-local-resolution",
	})
}

func (s *Server) resolvePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var lease policyresolver.PolicyLease
	if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.resolver.ResolveAndActivatePolicy(r.Context(), lease)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getActivePolicy(w http.ResponseWriter, r *http.Request) {
	dataStoreID := strings.TrimPrefix(r.URL.Path, "/v1/policy/active/")
	if dataStoreID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data_store_id required"})
		return
	}

	ctx, ok := s.resolver.GetActivePolicy(dataStoreID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active policy not found"})
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

func (s *Server) gitUnavailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.resolver.SetGitUnavailable(true)
	writeJSON(w, http.StatusOK, map[string]bool{"git_unavailable": true})
}

func (s *Server) gitAvailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.resolver.SetGitUnavailable(false)
	writeJSON(w, http.StatusOK, map[string]bool{"git_unavailable": false})
}

func (s *Server) startExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req execution.ExecutionStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.orchestrator.Start(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) executionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/execution/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id required"})
		return
	}

	executionID := parts[0]

	if len(parts) == 1 {
		status, ok := s.orchestrator.GetStatus(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "execution not found"})
			return
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	switch parts[1] {
	case "summary":
		summary, ok := s.orchestrator.GetSummary(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "summary not found"})
			return
		}
		writeJSON(w, http.StatusOK, summary)

	case "provenance":
		prov, ok := s.orchestrator.GetProvenance(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provenance not found"})
			return
		}
		writeJSON(w, http.StatusOK, prov)

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown execution endpoint"})
	}
}

func (s *Server) cacheStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                 "available",
		"policy_cache":           "in_memory_and_disk_backed",
		"workflow_cache":         "demo_resolved",
		"last_known_good_policy": "enabled",
		"note":                   "hackathon demo cache status endpoint",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
