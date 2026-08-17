package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	aiproviders "everest.local/data-agent-policy-resolver/internal/ai/providers"
	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/controlplane"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
	"everest.local/data-agent-policy-resolver/internal/pluginruntime"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
	"everest.local/data-agent-policy-resolver/internal/searchindex"
	"everest.local/data-agent-policy-resolver/internal/secretstore"
)

type Server struct {
	cfg                config.Config
	resolver           *policyresolver.Resolver
	orchestrator       *execution.Orchestrator
	store              *duckdbstore.Store
	azureConfigStore   *azureblob.RuntimeConfigStore
	aiConfigStore      *aiproviders.RuntimeConfigStore
	runtimeAudit       *pluginruntime.Store
	scanJobs           *controlplane.AzureBlobScanJobStore
	portalState        *portalstate.Store
	secrets            *secretstore.Store
	searchIndex        *searchindex.ConfiguredIndex
	azureBlobConnector azureBlobConnectorPort
}

func NewServer(cfg config.Config, resolver *policyresolver.Resolver, orchestrator *execution.Orchestrator, store *duckdbstore.Store) *Server {
	initialAzureConfig := azureblob.Config{
		ConnectionString: "",
		StorageAccount:   cfg.AzureStorageAccount,
		ContainerName:    cfg.AzureBlobContainer,
		Prefix:           cfg.AzureBlobPrefix,
		DataStoreID:      cfg.AzureBlobDataStoreID,
		BatchID:          cfg.AzureBlobBatchID,
		ActionMode:       cfg.AzureActionMode,
		WritebackEnabled: cfg.AzureWritebackEnabled,
	}
	initialAIConfig := aiproviders.Config{
		ProviderID:                cfg.AIProviderID,
		EndpointURL:               cfg.AIProviderEndpoint,
		Model:                     cfg.AIProviderModel,
		DeploymentName:            cfg.AIProviderDeploymentName,
		AuthScheme:                cfg.AIProviderAuthScheme,
		NetworkMode:               cfg.AINetworkMode,
		AutomationMode:            cfg.AIAutomationMode,
		RedactionMode:             cfg.AIRedactionMode,
		PromptAuditEnabled:        cfg.AIPromptAuditEnabled,
		EvidenceGroundingRequired: cfg.AIEvidenceGroundingRequired,
		AllowHITLSubmission:       cfg.AIAllowHITLSubmission,
		AllowAutonomousApply:      cfg.AIAllowAutonomousApply,
	}

	secretStore := secretstore.NewWithOptions(secretstore.Options{
		Provider:  cfg.SecretStoreProvider,
		FilePath:  cfg.SecretStoreFilePath,
		EnvPrefix: cfg.SecretStoreEnvPrefix,
	})
	portalStateStore := portalstate.New(cfg.PortalStatePath)
	searchIndexPublisher := searchindex.NewConfiguredIndex(searchindex.FromAppConfig(cfg), portalStateStore)
	if orchestrator != nil {
		orchestrator.SetSearchIndex(searchIndexPublisher)
	}

	if cfg.AzureStorageConnectionString != "" && !secretStore.Exists(azureblob.ConnectionStringSecretName) {
		_ = secretStore.Save(azureblob.ConnectionStringSecretName, cfg.AzureStorageConnectionString)
	}
	if cfg.AzureStorageConnectionString != "" {
		sourceSecret := azureblob.SourceCredentialSecretName(firstNonBlank(cfg.AzureBlobDataStoreID, cfg.AzureStorageAccount, "azureblob-local"))
		if !secretStore.Exists(sourceSecret) {
			_ = secretStore.Save(sourceSecret, cfg.AzureStorageConnectionString)
		}
	}
	if cfg.AIProviderAPIKey != "" && !secretStore.Exists(aiproviders.SecretName(cfg.AIProviderID, "api_key")) {
		_ = secretStore.Save(aiproviders.SecretName(cfg.AIProviderID, "api_key"), cfg.AIProviderAPIKey)
	}
	portalStateStore.UpsertAzureSource(portalstate.AzureSource{
		SourceID:         cfg.AzureBlobDataStoreID,
		DataStoreID:      cfg.AzureBlobDataStoreID,
		DisplayName:      "Azure Blob - " + cfg.AzureBlobDataStoreID,
		ConnectorID:      "azureblob",
		DeploymentMode:   "cloud_connected",
		StorageAccount:   cfg.AzureStorageAccount,
		Container:        cfg.AzureBlobContainer,
		Prefix:           cfg.AzureBlobPrefix,
		ActionMode:       cfg.AzureActionMode,
		WritebackEnabled: cfg.AzureWritebackEnabled,
		CredentialSet:    secretStore.Exists(azureblob.SourceCredentialSecretName(firstNonBlank(cfg.AzureBlobDataStoreID, cfg.AzureStorageAccount, "azureblob-local"))) || cfg.AzureStorageConnectionString != "",
	})
	portalStateStore.UpsertAIProvider(portalstate.AIProvider{
		ProviderID:    cfg.AIProviderID,
		Model:         cfg.AIProviderModel,
		NetworkMode:   cfg.AINetworkMode,
		Configured:    true,
		CredentialSet: secretStore.Exists(aiproviders.SecretName(cfg.AIProviderID, "api_key")) || cfg.AIProviderAPIKey != "",
	})

	return &Server{
		cfg:                cfg,
		resolver:           resolver,
		orchestrator:       orchestrator,
		store:              store,
		azureConfigStore:   azureblob.NewRuntimeConfigStore(initialAzureConfig),
		aiConfigStore:      aiproviders.NewRuntimeConfigStore(initialAIConfig),
		runtimeAudit:       pluginruntime.NewStore(500),
		scanJobs:           controlplane.NewAzureBlobScanJobStoreWithPath(cfg.AzureBlobScanJobsPath),
		portalState:        portalStateStore,
		secrets:            secretStore,
		searchIndex:        searchIndexPublisher,
		azureBlobConnector: defaultAzureBlobConnector{},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/connectors/azureblob/config", s.azureBlobConfigHandler)
	mux.HandleFunc("/v1/connectors/azureblob/test", s.testAzureBlob)
	mux.HandleFunc("/v1/connectors/azureblob/scan", s.scanAzureBlob)
	mux.HandleFunc("/v1/scan/azureblob/jobs", s.azureBlobAsyncScanJobs)
	mux.HandleFunc("/v1/scan/azureblob/jobs/", s.azureBlobAsyncScanJobByID)
	mux.HandleFunc("/v1/connectors/azureblob/status", s.azureBlobStatus)
	mux.HandleFunc("/v1/connectors/azureblob/containers", s.listAzureBlobContainers)
	mux.HandleFunc("/v1/connectors/azureblob/writeback", s.azureBlobWriteback)
	mux.HandleFunc("/v1/connectors/azureblob/writeback/apply-approved", s.applyApprovedAzureBlobWritebacks)
	mux.HandleFunc("/v1/connectors/azureblob/actions/execute", s.azureBlobActionExecute)
	mux.HandleFunc("/v1/tools/azureblob/workbench", s.azureBlobWorkbench)
	mux.HandleFunc("/v1/connectors/catalog", s.connectorCatalog)
	mux.HandleFunc("/v1/connectors/manifest/validate", s.validateConnectorManifest)
	mux.HandleFunc("/v1/connectors/runtime/invoke", s.connectorRuntimeInvoke)
	mux.HandleFunc("/v1/ai/providers/catalog", s.aiProviderCatalog)
	mux.HandleFunc("/v1/ai/providers/manifest/validate", s.validateAIProviderManifest)
	mux.HandleFunc("/v1/ai/providers/config", s.aiProviderConfigHandler)
	mux.HandleFunc("/v1/ai/providers/test", s.testAIProvider)
	mux.HandleFunc("/v1/ai/runtime/invoke", s.aiRuntimeInvoke)
	mux.HandleFunc("/v1/ai/azureblob/recommendations", s.azureBlobAIRecommendations)
	mux.HandleFunc("/v1/dspm/catalog", s.dspmCatalog)
	mux.HandleFunc("/v1/dspm/content-scan", s.dspmContentScan)
	mux.HandleFunc("/v1/runtime/audit", s.runtimeAuditHistory)
	mux.HandleFunc("/v1/plugins/contract-test", s.pluginContractTest)
	mux.HandleFunc("/v1/auth/login", s.authLogin)
	mux.HandleFunc("/v1/auth/register", s.authRegister)
	mux.HandleFunc("/v1/auth/users", s.authUsers)
	mux.HandleFunc("/v1/auth/logout", s.authLogout)
	mux.HandleFunc("/v1/auth/session", s.authSession)
	mux.HandleFunc("/v1/portal/options", s.portalOptions)
	mux.HandleFunc("/v1/evidence", s.portalEvidence)
	mux.HandleFunc("/v1/evidence/package", s.enterpriseEvidencePackage)
	mux.HandleFunc("/v1/data-passports/", s.dataPassportByID)
	mux.HandleFunc("/v1/data-passports", s.dataPassports)
	mux.HandleFunc("/v1/search", s.search)
	mux.HandleFunc("/v1/audit/events", s.portalAuditEvents)
	mux.HandleFunc("/v1/actions/history", s.portalActionHistory)
	mux.HandleFunc("/v1/actions/pending", s.portalActionHistory)

	mux.HandleFunc("/healthz", s.health)

	mux.HandleFunc("/v1/agent/runtime/status", s.agentRuntimeStatus)

	mux.HandleFunc("/v1/policy/resolve", s.resolvePolicy)
	mux.HandleFunc("/v1/policy/artifact/trust", s.policyArtifactTrust)
	mux.HandleFunc("/v1/policy/active/", s.getActivePolicy)
	mux.HandleFunc("/v1/policy/simulate/git-unavailable", s.gitUnavailable)
	mux.HandleFunc("/v1/policy/simulate/git-available", s.gitAvailable)

	mux.HandleFunc("/v1/execution/azureblob/start", s.startAzureBlobExecution)
	mux.HandleFunc("/v1/execution/start", s.startExecution)
	mux.HandleFunc("/v1/execution/", s.executionByID)

	mux.HandleFunc("/v1/localstore/tables", s.localStoreTables)
	mux.HandleFunc("/v1/localstore/normalized-files", s.normalizedFilesView)

	mux.HandleFunc("/v1/analytics/quick-scan", s.quickScanAnalytics)
	mux.HandleFunc("/v1/analytics/enterprise-posture", s.enterprisePosture)
	mux.HandleFunc("/v1/playbooks/enterprise", s.enterprisePlaybooks)

	mux.HandleFunc("/v1/actions/plan", s.actionPlan)
	mux.HandleFunc("/v1/guardian/decisions", s.guardianDecisions)

	mux.HandleFunc("/v1/leases/current", s.currentLeaseView)
	mux.HandleFunc("/v1/controlplane/run-next-lease", s.runNextControlPlaneLease)
	mux.HandleFunc("/v1/cache/status", s.cacheStatus)

	mux.HandleFunc("/v1/demo/default-request", s.defaultDemoRequest)
	mux.HandleFunc("/v1/demo/seed-duckdb", s.seedDuckDB)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.Handle("/", http.FileServer(http.Dir("web")))

	return s.WithSessionTouch(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "data-agent-local-resolution",
	})
}

