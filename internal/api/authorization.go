// COPY THIS FILE TO: internal/api/authorization.go  (REPLACES the existing file)
//
// Only one thing changed vs. the original: enterpriseActor() now derives the
// trusted actor/role/tenant from the validated bearer SESSION (see
// auth_handlers.go: sessionActor) when one is present. The old X-Everest-Role
// header is now only a fallback for unauthenticated callers (the agent-runtime
// lease poll, curl/dev). A logged-in portal user can no longer self-assert a
// higher role by editing a header — the role is whatever the server issued at
// login. All RBAC decision logic below is unchanged.
package api

import (
	"net/http"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type enterpriseActor struct {
	Actor  string `json:"actor"`
	Role   string `json:"role"`
	Tenant string `json:"tenant,omitempty"`
}

func (s *Server) enterpriseActor(r *http.Request) enterpriseActor {
	// Trusted path: a valid server-issued session wins over any client header.
	if sess, ok := s.sessionActor(r); ok {
		if sess.Tenant == "" {
			sess.Tenant = "tenant-local"
		}
		return sess
	}
	// Fallback: header-asserted identity for unauthenticated callers
	// (agent runtime poll, diagnostics, dev). RBAC still gates what they can do.
	return enterpriseActor{
		Actor:  firstNonBlank(r.Header.Get("X-Everest-Actor"), r.Header.Get("X-User"), "portal-user"),
		Role:   strings.ToLower(firstNonBlank(r.Header.Get("X-Everest-Role"), r.Header.Get("X-Role"), "data_steward")),
		Tenant: firstNonBlank(r.Header.Get("X-Everest-Tenant"), r.Header.Get("X-Tenant"), "tenant-local"),
	}
}

func (s *Server) authorizeEnterpriseAction(w http.ResponseWriter, r *http.Request, operation string, allowedRoles ...string) bool {
	actor := s.enterpriseActor(r)
	allowed := roleAllowed(actor.Role, allowedRoles...)
	mode := strings.ToLower(strings.TrimSpace(s.cfg.AuthorizationMode))
	if mode == "" {
		mode = "audit"
	}

	if allowed {
		return true
	}

	status := "authorization_audit"
	summary := "Enterprise authorization would block this operation in enforce mode."
	if mode == "enforce" {
		status = "blocked"
		summary = "Enterprise authorization blocked this operation."
	}
	s.recordAudit(portalstate.AuditEvent{
		EventType:  "enterprise.authorization." + status,
		Actor:      actor.Actor,
		Role:       actor.Role,
		Status:     status,
		TargetType: "enterprise_operation",
		TargetID:   operation,
		Operation:  operation,
		Summary:    summary,
		Details: map[string]any{
			"tenant":        actor.Tenant,
			"allowed_roles": allowedRoles,
			"auth_mode":     mode,
		},
	})

	if mode == "enforce" {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":         "operation is not authorized for role " + actor.Role,
			"operation":     operation,
			"required_role": allowedRoles,
		})
		return false
	}
	return true
}

func (s *Server) authorizeWorkbenchOperation(w http.ResponseWriter, r *http.Request, operation string) bool {
	operation = normalizeWorkbenchOperation(operation)
	if !workbenchOperationCreatesAction(operation) {
		return true
	}
	switch operation {
	case "delete_blob", "delete_container", "move_blob", "set_tier", "rehydrate_blob":
		return s.authorizeEnterpriseAction(w, r, "azureblob.workbench."+operation, "platform_admin", "security_officer", "compliance_officer")
	default:
		return s.authorizeEnterpriseAction(w, r, "azureblob.workbench."+operation, "platform_admin", "security_officer", "data_steward")
	}
}

func (s *Server) authorizeAsyncScanAction(w http.ResponseWriter, r *http.Request, action string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "resume", "pause":
		return s.authorizeEnterpriseAction(w, r, "azureblob.async_scan."+action, "platform_admin", "security_officer", "data_steward", "data_analyst")
	case "cancel", "complete", "fail":
		return s.authorizeEnterpriseAction(w, r, "azureblob.async_scan."+action, "platform_admin", "security_officer")
	case "checkpoint":
		return s.authorizeEnterpriseAction(w, r, "azureblob.async_scan.checkpoint", "platform_admin", "agent_runtime")
	default:
		return s.authorizeEnterpriseAction(w, r, "azureblob.async_scan."+action, "platform_admin")
	}
}

func roleAllowed(role string, allowedRoles ...string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return false
	}
	for _, allowed := range allowedRoles {
		if role == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}
func (s *Server) authorizeEnterpriseApplyAction(w http.ResponseWriter, r *http.Request, operation string, allowedRoles ...string) bool {
	if ok, reason := s.checkEnterpriseApplyAction(r, operation, allowedRoles...); !ok {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":         reason,
			"operation":     operation,
			"required_role": allowedRoles,
			"auth_mode":     "apply_enforced",
		})
		return false
	}
	return true
}

func (s *Server) checkEnterpriseApplyAction(r *http.Request, operation string, allowedRoles ...string) (bool, string) {
	actor := s.enterpriseActor(r)
	if roleAllowed(actor.Role, allowedRoles...) {
		return true, ""
	}

	reason := "apply operation is not authorized for role " + actor.Role
	s.recordAudit(portalstate.AuditEvent{
		EventType:  "enterprise.authorization.blocked",
		Actor:      actor.Actor,
		Role:       actor.Role,
		Status:     "blocked",
		TargetType: "enterprise_apply_operation",
		TargetID:   operation,
		Operation:  operation,
		Summary:    "Enterprise authorization blocked apply-mode mutation.",
		Details: map[string]any{
			"tenant":        actor.Tenant,
			"allowed_roles": allowedRoles,
			"auth_mode":     "apply_enforced",
		},
	})
	return false, reason
}

func azureBlobApplyRoles(operation string) []string {
	operation = strings.ToLower(strings.TrimSpace(operation))
	switch operation {
	case "delete", "archive", "retrieve", "rehydrate", "quarantine", "migration":
		return []string{"platform_admin", "security_officer", "compliance_officer"}
	default:
		return []string{"platform_admin", "security_officer", "data_steward"}
	}
}
