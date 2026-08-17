package api

import (
	"net/http"
	"strings"

	aicatalog "everest.local/data-agent-policy-resolver/internal/ai/catalog"
	aiproviders "everest.local/data-agent-policy-resolver/internal/ai/providers"
	airuntime "everest.local/data-agent-policy-resolver/internal/ai/runtime"
)

func (s *Server) aiRuntimeInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req airuntime.Request
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg := s.aiProviderConfig()
	if strings.TrimSpace(req.ProviderID) != "" {
		cfg.ProviderID = strings.TrimSpace(req.ProviderID)
	}
	if strings.TrimSpace(req.Model) != "" {
		cfg.Model = strings.TrimSpace(req.Model)
	}
	cfg = s.withAIProviderManifestDefaults(cfg)
	if strings.TrimSpace(req.ProviderID) == "" {
		req.ProviderID = cfg.ProviderID
	}

	manifest, ok := findAIProviderManifest(req.ProviderID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "AI provider manifest not found: " + req.ProviderID})
		return
	}

	credential := s.aiRuntimeCredential(cfg)
	resp, err := airuntime.Invoke(r.Context(), manifest, cfg, req, credential)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.runtimeAudit.Add(runtimeAuditRecord(
		"ai",
		resp.ProviderID,
		resp.Task,
		resp.Status,
		resp.RuntimeKind,
		resp.ContractVersion,
		false,
		true,
		resp.DurationMillis,
		resp.Error,
		resp.Controls,
	))

	writeJSON(w, http.StatusOK, resp)
}

func findAIProviderManifest(providerID string) (aicatalog.AIProviderManifest, bool) {
	items := aicatalog.LoadCatalog(aiProviderManifestDirs()...)
	for _, item := range items {
		if item.ProviderID == providerID {
			return item, true
		}
	}
	return aicatalog.AIProviderManifest{}, false
}

func (s *Server) aiRuntimeCredential(cfg aiproviders.Config) string {
	switch strings.TrimSpace(cfg.AuthScheme) {
	case "api_key":
		secret, _ := s.secrets.Load(aiproviders.SecretName(cfg.ProviderID, "api_key"))
		return secret
	case "bearer_token":
		secret, _ := s.secrets.Load(aiproviders.SecretName(cfg.ProviderID, "bearer_token"))
		return secret
	default:
		return ""
	}
}