func (s *Server) defaultDemoRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	req, err := loadDefaultExecutionRequest()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, req)
}

func (s *Server) resolvePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var lease policyresolver.PolicyLease
	if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.resolver.ResolveAndActivatePolicy(r.Context(), lease)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getActivePolicy(w http.ResponseWriter, r *http.Request) {
	dataStoreID := strings.TrimPrefix(r.URL.Path, "/v1/policy/active/")
	if dataStoreID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data_store_id required"})
		return
	}

	ctx, ok := s.resolver.GetActivePolicy(dataStoreID)
	if !ok {
		safeID := filepath.Base(dataStoreID)
		if safeID != dataStoreID || safeID == "." || safeID == string(filepath.Separator) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid data_store_id"})
			return
		}
		var diskCtx policyresolver.PolicyContext
		if err := readJSONFile(filepath.Join("data", "active-policy", safeID+".json"), &diskCtx); err == nil && diskCtx.DataStoreID == dataStoreID {
			writeJSON(w, http.StatusOK, &diskCtx)
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active policy not found"})
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

func (s *Server) gitUnavailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.resolver.SetGitUnavailable(true)
	writeJSON(w, http.StatusOK, map[string]bool{"git_unavailable": true})
}

func (s *Server) gitAvailable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.resolver.SetGitUnavailable(false)
	writeJSON(w, http.StatusOK, map[string]bool{"git_unavailable": false})
}

