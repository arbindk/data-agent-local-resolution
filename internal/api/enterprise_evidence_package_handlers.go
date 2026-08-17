package api

import (
	"net/http"
	"time"

	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) enterpriseEvidencePackage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.authorizeEnterpriseAction(w, r, "enterprise.evidence_package.generate", "platform_admin", "security_officer", "compliance_officer", "auditor") {
		return
	}

	actor := s.enterpriseActor(r)
	packageID := "evpkg-" + time.Now().UTC().Format("20060102-150405")
	payload := map[string]any{
		"package_id":   packageID,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"generated_by": actor.Actor,
		"role":         actor.Role,
		"tenant":       actor.Tenant,
		"source":       "everest_enterprise",
		"scope":        "enterprise_evidence_package",
	}

	if s.portalState != nil {
		options := s.portalState.Options()
		payload["sources"] = options.AzureSources
		payload["containers"] = options.Containers
		payload["objects"] = options.Blobs
		payload["evidence_records"] = options.EvidenceRecords
		payload["audit_events"] = options.AuditEvents
		payload["actions"] = options.ActionRecords
		payload["jobs"] = options.Jobs
		payload["policies"] = options.Policies
		payload["ai_providers"] = options.AIProviders
	}
	if s.scanJobs != nil {
		payload["async_scan_jobs"] = s.scanJobs.List()
	}
	payload["notice"] = "Evidence package excludes raw secret values and does not include unrestricted raw object content."

	s.recordAuditEventForEvidencePackage(packageID, actor)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) recordAuditEventForEvidencePackage(packageID string, actor enterpriseActor) {
	s.recordAudit(portalstate.AuditEvent{
		EventType:  "enterprise.evidence_package.generated",
		Actor:      actor.Actor,
		Role:       actor.Role,
		Status:     "generated",
		TargetType: "evidence_package",
		TargetID:   packageID,
		Operation:  "export_evidence_package",
		Summary:    "Enterprise evidence package was generated.",
		Details: map[string]any{
			"tenant": actor.Tenant,
		},
	})
}
