package config

import "os"

type Config struct {
	CompiledRepoPath string
	WorkflowRepoPath string
	CachePath        string
	ActivePolicyPath string
	LKGPath          string
	AuditPath        string
	PublicKeyPath    string
}

func Load() Config {
	return Config{
		CompiledRepoPath: getEnv("COMPILED_REPO_PATH", "demo/compiled-policy-repo"),
		WorkflowRepoPath: getEnv("WORKFLOW_REPO_PATH", "demo/compiled-policy-repo/workflows"),
		CachePath:        getEnv("POLICY_CACHE_PATH", "data/policy-cache"),
		ActivePolicyPath: getEnv("ACTIVE_POLICY_PATH", "data/active-policy"),
		LKGPath:          getEnv("LKG_POLICY_PATH", "data/last-known-good"),
		AuditPath:        getEnv("AUDIT_PATH", "data/audit"),
		PublicKeyPath:    getEnv("POLICY_PUBLIC_KEY_PATH", "demo/keys/ed25519-public.b64"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