func (s *Server) startExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req execution.ExecutionStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := s.orchestrator.Start(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) executionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/execution/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_id required"})
		return
	}

	executionID := parts[0]

	if len(parts) == 1 {
		status, ok := s.orchestrator.GetStatus(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "execution not found"})
			return
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	switch parts[1] {
	case "summary":
		summary, ok := s.orchestrator.GetSummary(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "summary not found"})
			return
		}
		writeJSON(w, http.StatusOK, summary)

	case "provenance":
		prov, ok := s.orchestrator.GetProvenance(executionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provenance not found"})
			return
		}
		writeJSON(w, http.StatusOK, prov)

	case "manifest":
		if len(parts) < 3 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest endpoint required"})
			return
		}

		switch parts[2] {
		case "summary":
			s.getManifestSummary(w, r, executionID)

		case "details":
			s.getManifestDetails(w, r, executionID)

		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown manifest endpoint"})
		}

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown execution endpoint"})
	}
}

func (s *Server) cacheStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                 "available",
		"policy_cache":           "in_memory_and_disk_backed",
		"workflow_cache":         "resolved",
		"last_known_good_policy": "enabled",
		"note":                   "cache status endpoint",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) seedDuckDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	batch, err := execution.LoadNormalizedBatchFile("demo/normalized/blob-batch-001.json")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := s.store.ReplaceNormalizedFiles(r.Context(), *batch); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if s.portalState != nil {
		cfg := s.azureBlobConfig()
		cfg.DataStoreID = batch.DataStoreID
		source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg))
		containerSeen := map[string]bool{}
		containers := []portalstate.AzureContainer{}
		blobs := make([]portalstate.AzureBlob, 0, len(batch.Records))
		for _, record := range batch.Records {
			containerName := cfg.ContainerName
			if parsed, err := azureblob.ContainerFromFileID(record.FileID); err == nil && parsed != "" {
				containerName = parsed
			}
			if containerName != "" && !containerSeen[containerName] {
				containerSeen[containerName] = true
				containers = append(containers, portalstate.AzureContainer{Name: containerName})
			}
			blobs = append(blobs, portalstate.AzureBlob{
				Container:    containerName,
				Name:         record.Path,
				FileID:       record.FileID,
				SizeBytes:    record.SizeBytes,
				ContentType:  record.ContentType,
				LastModified: record.LastModified,
			})
		}
		s.portalState.UpsertAzureContainers(source, containers)
		s.portalState.UpsertAzureBlobs(source, blobs)
	}

	count, err := s.store.CountNormalizedFiles(r.Context(), batch.BatchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	containerName := ""
	if len(batch.Records) > 0 {
		if parsed, err := azureblob.ContainerFromFileID(batch.Records[0].FileID); err == nil {
			containerName = parsed
		}
	}
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "metadata_scan",
		SubjectID:    batch.BatchID,
		EventType:    "azureblob.sample_batch.seeded",
		Status:       "seeded",
		Severity:     "info",
		SourceSystem: "azure_blob",
		DataStoreID:  batch.DataStoreID,
		JobID:        batch.BatchID,
		Container:    containerName,
		Operation:    "metadata_scan",
		Summary:      "Sample Azure Blob metadata batch was seeded and normalized " + strconv.Itoa(count) + " records.",
		Details: map[string]any{
			"batch_id":        batch.BatchID,
			"lease_id":        batch.LeaseID,
			"records_scanned": len(batch.Records),
			"records_written": count,
			"sample":          true,
		},
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:   "azureblob.sample_batch.seeded",
		Actor:       "portal-user",
		Status:      "seeded",
		TargetType:  "azure_blob_container",
		TargetID:    containerName,
		DataStoreID: batch.DataStoreID,
		Operation:   "metadata_scan",
		DryRun:      true,
		Summary:     "Sample Azure Blob metadata batch was seeded.",
		Details: map[string]any{
			"batch_id":        batch.BatchID,
			"records_written": count,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "seeded",
		"batch_id":        batch.BatchID,
		"data_store_id":   batch.DataStoreID,
		"records_written": count,
		"store":           "local_ephemeral_db_sqlite_backend",
	})
}

