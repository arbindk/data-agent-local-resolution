package api

import "net/http"

type enterprisePlaybook struct {
	PlaybookID       string   `json:"playbook_id"`
	Title            string   `json:"title"`
	Scenario         string   `json:"scenario"`
	Priority         string   `json:"priority"`
	RecommendedRoles []string `json:"recommended_roles"`
	Actions          []string `json:"actions"`
	EvidenceRequired []string `json:"evidence_required"`
	HITLRequired     bool     `json:"hitl_required"`
}

func (s *Server) enterprisePlaybooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	playbooks := []enterprisePlaybook{
		{
			PlaybookID:       "pii-public-exposure",
			Title:            "PII Exposure Response",
			Scenario:         "Sensitive personal data appears in a public or broad-access container.",
			Priority:         "critical",
			RecommendedRoles: []string{"security_officer", "data_steward", "compliance_officer"},
			Actions:          []string{"run_content_scan", "recommend_tags", "preview_quarantine", "submit_hitl", "export_evidence"},
			EvidenceRequired: []string{"source", "container", "object", "entity_findings", "policy_decision", "approval", "action_receipt"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "secret-detected",
			Title:            "Secret Or Token Detection",
			Scenario:         "API keys, tokens, passwords, or credentials are detected in object content or metadata.",
			Priority:         "critical",
			RecommendedRoles: []string{"security_officer", "platform_admin"},
			Actions:          []string{"run_tiered_scan", "tag_restricted", "preview_quarantine", "open_security_review", "export_evidence"},
			EvidenceRequired: []string{"entity_findings", "object_reference", "risk_score", "guardian_decision", "approval"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "phi-misplaced",
			Title:            "PHI Misplacement Review",
			Scenario:         "Healthcare-sensitive records are found outside approved PHI storage scope.",
			Priority:         "high",
			RecommendedRoles: []string{"compliance_officer", "data_steward"},
			Actions:          []string{"classify_phi", "complete_metadata", "preview_move_prep", "submit_hitl"},
			EvidenceRequired: []string{"classification", "policy_context", "source_scope", "target_scope", "approval"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "cross-border-transfer",
			Title:            "Cross-Border Transfer Control",
			Scenario:         "Copy or move request crosses source and target data residency boundaries.",
			Priority:         "high",
			RecommendedRoles: []string{"compliance_officer", "contract_owner", "security_officer"},
			Actions:          []string{"preview_copy_or_move", "evaluate_residency_policy", "submit_hitl", "block_if_unapproved"},
			EvidenceRequired: []string{"source_region", "target_region", "policy_decision", "approval", "action_receipt"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "legal-hold-tier-change",
			Title:            "Legal Hold Tier Change Review",
			Scenario:         "Archive, rehydrate, or tier action is requested for legal or retention-controlled data.",
			Priority:         "high",
			RecommendedRoles: []string{"compliance_officer", "records_owner"},
			Actions:          []string{"review_retention_metadata", "preview_tier_change", "submit_hitl"},
			EvidenceRequired: []string{"retention_metadata", "tier_change_preview", "approval", "action_receipt"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "ransomware-indicator",
			Title:            "Ransomware Indicator Containment",
			Scenario:         "Suspicious names, extensions, or ransom-note-like patterns appear in storage.",
			Priority:         "critical",
			RecommendedRoles: []string{"security_officer", "incident_responder"},
			Actions:          []string{"run_targeted_scan", "tag_security_hold", "preview_quarantine", "open_incident_review"},
			EvidenceRequired: []string{"indicator_pattern", "affected_scope", "recommendation", "guardian_decision"},
			HITLRequired:     true,
		},
		{
			PlaybookID:       "async-large-scan",
			Title:            "Large Account Scan Operations",
			Scenario:         "Large Azure Blob account requires checkpointed scan, pause, resume, and evidence monitoring.",
			Priority:         "medium",
			RecommendedRoles: []string{"data_analyst", "platform_admin"},
			Actions:          []string{"queue_async_scan", "monitor_checkpoint", "resume_or_pause", "review_normalized_records"},
			EvidenceRequired: []string{"job_id", "checkpoint", "records_written", "normalized_artifact"},
			HITLRequired:     false,
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"playbooks": playbooks,
	})
}
