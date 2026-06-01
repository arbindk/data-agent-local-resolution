package config

import "os"

type Config struct {
	CompiledRepoPath             string
	WorkflowRepoPath             string
	CachePath                    string
	ActivePolicyPath             string
	LKGPath                      string
	AuditPath                    string
	PublicKeyPath                string
	DuckDBPath                   string
	EphemeralContentPath         string
	AzureStorageConnectionString string
	AzureBlobContainer           string
	AzureBlobPrefix              string
	AzureBlobDataStoreID         string
	AzureBlobBatchID             string
	AzureActionMode              string
	AzureWritebackEnabled        string
}

func Load() Config {
	return Config{
		CompiledRepoPath:             getEnv("COMPILED_REPO_PATH", "demo/compiled-policy-repo"),
		WorkflowRepoPath:             getEnv("WORKFLOW_REPO_PATH", "demo/compiled-policy-repo/workflows"),
		CachePath:                    getEnv("POLICY_CACHE_PATH", "data/policy-cache"),
		ActivePolicyPath:             getEnv("ACTIVE_POLICY_PATH", "data/active-policy"),
		LKGPath:                      getEnv("LKG_POLICY_PATH", "data/last-known-good"),
		AuditPath:                    getEnv("AUDIT_PATH", "data/audit"),
		PublicKeyPath:                getEnv("POLICY_PUBLIC_KEY_PATH", "demo/keys/ed25519-public.b64"),
		DuckDBPath:                   getEnv("DUCKDB_PATH", "data/duckdb/agentcore.sqlite"),
		EphemeralContentPath:         getEnv("EPHEMERAL_CONTENT_PATH", "data/ephemeral/content"),
		AzureStorageConnectionString: getEnv("AZURE_STORAGE_CONNECTION_STRING", ""),
		AzureBlobContainer:           getEnv("AZURE_BLOB_CONTAINER", "hulk01"),
		AzureBlobPrefix:              getEnv("AZURE_BLOB_PREFIX", ""),
		AzureBlobDataStoreID:         getEnv("AZURE_BLOB_DATA_STORE_ID", "azureblob-hulk01"),
		AzureBlobBatchID:             getEnv("AZURE_BLOB_BATCH_ID", "batch-azure-001"),
		AzureActionMode:              getEnv("AZURE_ACTION_MODE", "dry_run"),
		AzureWritebackEnabled:        getEnv("AZURE_WRITEBACK_ENABLED", "false"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