func (s *Server) azureBlobConfig() azureblob.Config {
	cfg := s.azureConfigStore.Get()

	if secret, err := s.secrets.Load(azureblob.ConnectionStringSecretName); err == nil && secret != "" {
		cfg.ConnectionString = secret
		if strings.TrimSpace(cfg.StorageAccount) == "" {
			cfg.StorageAccount = storageAccountFromConnectionString(secret)
		}
	} else if strings.TrimSpace(s.cfg.AzureStorageConnectionString) != "" {
		cfg.ConnectionString = strings.TrimSpace(s.cfg.AzureStorageConnectionString)
		if strings.TrimSpace(cfg.StorageAccount) == "" {
			cfg.StorageAccount = storageAccountFromConnectionString(cfg.ConnectionString)
		}
	}

	return cfg
}

func (s *Server) azureBlobCredentialSet(sourceID string) bool {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID != "" && s.secrets.Exists(azureblob.SourceCredentialSecretName(sourceID)) {
		return true
	}
	return s.secrets.Exists(azureblob.ConnectionStringSecretName) || strings.TrimSpace(s.cfg.AzureStorageConnectionString) != ""
}

func (s *Server) azureBlobCredentialSetForConfig(cfg azureblob.Config) bool {
	return s.azureBlobCredentialSet(firstNonBlank(cfg.DataStoreID, cfg.StorageAccount, "azureblob-local"))
}

