package providers

import (
	"fmt"
	"strings"
	"sync"
)

const DefaultProviderID = "everest-local-policy-copilot"

type Config struct {
	ProviderID                string
	EndpointURL               string
	Model                     string
	DeploymentName            string
	AuthScheme                string
	NetworkMode               string
	AutomationMode            string
	RedactionMode             string
	PromptAuditEnabled        bool
	EvidenceGroundingRequired bool
	AllowHITLSubmission       bool
	AllowAutonomousApply      bool
}

type ConfigureRequest struct {
	ProviderID                string `json:"provider_id"`
	EndpointURL               string `json:"endpoint_url"`
	Model                     string `json:"model"`
	DeploymentName            string `json:"deployment_name"`
	AuthScheme                string `json:"auth_scheme"`
	NetworkMode               string `json:"network_mode"`
	AutomationMode            string `json:"automation_mode"`
	RedactionMode             string `json:"redaction_mode"`
	PromptAuditEnabled        bool   `json:"prompt_audit_enabled"`
	EvidenceGroundingRequired bool   `json:"evidence_grounding_required"`
	AllowHITLSubmission       bool   `json:"allow_hitl_submission"`
	AllowAutonomousApply      bool   `json:"allow_autonomous_apply"`
	APIKey                    string `json:"api_key,omitempty"`
	BearerToken               string `json:"bearer_token,omitempty"`
}

type PublicConfig struct {
	ProviderID                string `json:"provider_id"`
	EndpointURL               string `json:"endpoint_url"`
	Model                     string `json:"model"`
	DeploymentName            string `json:"deployment_name"`
	AuthScheme                string `json:"auth_scheme"`
	NetworkMode               string `json:"network_mode"`
	AutomationMode            string `json:"automation_mode"`
	RedactionMode             string `json:"redaction_mode"`
	PromptAuditEnabled        bool   `json:"prompt_audit_enabled"`
	EvidenceGroundingRequired bool   `json:"evidence_grounding_required"`
	AllowHITLSubmission       bool   `json:"allow_hitl_submission"`
	AllowAutonomousApply      bool   `json:"allow_autonomous_apply"`
	CredentialSet             bool   `json:"credential_set"`
}

type RuntimeConfigStore struct {
	mu  sync.RWMutex
	cfg Config
}

func NewRuntimeConfigStore(initial Config) *RuntimeConfigStore {
	return &RuntimeConfigStore{cfg: Defaults(initial)}
}

func (s *RuntimeConfigStore) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *RuntimeConfigStore) Update(next Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = Defaults(next)
}

func (s *RuntimeConfigStore) Public(credentialSet bool) PublicConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := Defaults(s.cfg)
	return PublicConfig{
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
		CredentialSet:             credentialSet,
	}
}

func ConfigFromRequest(req ConfigureRequest, existing Config) Config {
	next := existing

	if strings.TrimSpace(req.ProviderID) != "" {
		next.ProviderID = strings.TrimSpace(req.ProviderID)
	}
	if strings.TrimSpace(req.EndpointURL) != "" {
		next.EndpointURL = strings.TrimSpace(req.EndpointURL)
	}
	if strings.TrimSpace(req.Model) != "" {
		next.Model = strings.TrimSpace(req.Model)
	}
	if strings.TrimSpace(req.DeploymentName) != "" {
		next.DeploymentName = strings.TrimSpace(req.DeploymentName)
	}
	if strings.TrimSpace(req.AuthScheme) != "" {
		next.AuthScheme = strings.TrimSpace(req.AuthScheme)
	}
	if strings.TrimSpace(req.NetworkMode) != "" {
		next.NetworkMode = strings.TrimSpace(req.NetworkMode)
	}
	if strings.TrimSpace(req.AutomationMode) != "" {
		next.AutomationMode = strings.TrimSpace(req.AutomationMode)
	}
	if strings.TrimSpace(req.RedactionMode) != "" {
		next.RedactionMode = strings.TrimSpace(req.RedactionMode)
	}

	next.PromptAuditEnabled = req.PromptAuditEnabled
	next.EvidenceGroundingRequired = req.EvidenceGroundingRequired
	next.AllowHITLSubmission = req.AllowHITLSubmission
	next.AllowAutonomousApply = req.AllowAutonomousApply

	return Defaults(next)
}

func Defaults(cfg Config) Config {
	if strings.TrimSpace(cfg.ProviderID) == "" {
		cfg.ProviderID = DefaultProviderID
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "local-policy-rules-v1"
	}
	if strings.TrimSpace(cfg.AuthScheme) == "" {
		cfg.AuthScheme = "none"
	}
	if strings.TrimSpace(cfg.NetworkMode) == "" {
		cfg.NetworkMode = "air_gapped"
	}
	if strings.TrimSpace(cfg.AutomationMode) == "" {
		cfg.AutomationMode = "recommend_only"
	}
	if strings.TrimSpace(cfg.RedactionMode) == "" {
		cfg.RedactionMode = "metadata_only"
	}
	return cfg
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ProviderID) == "" {
		return fmt.Errorf("provider_id is required")
	}
	if strings.TrimSpace(c.NetworkMode) == "" {
		return fmt.Errorf("network_mode is required")
	}
	if strings.TrimSpace(c.AutomationMode) == "" {
		return fmt.Errorf("automation_mode is required")
	}
	if c.AllowAutonomousApply {
		return fmt.Errorf("autonomous AI apply is not supported by the Everest safety model")
	}
	return nil
}

func SecretName(providerID string, field string) string {
	providerID = sanitize(providerID)
	field = sanitize(field)
	if providerID == "" {
		providerID = DefaultProviderID
	}
	if field == "" {
		field = "credential"
	}
	return "ai/providers/" + providerID + "/" + field
}

func sanitize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	return strings.Trim(b.String(), "_")
}
