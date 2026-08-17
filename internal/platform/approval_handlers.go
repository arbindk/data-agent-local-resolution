package platform

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) approvals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		executionID := r.URL.Query().Get("execution_id")
		limit := 100
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		items, err := s.store.ListApprovals(status, executionID, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"count":  len(items),
			"items":  items,
		})

	case http.MethodPost:
		var req ApprovalCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		approval, err := s.store.CreateApproval(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "created",
			"approval": approval,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
	}
}

func (s *Server) approvalByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "approval_id required"})
		return
	}

	approvalID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
			return
		}

		approval, ok := s.store.GetApproval(approvalID)
		if !ok {
			writeJSON(w, http.StatusNotFound, APIError{Error: "approval not found"})
			return
		}

		writeJSON(w, http.StatusOK, approval)
		return
	}

	switch parts[1] {
	case "decision":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
			return
		}

		var req ApprovalDecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		approval, err := s.store.DecideApproval(approvalID, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "updated",
			"approval": approval,
		})

	default:
		writeJSON(w, http.StatusNotFound, APIError{Error: "unknown approval endpoint"})
	}
}