func (s *Server) testAzureBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scanner, err := s.newAzureBlobScanner(s.azureBlobConfig())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := scanner.TestConnection(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	cfg := s.azureBlobConfig()
	if s.portalState != nil {
		s.portalState.UpsertAzureSource(s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg)))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"source_system":     "azure_blob",
		"container":         cfg.ContainerName,
		"prefix":            cfg.Prefix,
		"data_store_id":     cfg.DataStoreID,
		"batch_id":          cfg.BatchID,
		"action_mode":       cfg.ActionMode,
		"writeback_enabled": cfg.WritebackEnabled,
	})
}

func (s *Server) scanAzureBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scanner, err := s.newAzureBlobScanner(s.azureBlobConfig())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	batch, result, err := scanner.Scan(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := s.store.ReplaceNormalizedFiles(r.Context(), *batch); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if s.portalState != nil {
		cfg := s.azureBlobConfig()
		cfg.DataStoreID = result.DataStoreID
		source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg))
		containerSeen := map[string]bool{}
		containers := []portalstate.AzureContainer{}
		blobs := make([]portalstate.AzureBlob, 0, len(batch.Records))
		for _, record := range batch.Records {
			containerName := result.ContainerName
			if parsed, err := azureblob.ContainerFromFileID(record.FileID); err == nil && parsed != "" {
				containerName = parsed
			}
			if containerName != "" && !containerSeen[containerName] {
				containerSeen[containerName] = true
				containers = append(containers, portalstate.AzureContainer{Name: containerName})
			}
			blobs = append(blobs, portalstate.AzureBlob{
				Container:    containerName,
				Name:         firstNonBlank(record.Path, record.Name),
				FileID:       record.FileID,
				SizeBytes:    record.SizeBytes,
				ContentType:  record.ContentType,
				LastModified: record.LastModified,
			})
		}
		s.portalState.UpsertAzureContainers(source, containers)
		s.portalState.UpsertAzureBlobs(source, blobs)
	}

	count, err := s.store.CountNormalizedFiles(r.Context(), batch.BatchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.recordEvidence(portalstate.EvidenceRecord{
		SubjectType:  "metadata_scan",
		SubjectID:    result.BatchID,
		EventType:    "azureblob.metadata_scan.completed",
		Status:       result.Status,
		Severity:     "info",
		SourceSystem: "azure_blob",
		DataStoreID:  result.DataStoreID,
		JobID:        result.BatchID,
		Container:    result.ContainerName,
		Operation:    "metadata_scan",
		Summary:      "Azure Blob metadata scan completed and normalized " + strconv.Itoa(count) + " records.",
		Details: map[string]any{
			"batch_id":          result.BatchID,
			"prefix":            result.Prefix,
			"records_scanned":   result.RecordsScanned,
			"records_written":   count,
			"action_mode":       result.ActionMode,
			"writeback_enabled": result.Writeback,
		},
	})
	s.recordAudit(portalstate.AuditEvent{
		EventType:   "azureblob.metadata_scan.completed",
		Actor:       "portal-user",
		Status:      result.Status,
		TargetType:  "azure_blob_container",
		TargetID:    result.ContainerName,
		DataStoreID: result.DataStoreID,
		Operation:   "metadata_scan",
		DryRun:      true,
		Summary:     "Azure Blob metadata scan completed.",
		Details: map[string]any{
			"batch_id":        result.BatchID,
			"records_written": count,
		},
	})

	w.Header().Set("X-Everest-Endpoint-Mode", "demo-dev-direct-connector")
	w.Header().Set("X-Everest-Preferred-Endpoint", "/v1/controlplane/run-next-lease")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                        result.Status,
		"source_system":                 "azure_blob",
		"container":                     result.ContainerName,
		"prefix":                        result.Prefix,
		"data_store_id":                 result.DataStoreID,
		"batch_id":                      result.BatchID,
		"records_scanned":               result.RecordsScanned,
		"records_written":               count,
		"store":                         "local_ephemeral_db",
		"action_mode":                   result.ActionMode,
		"writeback":                     result.Writeback,
		"endpoint_mode":                 "demo_dev_direct_connector",
		"preferred_enterprise_endpoint": "/v1/controlplane/run-next-lease",
		"note":                          "Direct Azure Blob scan is retained for demo/dev. Enterprise execution should use the Control Plane lease path.",
	})
}

