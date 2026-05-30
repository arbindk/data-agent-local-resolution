package policyresolver

import (
	"context"
	"fmt"
	"time"

	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/policyresolver/artifact"
	"everest.local/data-agent-policy-resolver/internal/policyresolver/cache"
	pcrypto "everest.local/data-agent-policy-resolver/internal/policyresolver/crypto"
	"everest.local/data-agent-policy-resolver/internal/policyresolver/lease"
)

type Resolver struct {
	cfg     config.Config
	git     *artifact.LocalGitResolver
	cache   *cache.PolicyCache
	gitDown bool
}

func NewResolver(cfg config.Config) *Resolver {
	return &Resolver{
		cfg:   cfg,
		git:   artifact.NewLocalGitResolver(cfg.CompiledRepoPath),
		cache: cache.NewPolicyCache(cfg.ActivePolicyPath, cfg.LKGPath),
	}
}

func (r *Resolver) SetGitUnavailable(v bool) {
	r.gitDown = v
}

func (r *Resolver) ResolveAndActivatePolicy(ctx context.Context, l PolicyLease) (*ResolveResult, error) {
	if err := lease.Validate(l); err != nil {
		return nil, err
	}

	ref := ArtifactRef{
		CompiledPolicyVersion: l.CompiledPolicyVersion,
		GitTag:                l.CompiledGitTag,
		GitCommit:             l.CompiledGitCommit,
		ExpectedHash:          l.CompiledArtifactHash,
	}

	if r.gitDown {
		return r.fallbackToLKG(l, "local_git_unavailable")
	}

	location, err := r.git.Resolve(ref)
	if err != nil {
		if l.AllowLastKnownGood {
			return r.fallbackToLKG(l, "local_git_resolution_failed")
		}
		return &ResolveResult{
			Status:                "failed_closed",
			DataStoreID:           l.DataStoreID,
			CompiledPolicyVersion: l.CompiledPolicyVersion,
			Reason:                err.Error(),
		}, nil
	}

	if err := pcrypto.VerifySHA256(location.ArtifactPath, l.CompiledArtifactHash); err != nil {
		return r.securityRejectOrFallback(l, "hash_verification_failed: "+err.Error())
	}

	if err := pcrypto.VerifyEd25519(r.cfg.PublicKeyPath, location.ArtifactPath, location.SignaturePath); err != nil {
		return r.securityRejectOrFallback(l, "signature_verification_failed: "+err.Error())
	}

	compiled, err := artifact.ParseCompiledPolicy(location.ArtifactPath)
	if err != nil {
		return r.securityRejectOrFallback(l, "artifact_parse_failed: "+err.Error())
	}

	policyCtx := &PolicyContext{
		DataStoreID:             l.DataStoreID,
		CompiledPolicyVersion:   compiled.Metadata.CompiledPolicyVersion,
		SourcePolicyID:          compiled.Metadata.SourcePolicyID,
		SourcePolicyVersion:     compiled.Metadata.SourcePolicyVersion,
		ArtifactHash:            l.CompiledArtifactHash,
		SignatureVerified:       true,
		ArtifactHashVerified:    true,
		EntitiesOfInterest:      compiled.EntitiesOfInterest,
		ProcessingMode:          "summary",
		SovereigntyRules:        compiled.CompiledPolicyLogic.SovereigntyRules,
		ActivationSource:        "local_git",
		FallbackUsed:            false,
		LocalExecutionConfirmed: true,
		ActivatedAt:             time.Now().UTC(),
	}

	if err := r.cache.Activate(policyCtx); err != nil {
		return nil, fmt.Errorf("activate policy: %w", err)
	}

	return &ResolveResult{
		Status:                "activated",
		DataStoreID:           l.DataStoreID,
		CompiledPolicyVersion: l.CompiledPolicyVersion,
		ArtifactHashVerified:  true,
		SignatureVerified:     true,
		ActivationSource:      "local_git",
		FallbackUsed:          false,
		LocalExecutionReady:   true,
		PolicyContext:         policyCtx,
	}, nil
}

func (r *Resolver) GetActivePolicy(dataStoreID string) (*PolicyContext, bool) {
	return r.cache.GetActive(dataStoreID)
}

func (r *Resolver) fallbackToLKG(l PolicyLease, reason string) (*ResolveResult, error) {
	if !l.AllowLastKnownGood {
		return &ResolveResult{
			Status:                "failed_closed",
			DataStoreID:           l.DataStoreID,
			CompiledPolicyVersion: l.CompiledPolicyVersion,
			Reason:                reason,
		}, nil
	}

	lkg, err := r.cache.GetLastKnownGood(l.DataStoreID)
	if err != nil {
		return &ResolveResult{
			Status:                "failed_closed",
			DataStoreID:           l.DataStoreID,
			CompiledPolicyVersion: l.CompiledPolicyVersion,
			Reason:                "no_verified_last_known_good_available: " + err.Error(),
		}, nil
	}

	lkg.FallbackUsed = true
	lkg.ActivationSource = "last_known_good"

	return &ResolveResult{
		Status:                "degraded_ready",
		DataStoreID:           l.DataStoreID,
		CompiledPolicyVersion: lkg.CompiledPolicyVersion,
		ArtifactHashVerified:  lkg.ArtifactHashVerified,
		SignatureVerified:     lkg.SignatureVerified,
		ActivationSource:      "last_known_good",
		FallbackUsed:          true,
		LocalExecutionReady:   true,
		Reason:                reason,
		PolicyContext:         lkg,
	}, nil
}

func (r *Resolver) securityRejectOrFallback(l PolicyLease, reason string) (*ResolveResult, error) {
	if l.AllowLastKnownGood {
		return r.fallbackToLKG(l, reason)
	}

	return &ResolveResult{
		Status:                "rejected",
		DataStoreID:           l.DataStoreID,
		CompiledPolicyVersion: l.CompiledPolicyVersion,
		Reason:                reason,
	}, nil
}
