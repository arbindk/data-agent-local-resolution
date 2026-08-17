package api

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

type enterprisePostureResponse struct {
	Status          string                   `json:"status"`
	GeneratedAt     string                   `json:"generated_at"`
	Metrics         enterprisePostureMetrics `json:"metrics"`
	RiskBySeverity  map[string]int           `json:"risk_by_severity"`
	JobStatus       map[string]int           `json:"job_status"`
	ActionStatus    map[string]int           `json:"action_status"`
	ApprovalStatus  map[string]int           `json:"approval_status"`
	Trends          []enterpriseTrendPoint   `json:"trends"`
	TopRisks        []enterpriseRiskItem     `json:"top_risks"`
	Recommendations []string                 `json:"recommendations"`
}

type enterprisePostureMetrics struct {
	Sources              int    `json:"sources"`
	SourcesWithoutAccess int    `json:"sources_without_access"`
	Containers           int    `json:"containers"`
	Objects              int    `json:"objects"`
	EvidenceRecords      int    `json:"evidence_records"`
	AuditEvents          int    `json:"audit_events"`
	Actions              int    `json:"actions"`
	PendingActions       int    `json:"pending_actions"`
	HighRiskEvidence     int    `json:"high_risk_evidence"`
	SensitiveFindings    int    `json:"sensitive_findings"`
	AsyncJobs            int    `json:"async_jobs"`
	AsyncActiveJobs      int    `json:"async_active_jobs"`
	AsyncRecords         int    `json:"async_records"`
	AsyncRetries         int    `json:"async_retries"`
	AsyncThrottles       int    `json:"async_throttles"`
	AsyncStaleWorkers    int    `json:"async_stale_workers"`
	ApprovalsPending     int    `json:"approvals_pending"`
	ApprovalsApproved    int    `json:"approvals_approved"`
	ApprovalsRejected    int    `json:"approvals_rejected"`
	RiskScore            int    `json:"risk_score"`
	RiskLevel            string `json:"risk_level"`
}