func (s *Server) azureBlobStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.azureBlobConfig()

	writeJSON(w, http.StatusOK, map[string]any{
		"source_system":         "azure_blob",
		"container_configured":  cfg.ContainerName != "",
		"container":             cfg.ContainerName,
		"prefix":                cfg.Prefix,
		"data_store_id":         cfg.DataStoreID,
		"batch_id":              cfg.BatchID,
		"action_mode":           cfg.ActionMode,
		"writeback_enabled":     cfg.WritebackEnabled,
		"credential_configured": s.azureBlobCredentialSetForConfig(cfg),
	})
}

func (s *Server) listAzureBlobContainers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scanner, err := s.newAzureBlobScanner(s.azureBlobConfig())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	containers, err := scanner.ListContainers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if s.portalState != nil {
		cfg := s.azureBlobConfig()
		source := s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg))
		items := make([]portalstate.AzureContainer, 0, len(containers))
		for _, name := range containers {
			items = append(items, portalstate.AzureContainer{Name: name})
		}
		s.portalState.UpsertAzureContainers(source, items)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"source_system": "azure_blob",
		"containers":    containers,
		"count":         len(containers),
	})
}

func (s *Server) getManifestSummary(w http.ResponseWriter, r *http.Request, executionID string) {
	summary, err := s.store.GetManifestSummary(r.Context(), executionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) getManifestDetails(w http.ResponseWriter, r *http.Request, executionID string) {
	limit := queryInt(r, "limit", 100)
	offset := queryInt(r, "offset", 0)

	details, total, err := s.store.ListManifestDetails(r.Context(), executionID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"execution_id": executionID,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"items":        details,
	})
}

