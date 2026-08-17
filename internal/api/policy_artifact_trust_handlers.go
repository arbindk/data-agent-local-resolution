package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

func (s *Server) policyArtifactTrust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	dataStoreID := strings.TrimSpace(r.URL.Query().Get("data_store_id"))
	if dataStoreID == "" {
		dataStoreID = s.cfg.AzureBlobDataStoreID
	}

	compiledPolicyVersion := s.cfg.ActiveCompiledPolicyVersion
	if compiledPolicyVersion == "" {
		compiledPolicyVersion = "cpv-2026.05.29.001"
	}

	policyDir := filepath.Join(s.cfg.CompiledRepoPath, "policies", compiledPolicyVersion)
	artifactPath := filepath.Join(policyDir, "compiled-policy.json")
	signaturePath := filepath.Join(policyDir, "compiled-policy.sig")

	actualHash, hashErr := sha256File(artifactPath)
	expectedHash := s.cfg.ActiveCompiledPolicyHash

	hashVerified := hashErr == nil && expectedHash != "" && actualHash == expectedHash

	signatureVerified := false
	signatureStatus := "not_verified"
	if _, err := os.Stat(signaturePath); err == nil {
		if err := policyresolver.VerifyEd25519(s.cfg.PublicKeyPath, artifactPath, signaturePath); err == nil {
			signatureVerified = true
			signatureStatus = "verified"
		} else {
			signatureStatus = "failed: " + err.Error()
		}
	} else {
		signatureStatus = "missing: " + err.Error()
	}

	activePolicyPath := filepath.Join("data", "active-policy", dataStoreID+".json")
	lkgPolicyPath := filepath.Join(s.cfg.LKGPath, dataStoreID+".json")

	activeAvailable := fileExists(activePolicyPath)
	lkgAvailable := fileExists(lkgPolicyPath)

	activePolicy := map[string]any{}
	if activeAvailable {
		_ = readJSONFile(activePolicyPath, &activePolicy)
	}

	lkgPolicy := map[string]any{}
	if lkgAvailable {
		_ = readJSONFile(lkgPolicyPath, &lkgPolicy)
	}

	verificationStatus := "verified"
	if hashErr != nil {
		verificationStatus = "artifact_missing_or_unreadable"
	} else if !hashVerified {
		verificationStatus = "hash_mismatch"
	} else if !signatureVerified {
		verificationStatus = "signature_not_verified"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data_store_id":               dataStoreID,
		"compiled_policy_version":     compiledPolicyVersion,
		"compiled_git_tag":            s.cfg.ActiveCompiledPolicyTag,
		"compiled_git_commit":         gitCommit(s.cfg.CompiledRepoPath),
		"workflow_git_tag":            s.cfg.ActiveWorkflowTag,
		"artifact_source":             "local_git",
		"artifact_path":               artifactPath,
		"signature_path":              signaturePath,
		"expected_artifact_hash":      expectedHash,
		"actual_artifact_hash":        actualHash,
		"artifact_hash_verified":      hashVerified,
		"signature_verified":          signatureVerified,
		"signature_status":            signatureStatus,
		"verification_status":         verificationStatus,
		"active_policy_available":     activeAvailable,
		"last_known_good_available":   lkgAvailable,
		"active_policy_path":          activePolicyPath,
		"last_known_good_policy_path": lkgPolicyPath,
		"activation_source":           firstString(activePolicy["activation_source"], lkgPolicy["activation_source"], "local_git"),
		"rollback_available":          lkgAvailable,
		"checked_at":                  time.Now().UTC().Format(time.RFC3339Nano),
		"item13_scope": []string{
			"local_git_resolution",
			"compiled_policy_artifact_verification",
			"artifact_hash_verification",
			"signature_verification",
			"active_policy_cache",
			"last_known_good_fallback",
			"local_execution_readiness",
		},
	})
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readJSONFile(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func gitCommit(repoPath string) string {
	if repoPath == "" {
		return ""
	}

	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func firstString(values ...any) string {
	for _, value := range values {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		case error:
			if v != nil && !errors.Is(v, os.ErrNotExist) {
				return v.Error()
			}
		}
	}
	return ""
}
