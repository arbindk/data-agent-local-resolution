package platform

import (
	"encoding/json"
	"net/http"
)

func (s *Server) governanceContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, s.store.GovernanceContext())
}

func (s *Server) governanceEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req GovernanceEvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, s.store.EvaluateGovernance(req))
}
