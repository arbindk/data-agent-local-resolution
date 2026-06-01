package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
	"everest.local/data-agent-policy-resolver/internal/secretstore"
)

type Server struct {
	cfg              config.Config
	resolver         *policyresolver.Resolver
	orchestrator     *execution.Orchestrator
	store            *duckdbstore.Store
	azureConfigStore *azureblob.RuntimeConfigStore
	secrets          *secretstore.Store
}

func NewServer(cfg config.Config, resolver *policyresolver.Resolver, orchestrator *execution.Orchestrator, store *duckdbstore.Store) *Server {
	initialAzureConfig := azureblob.Config{
		ConnectionString: "",
		ContainerName:    cfg.AzureBlobContainer,
		Prefix:           cfg.AzureBlobPrefix,
		DataStoreID:      cfg.AzureBlobDataStoreID,
		BatchID:          cfg.AzureBlobBatchID,
		ActionMode:       cfg.AzureActionMode,
		WritebackEnabled: cfg.AzureWritebackEnabled,
	}

	secretStore := secretstore.New()

	if cfg.AzureStorageConnectionString != "" && !secretStore.Exists(azureblob.ConnectionStringSecretName) {
		_ = secretStore.Save(azureblob.ConnectionStringSecretName, cfg.AzureStorageConnectionString)
	}

	return &Server{
		cfg:              cfg,
		resolver:         resolver,
		orchestrator:     orchestrator,
		store:            store,
		azureConfigStore: azureblob.NewRuntimeConfigStore(initialAzureConfig),
		secrets:          secretStore,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/connectors/azureblob/config", s.azureBlobConfigHandler)
	mux.HandleFunc("/v1/connectors/azureblob/test", s.testAzureBlob)
	mux.HandleFunc("/v1/connectors/azureblob/scan", s.scanAzureBlob)
	mux.HandleFunc("/v1/connectors/azureblob/status", s.azureBlobStatus)
	mux.HandleFunc("/v1/connectors/azureblob/containers", s.listAzureBlobContainers)

	mux.HandleFunc("/healthz", s.health)

	mux.HandleFunc("/v1/policy/resolve", s.resolvePolicy)
	mux.HandleFunc("/v1/policy/active/", s.getActivePolicy)
	mux.HandleFunc("/v1/policy/simulate/git-unavailable", s.gitUnavailable)
	mux.HandleFunc("/v1/policy/simulate/git-available", s.gitAvailable)

	mux.HandleFunc("/v1/execution/start", s.startExecution)
	mux.HandleFunc("/v1/execution/", s.executionByID)
	mux.HandleFunc("/v1/cache/status", s.cacheStatus)

	mux.HandleFunc("/v1/demo/default-request", s.defaultDemoRequest)
	mux.HandleFunc("/v1/demo/seed-duckdb", s.seedDuckDB)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	return mux
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
		"workflow_cache":         "demo_resolved",
		"last_known_good_policy": "enabled",
		"note":                   "hackathon demo cache status endpoint",
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

	count, err := s.store.CountNormalizedFiles(r.Context(), batch.BatchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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
	}

	return cfg
}

func (s *Server) testAzureBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scanner, err := azureblob.NewScanner(s.azureBlobConfig())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := scanner.TestConnection(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	cfg := s.azureBlobConfig()
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

	scanner, err := azureblob.NewScanner(s.azureBlobConfig())
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

	count, err := s.store.CountNormalizedFiles(r.Context(), batch.BatchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          result.Status,
		"source_system":   "azure_blob",
		"container":       result.ContainerName,
		"prefix":          result.Prefix,
		"data_store_id":   result.DataStoreID,
		"batch_id":        result.BatchID,
		"records_scanned": result.RecordsScanned,
		"records_written": count,
		"store":           "local_ephemeral_db",
		"action_mode":     result.ActionMode,
		"writeback":       result.Writeback,
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
		"credential_configured": s.secrets.Exists(azureblob.ConnectionStringSecretName),
	})
}

func (s *Server) listAzureBlobContainers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	scanner, err := azureblob.NewScanner(s.azureBlobConfig())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	containers, err := scanner.ListContainers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
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
	public.CredentialSet = s.secrets.Exists(azureblob.ConnectionStringSecretName)
	return public
}

func (s *Server) azureBlobConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.publicAzureBlobConfig())
		return

	case http.MethodPost:
		var req azureblob.ConfigureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		if strings.TrimSpace(req.ConnectionString) != "" {
			if err := s.secrets.Save(azureblob.ConnectionStringSecretName, strings.TrimSpace(req.ConnectionString)); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store Azure credential securely: " + err.Error()})
				return
			}

			req.ConnectionString = ""
		}

		current := s.azureConfigStore.Get()
		next := azureblob.ConfigFromRequest(req, current)
		next.ConnectionString = ""

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
