package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	aicatalog "everest.local/data-agent-policy-resolver/internal/ai/catalog"
	aiproviders "everest.local/data-agent-policy-resolver/internal/ai/providers"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

func (s *Server) aiProviderCatalog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := aicatalog.LoadCatalog(aiProviderManifestDirs()...)
		if s.portalState != nil {
			for _, item := range items {
				s.portalState.UpsertAIProvider(portalstate.AIProvider{
					ProviderID:      item.ProviderID,
					DisplayName:     item.DisplayName,
					Status:          item.Status,
					Runtime:         item.Runtime.Kind,
					DeploymentModes: item.DeploymentModes,
				})
			}
			s.portalState.UpsertAIProvider(s.portalAIProviderFromConfig())
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":              "ok",
			"version":             "ai_provider_manifest_v1",
			"count":               len(items),
			"items":               items,
			"configured_provider": s.publicAIProviderConfig(),
			"deployment_modes": []string{
				"cloud_connected",
				"private_network",
				"air_gapped",
				"hybrid",
			},
			"automation_modes": []string{
				"recommend_only",
				"submit_hitl_approvals",
			},
			"custom_ai_provider_contract": map[string]any{
				"manifest_schema":              "contracts/ai_provider_manifest.schema.json",
				"runtime_request":              "contracts/ai_provider_request.schema.json",
				"runtime_response":             "contracts/ai_provider_response.schema.json",
				"runtime_protocol":             "everest_ai_provider_v1",
				"structured_json_required":     true,
				"evidence_grounding_required":  true,
				"prompt_audit_required":        true,
				"autonomous_apply_supported":   false,
				"hitl_before_action_execution": true,
			},
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) validateAIProviderManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var manifest aicatalog.AIProviderManifest
	if err := jsonDecode(r, &manifest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := aicatalog.ValidateManifest(manifest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "invalid",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "valid",
		"provider_id": manifest.ProviderID,
		"provider":    manifest,
		"next_steps": []string{
			"Place the manifest under ai-providers, demo/ai-providers, or EVEREST_AI_PROVIDER_MANIFEST_DIRS.",
			"Configure endpoint, model, auth scheme, and customer network mode.",
			"Validate structured JSON recommendation output before enabling HITL submission.",
			"Keep autonomous apply disabled; Everest routes executable actions through Guardian and HITL.",
		},
	})
}

func (s *Server) aiProviderConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.publicAIProviderConfig())
		return

	case http.MethodPost:
		var req aiproviders.ConfigureRequest
		if err := jsonDecode(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}

		current := s.aiProviderConfig()
		next := aiproviders.ConfigFromRequest(req, current)
		next = s.withAIProviderManifestDefaults(next)

		if err := s.validateAIProviderConfig(next); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		manifest, _ := aicatalog.FindProvider(aicatalog.LoadCatalog(aiProviderManifestDirs()...), next.ProviderID)
		credentialReady := s.aiCredentialReady(next) ||
			(next.AuthScheme == "api_key" && strings.TrimSpace(req.APIKey) != "") ||
			(next.AuthScheme == "bearer_token" && strings.TrimSpace(req.BearerToken) != "")
		if missing := s.missingAIProviderConfigFields(manifest, next, credentialReady); len(missing) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required AI provider config: " + strings.Join(missing, ", ")})
			return
		}

		if strings.TrimSpace(req.APIKey) != "" {
			if err := s.secrets.Save(aiproviders.SecretName(next.ProviderID, "api_key"), strings.TrimSpace(req.APIKey)); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store AI provider API key securely: " + err.Error()})
				return
			}
		}
		if strings.TrimSpace(req.BearerToken) != "" {
			if err := s.secrets.Save(aiproviders.SecretName(next.ProviderID, "bearer_token"), strings.TrimSpace(req.BearerToken)); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store AI provider bearer token securely: " + err.Error()})
				return
			}
		}

		s.aiConfigStore.Update(next)
		if s.portalState != nil {
			s.portalState.UpsertAIProvider(s.portalAIProviderFromConfig())
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "saved",
			"config": s.publicAIProviderConfig(),
			"controls": []string{
				"credentials_stored_in_windows_credential_manager",
				"prompt_audit_policy_recorded",
				"evidence_grounding_policy_recorded",
				"autonomous_apply_blocked",
			},
		})
		return

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}