func queryInt(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func (s *Server) publicAzureBlobConfig() azureblob.PublicConfig {
	public := s.azureConfigStore.Public()
	public.CredentialSet = s.azureBlobCredentialSet(firstNonBlank(public.DataStoreID, public.StorageAccount, "azureblob-local"))
	return public
}

func (s *Server) azureBlobConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.publicAzureBlobConfig())
		return

	case http.MethodDelete:
		if !s.authorizeEnterpriseAction(w, r, "azureblob.source.delete", "platform_admin") {
			return
		}
		current := s.azureConfigStore.Get()
		currentSourceID := firstNonBlank(current.DataStoreID, current.StorageAccount, "azureblob-local")
		sourceID := firstNonBlank(r.URL.Query().Get("source_id"), currentSourceID)
		if err := s.secrets.Delete(azureblob.SourceCredentialSecretName(sourceID)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove saved Azure source access: " + err.Error()})
			return
		}
		if sourceID == currentSourceID {
			if err := s.secrets.Delete(azureblob.ConnectionStringSecretName); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove saved Azure account access: " + err.Error()})
				return
			}
			s.azureConfigStore.Update(azureblob.Config{
				ActionMode:       "dry_run",
				WritebackEnabled: "false",
			})
		}
		if s.portalState != nil {
			s.portalState.RemoveAzureSource(sourceID)
			s.recordAudit(portalstate.AuditEvent{
				EventType:  "azureblob.source.deleted",
				Actor:      "portal-user",
				Status:     "deleted",
				TargetType: "azure_blob_source",
				TargetID:   sourceID,
				Summary:    "Saved Azure Blob source was removed from the local portal.",
				DryRun:     false,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "deleted",
			"config": s.publicAzureBlobConfig(),
		})
		return

	case http.MethodPost:
		if !s.authorizeEnterpriseAction(w, r, "azureblob.source.save", "platform_admin", "data_steward") {
			return
		}
		var req azureblob.ConfigureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		requestConnectionString := azureblob.ConnectionStringFromRequest(req)
		sourceID := firstNonBlank(req.SourceID, req.DataStoreID, req.StorageAccount, "azureblob-local")
		credentialOnly := requestConnectionString != "" &&
			strings.TrimSpace(req.ContainerName) == "" &&
			strings.TrimSpace(req.Prefix) == "" &&
			strings.TrimSpace(req.DataStoreID) == "" &&
			strings.TrimSpace(req.BatchID) == "" &&
			strings.TrimSpace(req.ActionMode) == "" &&
			strings.TrimSpace(req.WritebackEnabled) == ""

		if requestConnectionString != "" {
			if err := s.secrets.Save(azureblob.SourceCredentialSecretName(sourceID), requestConnectionString); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store Azure source access securely: " + err.Error()})
				return
			}
			if err := s.secrets.Save(azureblob.ConnectionStringSecretName, requestConnectionString); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store Azure account access securely: " + err.Error()})
				return
			}
			if strings.TrimSpace(req.StorageAccount) == "" {
				req.StorageAccount = storageAccountFromConnectionString(requestConnectionString)
			}
			if strings.TrimSpace(req.DataStoreID) == "" {
				req.DataStoreID = sourceID
			}

			req.ConnectionString = ""
		}

		if credentialOnly {
			current := s.azureConfigStore.Get()
			if strings.TrimSpace(current.StorageAccount) == "" && strings.TrimSpace(req.StorageAccount) != "" {
				current.StorageAccount = strings.TrimSpace(req.StorageAccount)
				s.azureConfigStore.Update(current)
			}
			if s.portalState != nil {
				cfg := s.azureBlobConfig()
				cfg.DataStoreID = firstNonBlank(req.DataStoreID, sourceID)
				cfg.StorageAccount = firstNonBlank(req.StorageAccount, cfg.StorageAccount)
				s.portalState.UpsertAzureSource(s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg)))
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "saved",
				"config": s.publicAzureBlobConfig(),
			})
			return
		}

		current := s.azureConfigStore.Get()
		next := azureblob.ConfigFromRequest(req, current)
		next.ConnectionString = ""
		if strings.TrimSpace(next.StorageAccount) == "" && requestConnectionString != "" {
			next.StorageAccount = storageAccountFromConnectionString(requestConnectionString)
		}

		if strings.TrimSpace(next.ContainerName) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "container is required"})
			return
		}

		if strings.TrimSpace(next.DataStoreID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data_store_id is required"})
			return
		}

		if strings.TrimSpace(next.BatchID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "batch_id is required"})
			return
		}

		if strings.TrimSpace(next.ActionMode) == "" {
			next.ActionMode = "dry_run"
		}

		if strings.TrimSpace(next.WritebackEnabled) == "" {
			next.WritebackEnabled = "false"
		}

		s.azureConfigStore.Update(next)
		if s.portalState != nil {
			cfg := s.azureBlobConfig()
			s.portalState.UpsertAzureSource(s.portalAzureSourceFromConfig(cfg, s.azureBlobCredentialSetForConfig(cfg)))
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "saved",
			"config": s.publicAzureBlobConfig(),
		})
		return

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}

