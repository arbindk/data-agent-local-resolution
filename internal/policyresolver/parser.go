package policyresolver

import (
	"encoding/json"
	"fmt"
	"os"
)

func ParseCompiledPolicy(path string) (*CompiledPolicyArtifact, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compiled policy: %w", err)
	}

	var artifact CompiledPolicyArtifact
	if err := json.Unmarshal(b, &artifact); err != nil {
		return nil, fmt.Errorf("parse compiled policy json: %w", err)
	}

	if artifact.Metadata.CompiledPolicyVersion == "" {
		return nil, fmt.Errorf("compiled_policy_version is missing")
	}
	if artifact.Metadata.SourcePolicyID == "" {
		return nil, fmt.Errorf("source_policy_id is missing")
	}
	if artifact.Metadata.SourcePolicyVersion == "" {
		return nil, fmt.Errorf("source_policy_version is missing")
	}
	if artifact.Metadata.HITLApprovalID == "" {
		return nil, fmt.Errorf("hitl_approval_id is missing")
	}
	if artifact.Signature.Algorithm == "" {
		return nil, fmt.Errorf("signature algorithm is missing")
	}
	if len(artifact.EntitiesOfInterest.EntityTypes) == 0 {
		return nil, fmt.Errorf("entities_of_interest.entity_types is empty")
	}

	return &artifact, nil
}
