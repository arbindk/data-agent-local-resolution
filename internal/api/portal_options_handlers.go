package api

import (
	"net/http"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) portalOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if s.portalState == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"version":    portalstate.Version,
			"next_steps": []string{"Portal state store is not configured."},
		})
		return
	}

	cfg := s.azureBlobConfig()
	credentialSet := s.azureBlobCredentialSetForConfig(cfg)
	if azureBlobConfigHasSource(cfg) || credentialSet {
		s.portalState.UpsertAzureSource(s.portalAzureSourceFromConfig(cfg, credentialSet))
	}
	s.portalState.UpsertAIProvider(s.portalAIProviderFromConfig())

	writeJSON(w, http.StatusOK, s.portalState.Options())
}

func azureBlobConfigHasSource(cfg azureblob.Config) bool {
	return firstNonBlank(cfg.StorageAccount, cfg.ContainerName, cfg.Prefix, cfg.DataStoreID, cfg.BatchID) != ""
}

func (s *Server) portalAzureSourceFromConfig(cfg azureblob.Config, credentialSet bool) portalstate.AzureSource {
	sourceID := firstNonBlank(cfg.DataStoreID, "azureblob-local")
	storageAccount := firstNonBlank(cfg.StorageAccount, storageAccountFromConnectionString(cfg.ConnectionString))
	return portalstate.AzureSource{
		SourceID:         sourceID,
		DataStoreID:      sourceID,
		DisplayName:      azureSourceDisplayName(storageAccount, cfg.ContainerName, sourceID),
		ConnectorID:      "azureblob",
		DeploymentMode:   "cloud_connected",
		StorageAccount:   storageAccount,
		Container:        cfg.ContainerName,
		Prefix:           cfg.Prefix,
		ActionMode:       cfg.ActionMode,
		WritebackEnabled: cfg.WritebackEnabled,
		CredentialSet:    credentialSet,
	}
}

func (s *Server) portalAIProviderFromConfig() portalstate.AIProvider {
	cfg := s.publicAIProviderConfig()
	return portalstate.AIProvider{
		ProviderID:    cfg.ProviderID,
		Model:         cfg.Model,
		NetworkMode:   cfg.NetworkMode,
		Configured:    true,
		CredentialSet: cfg.CredentialSet,
	}
}

func storageAccountFromConnectionString(connectionString string) string {
	for _, part := range strings.Split(connectionString, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "AccountName") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func azureSourceDisplayName(storageAccount string, container string, fallback string) string {
	if strings.TrimSpace(storageAccount) != "" && strings.TrimSpace(container) != "" {
		return strings.TrimSpace(storageAccount) + " / " + strings.TrimSpace(container)
	}
	if strings.TrimSpace(container) != "" {
		return "Azure Blob / " + strings.TrimSpace(container)
	}
	return firstNonBlank(fallback, "Azure Blob source")
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
