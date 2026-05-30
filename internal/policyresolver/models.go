package policyresolver

import "time"

type PolicyLease struct {
	LeaseID               string `json:"lease_id"`
	JobID                 string `json:"job_id"`
	DataStoreID           string `json:"data_store_id"`
	CompiledPolicyVersion string `json:"compiled_policy_version"`
	CompiledGitTag        string `json:"compiled_git_tag,omitempty"`
	CompiledGitCommit     string `json:"compiled_git_commit,omitempty"`
	CompiledArtifactHash  string `json:"compiled_artifact_hash"`
	AllowLastKnownGood    bool   `json:"allow_last_known_good"`
}

type ArtifactRef struct {
	CompiledPolicyVersion string
	GitTag                string
	GitCommit             string
	ExpectedHash          string
}

type ArtifactLocation struct {
	PolicyDir     string
	ArtifactPath  string
	SignaturePath string
	ManifestPath  string
}

type CompiledPolicyArtifact struct {
	Metadata            PolicyMetadata      `json:"metadata"`
	CompiledPolicyLogic CompiledPolicyLogic `json:"compiled_policy_logic"`
	EntitiesOfInterest  EntitiesOfInterest  `json:"entities_of_interest"`
	Signature           SignatureMetadata   `json:"signature"`
}

type PolicyMetadata struct {
	CompiledPolicyVersion string `json:"compiled_policy_version"`
	SourcePolicyID        string `json:"source_policy_id"`
	SourcePolicyVersion   string `json:"source_policy_version"`
	CompiledGitTag        string `json:"compiled_git_tag"`
	CompiledGitCommit     string `json:"compiled_git_commit"`
	CreatedAt             string `json:"created_at"`
	CompiledBy            string `json:"compiled_by"`
	HITLApprovalID        string `json:"hitl_approval_id"`
	ArtifactHash          string `json:"artifact_hash"`
}

type CompiledPolicyLogic struct {
	ClassificationRules []ClassificationRule `json:"classification_rules"`
	RemediationRules    []RemediationRule    `json:"remediation_rules"`
	SovereigntyRules    []SovereigntyRule    `json:"sovereignty_rules"`
}

type ClassificationRule struct {
	RuleID              string `json:"rule_id"`
	Entity              string `json:"entity"`
	Condition           string `json:"condition"`
	SensitivityLabel    string `json:"sensitivity_label"`
	RiskScore           int    `json:"risk_score"`
	HumanReviewRequired bool   `json:"human_review_required"`
}

type RemediationRule struct {
	RuleID                string   `json:"rule_id"`
	Trigger               string   `json:"trigger"`
	AllowedActions        []string `json:"allowed_actions"`
	RequiresHumanApproval bool     `json:"requires_human_approval"`
}

type SovereigntyRule struct {
	RuleID                     string `json:"rule_id"`
	DataMovement               string `json:"data_movement"`
	AllowCrossBoundaryTransfer bool   `json:"allow_cross_boundary_transfer"`
}

type EntitiesOfInterest struct {
	EntityTypes   []string       `json:"entity_types"`
	KeywordGroups []KeywordGroup `json:"keyword_groups"`
	RegexGroups   []RegexGroup   `json:"regex_groups"`
}

type KeywordGroup struct {
	Group    string   `json:"group"`
	Keywords []string `json:"keywords"`
}

type RegexGroup struct {
	Group    string   `json:"group"`
	Patterns []string `json:"patterns"`
}

type SignatureMetadata struct {
	Algorithm     string `json:"algorithm"`
	KeyID         string `json:"key_id"`
	SignatureFile string `json:"signature_file"`
}

type PolicyContext struct {
	DataStoreID             string             `json:"data_store_id"`
	CompiledPolicyVersion   string             `json:"compiled_policy_version"`
	SourcePolicyID          string             `json:"source_policy_id"`
	SourcePolicyVersion     string             `json:"source_policy_version"`
	ArtifactHash            string             `json:"artifact_hash"`
	SignatureVerified       bool               `json:"signature_verified"`
	ArtifactHashVerified    bool               `json:"artifact_hash_verified"`
	EntitiesOfInterest      EntitiesOfInterest `json:"entities_of_interest"`
	ProcessingMode          string             `json:"processing_mode"`
	SovereigntyRules        []SovereigntyRule  `json:"sovereignty_rules"`
	ActivationSource        string             `json:"activation_source"`
	FallbackUsed            bool               `json:"fallback_used"`
	LocalExecutionConfirmed bool               `json:"local_execution_confirmed"`
	ActivatedAt             time.Time          `json:"activated_at"`
}

type ResolveResult struct {
	Status                string         `json:"status"`
	DataStoreID           string         `json:"data_store_id"`
	CompiledPolicyVersion string         `json:"compiled_policy_version"`
	ArtifactHashVerified  bool           `json:"artifact_hash_verified"`
	SignatureVerified     bool           `json:"signature_verified"`
	ActivationSource      string         `json:"activation_source"`
	FallbackUsed          bool           `json:"fallback_used"`
	LocalExecutionReady   bool           `json:"local_execution_ready"`
	Reason                string         `json:"reason,omitempty"`
	PolicyContext         *PolicyContext `json:"policy_context,omitempty"`
}
