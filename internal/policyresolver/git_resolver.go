package policyresolver

import (
	"fmt"
	"os"
	"path/filepath"
)

type LocalGitResolver struct {
	RepoPath string
}

func NewLocalGitResolver(repoPath string) *LocalGitResolver {
	return &LocalGitResolver{RepoPath: repoPath}
}

func (r *LocalGitResolver) Resolve(ref ArtifactRef) (*ArtifactLocation, error) {
	if _, err := os.Stat(r.RepoPath); err != nil {
		return nil, fmt.Errorf("compiled policy repo unavailable: %w", err)
	}

	policyDir := filepath.Join(r.RepoPath, "policies", ref.CompiledPolicyVersion)

	artifactPath := filepath.Join(policyDir, "compiled-policy.json")
	signaturePath := filepath.Join(policyDir, "compiled-policy.sig")
	manifestPath := filepath.Join(policyDir, "manifest.json")

	if _, err := os.Stat(artifactPath); err != nil {
		return nil, fmt.Errorf("compiled policy artifact not found: %w", err)
	}
	if _, err := os.Stat(signaturePath); err != nil {
		return nil, fmt.Errorf("compiled policy signature not found: %w", err)
	}

	return &ArtifactLocation{
		PolicyDir:     policyDir,
		ArtifactPath:  artifactPath,
		SignaturePath: signaturePath,
		ManifestPath:  manifestPath,
	}, nil
}