type enterpriseRiskItem struct {
	SubjectID   string `json:"subject_id"`
	EventType   string `json:"event_type"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	DataStoreID string `json:"data_store_id,omitempty"`
	Container   string `json:"container,omitempty"`
	Blob        string `json:"blob,omitempty"`
	Summary     string `json:"summary"`
	CreatedAt   string `json:"created_at"`
}

type enterpriseTrendPoint struct {
	Date              string `json:"date"`
	EvidenceRecords   int    `json:"evidence_records"`
	HighRiskEvidence  int    `json:"high_risk_evidence"`
	SensitiveFindings int    `json:"sensitive_findings"`
	Actions           int    `json:"actions"`
	AsyncRecords      int    `json:"async_records"`
}

func (s *Server) enterprisePosture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	resp := enterprisePostureResponse{
		Status:         "ok",
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		RiskBySeverity: map[string]int{},
		JobStatus:      map[string]int{},
		ActionStatus:   map[string]int{},
		ApprovalStatus: map[string]int{},
		Trends:         []enterpriseTrendPoint{},
		TopRisks:       []enterpriseRiskItem{},
		Recommendations: []string{
			"Add or select an Azure Blob source.",
			"Discover containers and choose a governed scan scope.",
			"Run DSPM or async scan to populate sensitive-data posture.",
		},
	}
	trendByDay := map[string]*enterpriseTrendPoint{}

	if s.portalState != nil {
		options := s.portalState.Options()
		resp.Metrics.Sources = len(options.AzureSources)
		resp.Metrics.Containers = len(options.Containers)
		resp.Metrics.Objects = len(options.Blobs)
		resp.Metrics.EvidenceRecords = len(options.EvidenceRecords)
		resp.Metrics.AuditEvents = len(options.AuditEvents)
		resp.Metrics.Actions = len(options.ActionRecords)
		for _, source := range options.AzureSources {
			if !source.CredentialSet {
				resp.Metrics.SourcesWithoutAccess++
			}
		}

		for _, evidence := range options.EvidenceRecords {
			trend := postureTrendForDay(trendByDay, evidence.CreatedAt)
			trend.EvidenceRecords++
			severity := normalizePostureValue(firstNonBlank(evidence.Severity, evidence.Status, "info"))
			resp.RiskBySeverity[severity]++
			if severity == "high" || severity == "critical" || evidenceLooksSensitive(evidence.EventType, evidence.Summary, evidence.Details) {
				resp.Metrics.SensitiveFindings++
				trend.SensitiveFindings++
			}
			if severity == "high" || severity == "critical" {
				resp.Metrics.HighRiskEvidence++
				trend.HighRiskEvidence++
				resp.TopRisks = append(resp.TopRisks, enterpriseRiskItem{
					SubjectID:   evidence.SubjectID,
					EventType:   evidence.EventType,
					Severity:    severity,
					Status:      evidence.Status,
					DataStoreID: evidence.DataStoreID,
					Container:   evidence.Container,
					Blob:        evidence.Blob,
					Summary:     evidence.Summary,
					CreatedAt:   evidence.CreatedAt,
				})
			}
		}

		for _, action := range options.ActionRecords {
			postureTrendForDay(trendByDay, firstNonBlank(action.UpdatedAt, action.CreatedAt)).Actions++
			status := normalizePostureValue(firstNonBlank(action.Status, "pending"))
			resp.ActionStatus[status]++
			if status == "pending" || status == "draft" || status == "blocked" {
				resp.Metrics.PendingActions++
			}
			approval := normalizePostureValue(firstNonBlank(action.ApprovalEnforcement, action.GuardianDecision))
			if strings.Contains(approval, "pending") || strings.Contains(approval, "required") {
				resp.Metrics.ApprovalsPending++
				resp.ApprovalStatus["pending"]++
			}
			if strings.Contains(approval, "approved") || approval == "allow" {
				resp.Metrics.ApprovalsApproved++
				resp.ApprovalStatus["approved"]++
			}
			if strings.Contains(approval, "reject") || strings.Contains(approval, "block") {
				resp.Metrics.ApprovalsRejected++
				resp.ApprovalStatus["rejected"]++
			}
		}
	}

	if s.scanJobs != nil {
		resp.Metrics.AsyncStaleWorkers = len(s.scanJobs.StaleLeasedJobs(time.Now().UTC()))
		for _, job := range s.scanJobs.List() {
			postureTrendForDay(trendByDay, firstNonBlank(job.UpdatedAt, job.CreatedAt)).AsyncRecords += job.RecordsWritten
			status := normalizePostureValue(firstNonBlank(job.Status, "unknown"))
			resp.JobStatus[status]++
			resp.Metrics.AsyncJobs++
			resp.Metrics.AsyncRecords += job.RecordsWritten
			resp.Metrics.AsyncRetries += job.RetryCount
			resp.Metrics.AsyncThrottles += job.ThrottleCount
			if status == "queued" || status == "running" || status == "paused" {
				resp.Metrics.AsyncActiveJobs++
			}
		}
	}

	sort.Slice(resp.TopRisks, func(i, j int) bool {
		return postureSeverityRank(resp.TopRisks[i].Severity) > postureSeverityRank(resp.TopRisks[j].Severity)
	})
	if len(resp.TopRisks) > 10 {
		resp.TopRisks = resp.TopRisks[:10]
	}
	resp.Trends = enterprisePostureTrends(trendByDay)
	resp.Metrics.RiskScore, resp.Metrics.RiskLevel = enterpriseRiskScore(resp.Metrics)
	resp.Recommendations = enterprisePostureRecommendations(resp)

	writeJSON(w, http.StatusOK, resp)
}

func enterpriseRiskScore(m enterprisePostureMetrics) (int, string) {
	score := 0
	score += m.HighRiskEvidence * 12
	score += m.SensitiveFindings * 5
	score += m.PendingActions * 4
	score += m.ApprovalsPending * 4
	score += m.AsyncActiveJobs * 2
	score += m.AsyncThrottles * 2
	if m.Objects > 1000 {
		score += 5
	}
	if score > 100 {
		score = 100
	}
	level := "low"
	switch {
	case score >= 80:
		level = "critical"
	case score >= 60:
		level = "high"
	case score >= 30:
		level = "medium"
	}
	return score, level
}

func enterprisePostureRecommendations(resp enterprisePostureResponse) []string {
	recommendations := []string{}
	if resp.Metrics.Sources == 0 {
		recommendations = append(recommendations, "Add an Azure Blob source with governed account access.")
	}
	if resp.Metrics.SourcesWithoutAccess > 0 {
		recommendations = append(recommendations, "Review saved sources without configured account access.")
	}
	if resp.Metrics.Sources > 0 && resp.Metrics.Containers == 0 {
		recommendations = append(recommendations, "Discover containers for the selected source.")
	}
	if resp.Metrics.Objects == 0 {
		recommendations = append(recommendations, "Run an async scan or list blobs to populate object posture.")
	}
	if resp.Metrics.HighRiskEvidence > 0 {
		recommendations = append(recommendations, "Review high-risk evidence and route restricted actions through HITL.")
	}
	if resp.Metrics.PendingActions > 0 || resp.Metrics.ApprovalsPending > 0 {
		recommendations = append(recommendations, "Open Action Center and resolve pending approvals or blocked actions.")
	}
	if resp.Metrics.AsyncActiveJobs > 0 {
		recommendations = append(recommendations, "Monitor async scan checkpoints until jobs complete or require intervention.")
	}
	if resp.Metrics.AsyncStaleWorkers > 0 {
		recommendations = append(recommendations, "Resume stale async scan jobs so another worker can acquire the expired lease.")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Enterprise posture is current; continue monitoring dashboard trends and evidence health.")
	}
	return recommendations
}

func postureTrendForDay(trends map[string]*enterpriseTrendPoint, timestamp string) *enterpriseTrendPoint {
	day := postureDay(timestamp)
	if trends[day] == nil {
		trends[day] = &enterpriseTrendPoint{Date: day}
	}
	return trends[day]
}

func enterprisePostureTrends(trends map[string]*enterpriseTrendPoint) []enterpriseTrendPoint {
	keys := make([]string, 0, len(trends))
	for key := range trends {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 14 {
		keys = keys[len(keys)-14:]
	}
	out := make([]enterpriseTrendPoint, 0, len(keys))
	for _, key := range keys {
		if trends[key] != nil {
			out = append(out, *trends[key])
		}
	}
	return out
}

func postureDay(timestamp string) string {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return time.Now().UTC().Format("2006-01-02")
	}
	if len(timestamp) >= 10 {
		return timestamp[:10]
	}
	return time.Now().UTC().Format("2006-01-02")
}

func evidenceLooksSensitive(eventType string, summary string, details map[string]any) bool {
	text := strings.ToLower(eventType + " " + summary)
	for key, value := range details {
		text += " " + strings.ToLower(key) + " " + strings.ToLower(anyToPostureString(value))
	}
	for _, needle := range []string{"pii", "phi", "pci", "secret", "token", "password", "credential", "sensitive", "restricted", "ransomware", "legal"} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func anyToPostureString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, " ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, anyToPostureString(item))
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}

func normalizePostureValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return strings.ReplaceAll(value, " ", "_")
}

func postureSeverityRank(severity string) int {
	switch normalizePostureValue(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
