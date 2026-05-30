package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

type Server struct {
	resolver *policyresolver.Resolver
}

func NewServer(resolver *policyresolver.Resolver) *Server {
	return &Server{resolver: resolver}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/v1/policy/resolve", s.resolvePolicy)
	mux.HandleFunc("/v1/policy/active/", s.getActivePolicy)
	mux.HandleFunc("/v1/policy/simulate/git-unavailable", s.gitUnavailable)
	mux.HandleFunc("/v1/policy/simulate/git-available", s.gitAvailable)

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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
