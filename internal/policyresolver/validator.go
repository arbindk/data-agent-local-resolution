package policyresolver

import (
	"errors"
	"strings"
)

func ValidateLease(l PolicyLease) error {
	if strings.TrimSpace(l.LeaseID) == "" {
		return errors.New("lease_id is required")
	}
	if strings.TrimSpace(l.DataStoreID) == "" {
		return errors.New("data_store_id is required")
	}
	if strings.TrimSpace(l.CompiledPolicyVersion) == "" {
		return errors.New("compiled_policy_version is required")
	}
	if strings.TrimSpace(l.CompiledArtifactHash) == "" {
		return errors.New("compiled_artifact_hash is required")
	}
	if strings.TrimSpace(l.CompiledGitTag) == "" && strings.TrimSpace(l.CompiledGitCommit) == "" {
		return errors.New("compiled_git_tag or compiled_git_commit is required")
	}
	return nil
}