func (s *Server) testAIProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cfg := s.aiProviderConfig()
	providedAPIKey := false
	providedBearerToken := false
	if r.ContentLength != 0 {
		var req aiproviders.ConfigureRequest
		if err := jsonDecode(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
			return
		}
		providedAPIKey = strings.TrimSpace(req.APIKey) != ""
		providedBearerToken = strings.TrimSpace(req.BearerToken) != ""
		cfg = aiproviders.ConfigFromRequest(req, cfg)
	}
	cfg = s.withAIProviderManifestDefaults(cfg)

	items := aicatalog.LoadCatalog(aiProviderManifestDirs()...)
	manifest, known := aicatalog.FindProvider(items, cfg.ProviderID)
	credentialReady := s.aiCredentialReady(cfg) ||
		(cfg.AuthScheme == "api_key" && providedAPIKey) ||
		(cfg.AuthScheme == "bearer_token" && providedBearerToken)
	checks := []map[string]string{}

	status := "ok"
	addCheck := func(name string, result string, detail string) {
		if result == "fail" && status == "ok" {
			status = "needs_configuration"
		}
		checks = append(checks, map[string]string{
			"name":   name,
			"result": result,
			"detail": detail,
		})
	}

	if !known {
		addCheck("provider_manifest", "fail", "Provider is not present in the AI provider catalog.")
	} else {
		addCheck("provider_manifest", "pass", manifest.DisplayName+" manifest loaded.")
	}

	if known && !aicatalog.SupportsDeploymentMode(manifest, cfg.NetworkMode) {
		addCheck("deployment_mode", "fail", cfg.ProviderID+" does not list "+cfg.NetworkMode+" support.")
	} else {
		addCheck("deployment_mode", "pass", cfg.NetworkMode+" is allowed.")
	}

	for _, missing := range s.missingAIProviderConfigFields(manifest, cfg, credentialReady) {
		addCheck("required_config", "fail", missing+" is required.")
	}
	if status == "ok" {
		addCheck("required_config", "pass", "Required provider fields are configured.")
	}

	if cfg.AllowAutonomousApply {
		addCheck("autonomous_apply", "fail", "Autonomous AI apply is blocked by the Everest safety model.")
	} else {
		addCheck("autonomous_apply", "pass", "AI can recommend and prepare HITL; executable actions remain gated.")
	}

	if !credentialReady {
		addCheck("credential", "fail", "Credential is not configured for auth scheme "+cfg.AuthScheme+".")
	} else {
		addCheck("credential", "pass", "Credential posture is ready for "+cfg.AuthScheme+".")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            status,
		"provider_id":       cfg.ProviderID,
		"model":             cfg.Model,
		"deployment_name":   cfg.DeploymentName,
		"network_mode":      cfg.NetworkMode,
		"runtime":           manifest.Runtime,
		"connectivity_test": "configuration_only_no_customer_prompt_sent",
		"checks":            checks,
		"generated_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

func aiProviderManifestDirs() []string {
	dirs := []string{
		"ai-providers",
		"demo/ai-providers",
	}

	if raw := strings.TrimSpace(os.Getenv("EVEREST_AI_PROVIDER_MANIFEST_DIRS")); raw != "" {
		for _, part := range strings.Split(raw, ";") {
			part = strings.TrimSpace(part)
			if part != "" {
				dirs = append(dirs, part)
			}
		}
	}

	return dirs
}

func (s *Server) aiProviderConfig() aiproviders.Config {
	return aiproviders.Defaults(s.aiConfigStore.Get())
}

func (s *Server) publicAIProviderConfig() aiproviders.PublicConfig {
	cfg := s.withAIProviderManifestDefaults(s.aiProviderConfig())
	return aiproviders.PublicConfig{
		ProviderID:                cfg.ProviderID,
		EndpointURL:               cfg.EndpointURL,
		Model:                     cfg.Model,
		DeploymentName:            cfg.DeploymentName,
		AuthScheme:                cfg.AuthScheme,
		NetworkMode:               cfg.NetworkMode,
		AutomationMode:            cfg.AutomationMode,
		RedactionMode:             cfg.RedactionMode,
		PromptAuditEnabled:        cfg.PromptAuditEnabled,
		EvidenceGroundingRequired: cfg.EvidenceGroundingRequired,
		AllowHITLSubmission:       cfg.AllowHITLSubmission,
		AllowAutonomousApply:      cfg.AllowAutonomousApply,
		CredentialSet:             s.aiCredentialReady(cfg),
	}
}

func (s *Server) aiCredentialReady(cfg aiproviders.Config) bool {
	switch strings.TrimSpace(cfg.AuthScheme) {
	case "", "none", "managed_identity":
		return true
	case "api_key":
		return s.secrets.Exists(aiproviders.SecretName(cfg.ProviderID, "api_key"))
	case "bearer_token":
		return s.secrets.Exists(aiproviders.SecretName(cfg.ProviderID, "bearer_token"))
	default:
		return s.secrets.Exists(aiproviders.SecretName(cfg.ProviderID, "api_key")) ||
			s.secrets.Exists(aiproviders.SecretName(cfg.ProviderID, "bearer_token"))
	}
}

func (s *Server) withAIProviderManifestDefaults(cfg aiproviders.Config) aiproviders.Config {
	cfg = aiproviders.Defaults(cfg)
	manifest, ok := aicatalog.FindProvider(aicatalog.LoadCatalog(aiProviderManifestDirs()...), cfg.ProviderID)
	if !ok {
		return cfg
	}

	if strings.TrimSpace(cfg.EndpointURL) == "" &&
		manifest.Runtime.Endpoint != "" &&
		manifest.Runtime.Endpoint != "customer_configured" {
		cfg.EndpointURL = manifest.Runtime.Endpoint
	}

	if (strings.TrimSpace(cfg.Model) == "" || (cfg.ProviderID != aiproviders.DefaultProviderID && cfg.Model == "local-policy-rules-v1")) &&
		len(manifest.ModelOptions) > 0 {
		cfg.Model = manifest.ModelOptions[0].ModelID
	}

	if strings.TrimSpace(cfg.AuthScheme) == "" && len(manifest.AuthSchemes) > 0 {
		cfg.AuthScheme = manifest.AuthSchemes[0].Scheme
	}

	return aiproviders.Defaults(cfg)
}

func (s *Server) validateAIProviderConfig(cfg aiproviders.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	manifest, ok := aicatalog.FindProvider(aicatalog.LoadCatalog(aiProviderManifestDirs()...), cfg.ProviderID)
	if !ok {
		return httpError("unknown AI provider: " + cfg.ProviderID)
	}
	if !aicatalog.SupportsDeploymentMode(manifest, cfg.NetworkMode) {
		return httpError("AI provider " + cfg.ProviderID + " does not support deployment mode " + cfg.NetworkMode)
	}
	if cfg.AllowAutonomousApply || !manifest.Safety.AutonomousApplyAllowed && cfg.AllowAutonomousApply {
		return httpError("autonomous AI apply is not allowed for provider " + cfg.ProviderID)
	}
	if cfg.AutomationMode == "submit_hitl_approvals" && !cfg.AllowHITLSubmission {
		return httpError("submit_hitl_approvals requires allow_hitl_submission=true")
	}
	return nil
}

func (s *Server) missingAIProviderConfigFields(manifest aicatalog.AIProviderManifest, cfg aiproviders.Config, credentialReady bool) []string {
	if manifest.ProviderID == "" {
		return nil
	}

	missing := []string{}
	for _, field := range manifest.ConfigFields {
		if !field.Required {
			continue
		}
		switch field.Name {
		case "endpoint_url":
			if strings.TrimSpace(cfg.EndpointURL) == "" {
				missing = append(missing, field.Name)
			}
		case "model":
			if strings.TrimSpace(cfg.Model) == "" {
				missing = append(missing, field.Name)
			}
		case "deployment_name":
			if strings.TrimSpace(cfg.DeploymentName) == "" {
				missing = append(missing, field.Name)
			}
		case "auth_scheme":
			if strings.TrimSpace(cfg.AuthScheme) == "" {
				missing = append(missing, field.Name)
			}
		case "api_key":
			if cfg.AuthScheme == "api_key" && !credentialReady {
				missing = append(missing, field.Name)
			}
		case "bearer_token":
			if cfg.AuthScheme == "bearer_token" && !credentialReady {
				missing = append(missing, field.Name)
			}
		}
	}
	return missing
}

type httpError string

func (e httpError) Error() string {
	return string(e)
}
