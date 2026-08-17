package platform

import (
	"sort"
	"strings"
	"time"
)

type RoleProfile struct {
	Role              string   `json:"role"`
	Description       string   `json:"description"`
	AllowedOperations []string `json:"allowed_operations"`
	RequiresApproval  bool     `json:"requires_approval"`
}

type PolicyPrecedenceRule struct {
	PolicyID   string `json:"policy_id"`
	Name       string `json:"name"`
	Precedence int    `json:"precedence"`
	Effect     string `json:"effect"`
	Reason     string `json:"reason"`
}

type JurisdictionRule struct {
	RuleID       string `json:"rule_id"`
	SourceRegion string `json:"source_region"`
	TargetRegion string `json:"target_region"`
	Operation    string `json:"operation"`
	Decision     string `json:"decision"`
	Reason       string `json:"reason"`
	HITLRequired bool   `json:"hitl_required"`
	OwnerRole    string `json:"owner_role"`
}

type HITLRule struct {
	Trigger       string   `json:"trigger"`
	RequiredRoles []string `json:"required_roles"`
	Reason        string   `json:"reason"`
}

type GovernanceContextResponse struct {
	Status            string                 `json:"status"`
	Roles             []RoleProfile          `json:"roles"`
	PolicyPrecedence  []PolicyPrecedenceRule `json:"policy_precedence"`
	JurisdictionRules []JurisdictionRule     `json:"jurisdiction_rules"`
	HITLRules         []HITLRule             `json:"hitl_rules"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

type GovernanceEvaluateRequest struct {
	TenantID         string   `json:"tenant_id"`
	RequestedBy      string   `json:"requested_by"`
	Role             string   `json:"role"`
	Operation        string   `json:"operation"`
	SourceRegion     string   `json:"source_region"`
	TargetRegion     string   `json:"target_region"`
	ContractOwner    string   `json:"contract_owner"`
	DataStoreID      string   `json:"data_store_id"`
	PolicyIDs        []string `json:"policy_ids"`
	ActionMode       string   `json:"action_mode"`
	WritebackEnabled bool     `json:"writeback_enabled"`
}

type GovernanceEvaluateResponse struct {
	Status            string                 `json:"status"`
	Decision          string                 `json:"decision"`
	Reason            string                 `json:"reason"`
	AppliedPolicyID   string                 `json:"applied_policy_id,omitempty"`
	AppliedPolicyName string                 `json:"applied_policy_name,omitempty"`
	PolicyPrecedence  []PolicyPrecedenceRule `json:"policy_precedence"`
	Role              RoleProfile            `json:"role"`
	JurisdictionRule  *JurisdictionRule      `json:"jurisdiction_rule,omitempty"`
	HITLRequired      bool                   `json:"hitl_required"`
	RequiredApprovers []string               `json:"required_approvers,omitempty"`
	ContractOwner     string                 `json:"contract_owner,omitempty"`
	Controls          []string               `json:"controls"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

func (s *Store) GovernanceContext() GovernanceContextResponse {
	return GovernanceContextResponse{
		Status:            "ok",
		Roles:             defaultRoleProfiles(),
		PolicyPrecedence:  defaultPolicyPrecedence(),
		JurisdictionRules: defaultJurisdictionRules(),
		HITLRules:         defaultHITLRules(),
		GeneratedAt:       time.Now().UTC(),
	}
}

func (s *Store) EvaluateGovernance(req GovernanceEvaluateRequest) GovernanceEvaluateResponse {
	req.Role = normalize(req.Role)
	req.Operation = normalize(req.Operation)
	req.SourceRegion = strings.ToUpper(strings.TrimSpace(req.SourceRegion))
	req.TargetRegion = strings.ToUpper(strings.TrimSpace(req.TargetRegion))
	req.ActionMode = normalize(req.ActionMode)

	if req.Role == "" {
		req.Role = "data_steward"
	}
	if req.Operation == "" {
		req.Operation = "metadata_scan"
	}
	if req.ActionMode == "" {
		req.ActionMode = "dry_run"
	}

	role := findRoleProfile(req.Role)
	precedence := selectedPolicyPrecedence(req.PolicyIDs)

	resp := GovernanceEvaluateResponse{
		Status:           "ok",
		Decision:         "allow",
		Reason:           "Operation is allowed by current role, policy, and jurisdiction controls.",
		PolicyPrecedence: precedence,
		Role:             role,
		HITLRequired:     false,
		ContractOwner:    strings.TrimSpace(req.ContractOwner),
		Controls: []string{
			"role_access_checked",
			"policy_precedence_evaluated",
			"jurisdiction_checked",
			"guardian_required_for_writeback",
		},
		GeneratedAt: time.Now().UTC(),
	}

	if len(precedence) > 0 {
		resp.AppliedPolicyID = precedence[0].PolicyID
		resp.AppliedPolicyName = precedence[0].Name
	}

	if !roleAllows(role, req.Operation) {
		resp.Decision = "block"
		resp.Reason = "Requested role is not allowed to perform this operation."
		resp.Controls = append(resp.Controls, "blocked_by_role_access")
		return resp
	}

	if jr := matchJurisdictionRule(req.SourceRegion, req.TargetRegion, req.Operation); jr != nil {
		resp.JurisdictionRule = jr
		resp.Controls = append(resp.Controls, "jurisdiction_rule_matched:"+jr.RuleID)

		if jr.HITLRequired {
			resp.HITLRequired = true
			resp.RequiredApprovers = uniqueAppend(resp.RequiredApprovers, jr.OwnerRole)
		}

		if jr.Decision == "block" {
			resp.Decision = "block"
			resp.Reason = jr.Reason
			resp.Controls = append(resp.Controls, "blocked_by_jurisdiction")
			return resp
		}

		if jr.Decision == "review_required" {
			resp.Decision = "review_required"
			resp.Reason = jr.Reason
		}
	}

	if isDestructiveOperation(req.Operation) {
		resp.Decision = "review_required"
		resp.HITLRequired = true
		resp.RequiredApprovers = uniqueAppend(resp.RequiredApprovers, "security_officer", "contract_owner")
		resp.Reason = "Destructive/recovery operations require human review and named owner approval."
		resp.Controls = append(resp.Controls, "hitl_required_for_destructive_operation")
	}

	if req.ActionMode == "apply" || req.WritebackEnabled {
		resp.HITLRequired = true
		resp.RequiredApprovers = uniqueAppend(resp.RequiredApprovers, "data_steward", "security_officer")
		resp.Controls = append(resp.Controls, "guardian_required_for_apply_mode")
		if resp.Decision == "allow" {
			resp.Decision = "review_required"
			resp.Reason = "Apply mode requires Guardian allow decision and authorized approval before real write-back."
		}
	}

	if strings.TrimSpace(req.ContractOwner) == "" && isDestructiveOperation(req.Operation) {
		resp.Decision = "review_required"
		resp.HITLRequired = true
		resp.RequiredApprovers = uniqueAppend(resp.RequiredApprovers, "contract_owner")
		resp.Reason = "A named contract owner is required for restricted Azure Blob actions."
		resp.Controls = append(resp.Controls, "contract_owner_required")
	}

	return resp
}

func defaultRoleProfiles() []RoleProfile {
	return []RoleProfile{
		{Role: "viewer", Description: "Read-only visibility into dashboard, evidence, and audit.", AllowedOperations: []string{"metadata_scan", "view_evidence"}, RequiresApproval: false},
		{Role: "data_steward", Description: "Can create scans, review findings, and request non-destructive write-back.", AllowedOperations: []string{"metadata_scan", "apply_blob_metadata", "apply_blob_index_tags", "view_evidence"}, RequiresApproval: true},
		{Role: "security_officer", Description: "Can approve Guardian-gated actions and review sensitive findings.", AllowedOperations: []string{"metadata_scan", "apply_blob_metadata", "apply_blob_index_tags", "quarantine", "delete", "view_evidence"}, RequiresApproval: true},
		{Role: "platform_admin", Description: "Can operate platform storage, agents, and job execution.", AllowedOperations: []string{"metadata_scan", "apply_blob_metadata", "apply_blob_index_tags", "archive", "retrieve", "rehydrate", "quarantine", "migration", "delete", "view_evidence"}, RequiresApproval: true},
		{Role: "contract_owner", Description: "Business owner responsible for migration/archive contractual approval.", AllowedOperations: []string{"metadata_scan", "migration", "archive", "retrieve", "rehydrate", "quarantine", "delete", "view_evidence"}, RequiresApproval: true},
	}
}

func defaultPolicyPrecedence() []PolicyPrecedenceRule {
	return []PolicyPrecedenceRule{
		{PolicyID: "regional-data-residency", Name: "Regional Data Residency", Precedence: 100, Effect: "block_or_review", Reason: "Jurisdiction restrictions override operational policies."},
		{PolicyID: "destructive-action-safety", Name: "Destructive Action Safety Policy", Precedence: 90, Effect: "review_required", Reason: "Delete/archive/retrieve/quarantine/migration require HITL."},
		{PolicyID: "azureblob-approved-tagging", Name: "Approved Azure Metadata / Index Tagging", Precedence: 70, Effect: "allow_with_guardian", Reason: "Metadata/index tags may be applied only after Guardian allow."},
		{PolicyID: "azureblob-confidential-tagging-preview", Name: "Confidential Data Tagging Preview", Precedence: 60, Effect: "dry_run", Reason: "Preview-only tagging plan."},
		{PolicyID: "azureblob-metadata-discovery", Name: "Azure Blob Metadata Discovery", Precedence: 50, Effect: "allow_read_only", Reason: "Read-only metadata discovery."},
	}
}

func defaultJurisdictionRules() []JurisdictionRule {
	return []JurisdictionRule{
		{RuleID: "IN_TO_AE_MIGRATION_BLOCK", SourceRegion: "IN", TargetRegion: "AE", Operation: "migration", Decision: "block", Reason: "Migration from India to UAE is blocked unless an explicit customer legal/contract exception policy is configured.", HITLRequired: true, OwnerRole: "contract_owner"},
		{RuleID: "CROSS_REGION_MIGRATION_REVIEW", SourceRegion: "*", TargetRegion: "*", Operation: "migration", Decision: "review_required", Reason: "Cross-region migration requires jurisdiction review and named contract owner approval.", HITLRequired: true, OwnerRole: "contract_owner"},
		{RuleID: "IN_PLACE_TAGGING_ALLOWED", SourceRegion: "*", TargetRegion: "*", Operation: "apply_blob_metadata", Decision: "allow", Reason: "In-place metadata/index tagging does not move data across jurisdiction boundaries.", HITLRequired: false, OwnerRole: "data_steward"},
	}
}

func defaultHITLRules() []HITLRule {
	return []HITLRule{
		{Trigger: "apply_mode_writeback", RequiredRoles: []string{"data_steward", "security_officer"}, Reason: "Real write-back needs Guardian allow and accountable approval."},
		{Trigger: "cross_region_migration", RequiredRoles: []string{"contract_owner", "security_officer"}, Reason: "Movement across jurisdictions needs business/legal accountability."},
		{Trigger: "destructive_action", RequiredRoles: []string{"contract_owner", "security_officer"}, Reason: "Destructive operations require named owner accountability."},
	}
}

func selectedPolicyPrecedence(policyIDs []string) []PolicyPrecedenceRule {
	all := defaultPolicyPrecedence()
	if len(policyIDs) == 0 {
		return all
	}

	selected := map[string]bool{}
	for _, id := range policyIDs {
		selected[strings.TrimSpace(id)] = true
	}

	out := []PolicyPrecedenceRule{}
	for _, p := range all {
		if selected[p.PolicyID] {
			out = append(out, p)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Precedence > out[j].Precedence
	})

	return out
}

func findRoleProfile(role string) RoleProfile {
	for _, r := range defaultRoleProfiles() {
		if r.Role == role {
			return r
		}
	}
	return defaultRoleProfiles()[1]
}

func roleAllows(role RoleProfile, operation string) bool {
	for _, op := range role.AllowedOperations {
		if op == operation {
			return true
		}
	}
	return false
}

func matchJurisdictionRule(source string, target string, operation string) *JurisdictionRule {
	for _, rule := range defaultJurisdictionRules() {
		if rule.Operation != operation {
			continue
		}
		if rule.SourceRegion == source && rule.TargetRegion == target {
			r := rule
			return &r
		}
	}

	if source != "" && target != "" && source != target && operation == "migration" {
		for _, rule := range defaultJurisdictionRules() {
			if rule.RuleID == "CROSS_REGION_MIGRATION_REVIEW" {
				r := rule
				return &r
			}
		}
	}

	if operation == "apply_blob_metadata" || operation == "apply_blob_index_tags" {
		for _, rule := range defaultJurisdictionRules() {
			if rule.RuleID == "IN_PLACE_TAGGING_ALLOWED" {
				r := rule
				return &r
			}
		}
	}

	return nil
}

func isDestructiveOperation(operation string) bool {
	switch operation {
	case "delete", "archive", "retrieve", "rehydrate", "quarantine", "migration":
		return true
	default:
		return false
	}
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func uniqueAppend(existing []string, roles ...string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range existing {
		r = normalize(r)
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	for _, r := range roles {
		r = normalize(r)
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}