func (s *Server) startAzureBlobExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cfg := s.azureBlobConfig()

	if cfg.DataStoreID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "data_store_id is required"})
		return
	}

	if cfg.BatchID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "batch_id is required"})
		return
	}

	if cfg.ActionMode == "" {
		cfg.ActionMode = "dry_run"
	}

	now := time.Now().UTC()
	suffix := now.Format("20060102-150405")

	executionID := "exec-azureblob-" + suffix
	leaseID := "lease-azureblob-" + suffix
	jobID := "job-azureblob-" + suffix
	envelope := s.buildDACActionRequestEnvelope(r, dacIntentInput{
		Operation:  "azure_blob_execution_start",
		SourceID:   cfg.DataStoreID,
		ActionMode: cfg.ActionMode,
		Container:  cfg.ContainerName,
		Prefix:     cfg.Prefix,
		Metadata: map[string]string{
			"entrypoint":   "azure_blob_execution_start",
			"execution_id": executionID,
			"job_id":       jobID,
			"batch_id":     cfg.BatchID,
		},
	})

	req := execution.ExecutionStartRequest{
		ExecutionID: executionID,
		Lease: execution.ExecutionLease{
			LeaseID:               leaseID,
			JobID:                 jobID,
			DataStoreID:           cfg.DataStoreID,
			CompiledPolicyVersion: s.cfg.ActiveCompiledPolicyVersion,
			CompiledGitTag:        s.cfg.ActiveCompiledPolicyTag,
			CompiledGitCommit:     "",
			CompiledArtifactHash:  s.cfg.ActiveCompiledPolicyHash,
			WorkflowVersion:       s.cfg.ActiveWorkflowVersion,
			WorkflowGitTag:        s.cfg.ActiveWorkflowTag,
		},
		ExecutionState: execution.ExecutionState{
			DataStoreID:     cfg.DataStoreID,
			ProcessingTier:  s.cfg.DefaultProcessingTier,
			ProcessingMode:  s.cfg.DefaultProcessingMode,
			PolicyVersion:   s.cfg.ActiveCompiledPolicyVersion,
			WorkflowVersion: s.cfg.ActiveWorkflowVersion,
			Settings: execution.ExecutionSettings{
				EnableContentIntelligence: true,
				EnableGuardianEvaluation:  true,
				EnableWriteBackActions:    cfg.WritebackEnabled == "true",
				EnableElasticsearchIndex:  true,
			},
			OperationalConfig: execution.ExecutionOperationalConfig{
				BatchSize:                    1000,
				LocalExecutionRequired:       true,
				GuardianBeforeActionRequired: true,
				ActionMode:                   cfg.ActionMode,
			},
		},
		BatchID:         cfg.BatchID,
		RequestEnvelope: &envelope,
	}

	result, err := s.orchestrator.Start(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if s.portalState != nil {
		s.portalState.UpsertJob(portalstate.Job{
			JobID:       jobID,
			DataStoreID: cfg.DataStoreID,
			Status:      result.Status.Status,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
