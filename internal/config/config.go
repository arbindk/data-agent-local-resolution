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
	PortalStatePath              string
	EphemeralContentPath         string
	AzureBlobScanJobsPath        string
	AzureBlobAsyncResultsPath    string
	SearchIndexMode              string
	SearchIndexEndpoint          string
	SearchIndexName              string
	SearchIndexAPIKey            string
	SearchIndexUsername          string
	SearchIndexPassword          string
	SearchIndexTimeoutSeconds    int
	AuthorizationMode            string
	SecretStoreProvider          string
	SecretStoreFilePath          string
	SecretStoreEnvPrefix         string
	AzureStorageConnectionString string
	AzureStorageAccount          string
	AzureBlobContainer           string
	AzureBlobPrefix              string
	AzureBlobDataStoreID         string
	AzureBlobBatchID             string
	AzureActionMode              string
	AzureWritebackEnabled        string
	AIProviderID                 string
	AIProviderEndpoint           string
	AIProviderModel              string
	AIProviderDeploymentName     string
	AIProviderAuthScheme         string
	AIProviderAPIKey             string
	AINetworkMode                string
	AIAutomationMode             string
	AIRedactionMode              string
	AIPromptAuditEnabled         bool
	AIEvidenceGroundingRequired  bool
	AIAllowHITLSubmission        bool
	AIAllowAutonomousApply       bool
	ActiveCompiledPolicyVersion  string
	ActiveCompiledPolicyTag      string
	ActiveCompiledPolicyHash     string
	ActiveWorkflowVersion        string
	ActiveWorkflowTag            string
	DefaultProcessingTier        string
	DefaultProcessingMode        string
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
		PortalStatePath:              getEnv("EVEREST_PORTAL_STATE_PATH", "data/portal_state.json"),
		EphemeralContentPath:         getEnv("EPHEMERAL_CONTENT_PATH", "data/ephemeral/content"),
		AzureBlobScanJobsPath:        getEnv("EVEREST_AZUREBLOB_SCAN_JOBS_PATH", "data/azureblob_scan_jobs.json"),
		AzureBlobAsyncResultsPath:    getEnv("EVEREST_AZUREBLOB_ASYNC_RESULTS_PATH", "data/async-normalized"),
		SearchIndexMode:              getEnv("EVEREST_SEARCH_INDEX_MODE", "local"),
		SearchIndexEndpoint:          getEnv("EVEREST_SEARCH_INDEX_ENDPOINT", ""),
		SearchIndexName:              getEnv("EVEREST_SEARCH_INDEX_NAME", "everest-summary"),
		SearchIndexAPIKey:            getEnv("EVEREST_SEARCH_INDEX_API_KEY", ""),
		SearchIndexUsername:          getEnv("EVEREST_SEARCH_INDEX_USERNAME", ""),
		SearchIndexPassword:          getEnv("EVEREST_SEARCH_INDEX_PASSWORD", ""),
		SearchIndexTimeoutSeconds:    getIntEnv("EVEREST_SEARCH_INDEX_TIMEOUT_SECONDS", 5),
		AuthorizationMode:            getEnv("EVEREST_AUTH_MODE", "audit"),
		SecretStoreProvider:          getEnv("EVEREST_SECRET_PROVIDER", "windows_credential_manager"),
		SecretStoreFilePath:          getEnv("EVEREST_SECRET_FILE_PATH", "data/secrets/local_secret_store.json"),
		SecretStoreEnvPrefix:         getEnv("EVEREST_SECRET_ENV_PREFIX", "EVEREST_SECRET_"),
		AzureStorageConnectionString: getEnv("AZURE_STORAGE_CONNECTION_STRING", ""),
		AzureStorageAccount:          getEnv("AZURE_STORAGE_ACCOUNT", ""),
		AzureBlobContainer:           getEnv("AZURE_BLOB_CONTAINER", "hulk01"),
		AzureBlobPrefix:              getEnv("AZURE_BLOB_PREFIX", ""),
		AzureBlobDataStoreID:         getEnv("AZURE_BLOB_DATA_STORE_ID", "azureblob-hulk01"),
		AzureBlobBatchID:             getEnv("AZURE_BLOB_BATCH_ID", "batch-azure-001"),
		AzureActionMode:              getEnv("AZURE_ACTION_MODE", "dry_run"),
		AzureWritebackEnabled:        getEnv("AZURE_WRITEBACK_ENABLED", "false"),
		AIProviderID:                 getEnv("EVEREST_AI_PROVIDER_ID", "everest-local-policy-copilot"),
		AIProviderEndpoint:           getEnv("EVEREST_AI_PROVIDER_ENDPOINT", ""),
		AIProviderModel:              getEnv("EVEREST_AI_PROVIDER_MODEL", "local-policy-rules-v1"),
		AIProviderDeploymentName:     getEnv("EVEREST_AI_PROVIDER_DEPLOYMENT", ""),
		AIProviderAuthScheme:         getEnv("EVEREST_AI_PROVIDER_AUTH_SCHEME", "none"),
		AIProviderAPIKey:             getEnv("EVEREST_AI_PROVIDER_API_KEY", ""),
		AINetworkMode:                getEnv("EVEREST_AI_NETWORK_MODE", "air_gapped"),
		AIAutomationMode:             getEnv("EVEREST_AI_AUTOMATION_MODE", "recommend_only"),
		AIRedactionMode:              getEnv("EVEREST_AI_REDACTION_MODE", "metadata_only"),
		AIPromptAuditEnabled:         getBoolEnv("EVEREST_AI_PROMPT_AUDIT_ENABLED", true),
		AIEvidenceGroundingRequired:  getBoolEnv("EVEREST_AI_EVIDENCE_GROUNDING_REQUIRED", true),
		AIAllowHITLSubmission:        getBoolEnv("EVEREST_AI_ALLOW_HITL_SUBMISSION", true),
		AIAllowAutonomousApply:       getBoolEnv("EVEREST_AI_ALLOW_AUTONOMOUS_APPLY", false),
		ActiveCompiledPolicyVersion:  getEnv("ACTIVE_COMPILED_POLICY_VERSION", "cpv-2026.05.29.001"),
		ActiveCompiledPolicyTag:      getEnv("ACTIVE_COMPILED_POLICY_TAG", "compiled/cpv-2026.05.29.001"),
		ActiveCompiledPolicyHash:     getEnv("ACTIVE_COMPILED_POLICY_HASH", "sha256:f5050ea952b39dc72b250d02b2c16f36a32b1e45da63f368cde0096444d8e73b"),
		ActiveWorkflowVersion:        getEnv("ACTIVE_WORKFLOW_VERSION", "workflow-mvs-linear-v1"),
		ActiveWorkflowTag:            getEnv("ACTIVE_WORKFLOW_TAG", "workflow/workflow-mvs-linear-v1"),
		DefaultProcessingTier:        getEnv("DEFAULT_PROCESSING_TIER", "tier_2"),
		DefaultProcessingMode:        getEnv("DEFAULT_PROCESSING_MODE", "standard"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return defaultValue
	}
}
func getIntEnv(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		out := 0
		for _, ch := range v {
			if ch < '0' || ch > '9' {
				return defaultValue
			}
			out = out*10 + int(ch-'0')
		}
		if out > 0 {
			return out
		}
	}
	return defaultValue
}
