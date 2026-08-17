package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	StatusAvailable    = "available"
	StatusConfigurable = "configurable"
	StatusPlanned      = "planned"
	StatusCustom       = "custom_manifest"
)

type AIProviderManifest struct {
	ProviderID      string               `json:"provider_id"`
	DisplayName     string               `json:"display_name"`
	ProviderType    string               `json:"provider_type"`
	Vendor          string               `json:"vendor"`
	Version         string               `json:"version"`
	Status          string               `json:"status"`
	Description     string               `json:"description"`
	DeploymentModes []string             `json:"deployment_modes"`
	NetworkModes    []string             `json:"network_modes"`
	Runtime         AIProviderRuntime    `json:"runtime"`
	AuthSchemes     []AuthScheme         `json:"auth_schemes"`
	ConfigFields    []ConfigField        `json:"config_fields"`
	ModelOptions    []ModelOption        `json:"model_options"`
	Capabilities    AIProviderCapability `json:"capabilities"`
	Safety          AISafetyPosture      `json:"safety"`
	DataPolicy      AIDataPolicy         `json:"data_policy"`
	Support         SupportPosture       `json:"support"`
	LoadedFrom      string               `json:"loaded_from,omitempty"`
}

type AIProviderRuntime struct {
	Kind        string   `json:"kind"`
	Protocol    string   `json:"protocol,omitempty"`
	Endpoint    string   `json:"endpoint,omitempty"`
	HealthCheck string   `json:"health_check,omitempty"`
	Entrypoint  string   `json:"entrypoint,omitempty"`
	Platforms   []string `json:"platforms"`
}

type AuthScheme struct {
	Scheme       string   `json:"scheme"`
	DisplayName  string   `json:"display_name"`
	SecretFields []string `json:"secret_fields,omitempty"`
	OfflineReady bool     `json:"offline_ready"`
}

type ConfigField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Secret      bool     `json:"secret"`
	Options     []string `json:"options,omitempty"`
	Description string   `json:"description,omitempty"`
}

type ModelOption struct {
	ModelID                string   `json:"model_id"`
	DisplayName            string   `json:"display_name"`
	Capabilities           []string `json:"capabilities"`
	DeploymentNameRequired bool     `json:"deployment_name_required"`
}

type AIProviderCapability struct {
	Chat                bool     `json:"chat"`
	Recommendations     bool     `json:"recommendations"`
	Classification      bool     `json:"classification"`
	Summarization       bool     `json:"summarization"`
	PolicyExplanation   bool     `json:"policy_explanation"`
	ActionPlanning      bool     `json:"action_planning"`
	PromptGrounding     bool     `json:"prompt_grounding"`
	StructuredJSON      bool     `json:"structured_json"`
	ToolCalling         bool     `json:"tool_calling"`
	Streaming           bool     `json:"streaming"`
	Embeddings          bool     `json:"embeddings"`
	OfflineInference    bool     `json:"offline_inference"`
	SupportedTasks      []string `json:"supported_tasks"`
	SupportedModalities []string `json:"supported_modalities"`
}

type AISafetyPosture struct {
	AdvisoryOnlyDefault       bool     `json:"advisory_only_default"`
	HITLSubmissionAllowed     bool     `json:"hitl_submission_allowed"`
	AutonomousApplyAllowed    bool     `json:"autonomous_apply_allowed"`
	PromptAuditRequired       bool     `json:"prompt_audit_required"`
	EvidenceGroundingRequired bool     `json:"evidence_grounding_required"`
	RedactionSupported        bool     `json:"redaction_supported"`
	AllowedAutomationModes    []string `json:"allowed_automation_modes"`
	BlockedActions            []string `json:"blocked_actions"`
}

type AIDataPolicy struct {
	CustomerDataLeavesEnvironment bool     `json:"customer_data_leaves_environment"`
	SupportsPrivateEndpoint       bool     `json:"supports_private_endpoint"`
	SupportsAirGap                bool     `json:"supports_air_gap"`
	RetentionNotes                []string `json:"retention_notes"`
	ResidencyNotes                []string `json:"residency_notes"`
	DataHandlingNotes             []string `json:"data_handling_notes"`
}

type SupportPosture struct {
	Owner             string   `json:"owner"`
	SupportLevel      string   `json:"support_level"`
	TestRequirements  []string `json:"test_requirements"`
	CustomerOwnedCode bool     `json:"customer_owned_code"`
}

func LoadCatalog(manifestDirs ...string) []AIProviderManifest {
	items := builtinManifests()

	for _, dir := range manifestDirs {
		custom, err := LoadManifestsFromDir(dir)
		if err != nil {
			continue
		}
		items = append(items, custom...)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return statusRank(items[i].Status) < statusRank(items[j].Status)
		}
		return items[i].DisplayName < items[j].DisplayName
	})

	return items
}

func LoadManifestsFromDir(dir string) ([]AIProviderManifest, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	items := []AIProviderManifest{}
	for _, path := range matches {
		manifest, err := LoadManifest(path)
		if err != nil {
			continue
		}
		manifest.LoadedFrom = path
		if manifest.Status == "" {
			manifest.Status = StatusCustom
		}
		items = append(items, manifest)
	}

	return items, nil
}

func LoadManifest(path string) (AIProviderManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AIProviderManifest{}, err
	}

	var manifest AIProviderManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return AIProviderManifest{}, err
	}

	if err := ValidateManifest(manifest); err != nil {
		return AIProviderManifest{}, err
	}

	return manifest, nil
}

func ValidateManifest(manifest AIProviderManifest) error {
	if strings.TrimSpace(manifest.ProviderID) == "" {
		return errors.New("provider_id is required")
	}
	if strings.TrimSpace(manifest.DisplayName) == "" {
		return errors.New("display_name is required")
	}
	if strings.TrimSpace(manifest.ProviderType) == "" {
		return errors.New("provider_type is required")
	}
	if strings.TrimSpace(manifest.Runtime.Kind) == "" {
		return errors.New("runtime.kind is required")
	}
	if len(manifest.DeploymentModes) == 0 {
		return errors.New("deployment_modes is required")
	}
	if !manifest.Capabilities.Chat &&
		!manifest.Capabilities.Recommendations &&
		!manifest.Capabilities.Classification &&
		!manifest.Capabilities.Summarization &&
		!manifest.Capabilities.PolicyExplanation &&
		!manifest.Capabilities.ActionPlanning {
		return errors.New("at least one AI capability is required")
	}
	return nil
}

func FindProvider(items []AIProviderManifest, providerID string) (AIProviderManifest, bool) {
	providerID = strings.TrimSpace(providerID)
	for _, item := range items {
		if item.ProviderID == providerID {
			return item, true
		}
	}
	return AIProviderManifest{}, false
}

func SupportsDeploymentMode(manifest AIProviderManifest, mode string) bool {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return true
	}
	for _, supported := range manifest.DeploymentModes {
		if supported == mode {
			return true
		}
	}
	return false
}

func statusRank(status string) int {
	switch status {
	case StatusAvailable:
		return 0
	case StatusConfigurable:
		return 1
	case StatusCustom:
		return 2
	default:
		return 3
	}
}

func builtinManifests() []AIProviderManifest {
	return []AIProviderManifest{
		localPolicyCopilotManifest(),
		azureOpenAIManifest(),
		openAIAPIManifest(),
		ollamaManifest(),
		vLLMManifest(),
		customHTTPManifest(),
	}
}

func localPolicyCopilotManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "everest-local-policy-copilot",
		DisplayName:     "Everest Local Policy Copilot",
		ProviderType:    "local_policy_rules",
		Vendor:          "Data Dynamics",
		Version:         "1.0.0",
		Status:          StatusAvailable,
		Description:     "Evidence-grounded local recommendation engine for governed actions, HITL preparation, and policy explanation without sending customer data to an external model.",
		DeploymentModes: []string{"cloud_connected", "private_network", "air_gapped", "hybrid"},
		NetworkModes:    []string{"local_only", "no_internet_required"},
		Runtime: AIProviderRuntime{
			Kind:      "builtin",
			Protocol:  "evidence_rules_v1",
			Platforms: []string{"windows"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "none", DisplayName: "No credential required", OfflineReady: true},
		},
		ConfigFields: []ConfigField{
			{Name: "model", Label: "Reasoning profile", Type: "select", Required: true, Options: []string{"local-policy-rules-v1"}},
			{Name: "automation_mode", Label: "Automation mode", Type: "select", Required: true, Options: []string{"recommend_only", "submit_hitl_approvals"}},
		},
		ModelOptions: []ModelOption{
			{ModelID: "local-policy-rules-v1", DisplayName: "Local policy rules v1", Capabilities: []string{"recommendations", "action_planning", "policy_explanation"}},
		},
		Capabilities: AIProviderCapability{
			Recommendations:     true,
			PolicyExplanation:   true,
			ActionPlanning:      true,
			PromptGrounding:     true,
			StructuredJSON:      true,
			OfflineInference:    true,
			SupportedTasks:      []string{"azureblob_recommendations", "hitl_request_prefill", "policy_explanation"},
			SupportedModalities: []string{"metadata", "policy", "evidence"},
		},
		Safety: standardSafetyPosture(true),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: false,
			SupportsPrivateEndpoint:       true,
			SupportsAirGap:                true,
			RetentionNotes:                []string{"No external prompt retention; recommendations are derived locally from evidence rows."},
			ResidencyNotes:                []string{"Customer evidence remains inside the local agent/control-plane boundary."},
			DataHandlingNotes:             []string{"No customer content is transmitted to an external AI service."},
		},
		Support: SupportPosture{
			Owner:            "Data Dynamics",
			SupportLevel:     "supported",
			TestRequirements: []string{"local evidence rows", "action plan rows", "Guardian decision output"},
		},
	}
}

func azureOpenAIManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "azure-openai",
		DisplayName:     "Azure OpenAI",
		ProviderType:    "azure_openai",
		Vendor:          "Microsoft Azure",
		Version:         "configurable",
		Status:          StatusConfigurable,
		Description:     "Customer Azure OpenAI deployment using public or private endpoint access, with customer-selected deployment name and enterprise credential policy.",
		DeploymentModes: []string{"cloud_connected", "private_network", "hybrid"},
		NetworkModes:    []string{"public_endpoint", "private_endpoint"},
		Runtime: AIProviderRuntime{
			Kind:      "http",
			Protocol:  "azure_openai_chat_completions",
			Endpoint:  "customer_configured",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "api_key", DisplayName: "API key", SecretFields: []string{"api_key"}, OfflineReady: false},
			{Scheme: "managed_identity", DisplayName: "Managed identity", OfflineReady: false},
		},
		ConfigFields: []ConfigField{
			{Name: "endpoint_url", Label: "Endpoint URL", Type: "url", Required: true},
			{Name: "deployment_name", Label: "Deployment name", Type: "text", Required: true},
			{Name: "api_version", Label: "API version", Type: "text", Required: false},
			{Name: "api_key", Label: "API key", Type: "secret", Required: false, Secret: true},
			{Name: "auth_scheme", Label: "Auth scheme", Type: "select", Required: true, Options: []string{"api_key", "managed_identity"}},
		},
		ModelOptions: []ModelOption{
			{ModelID: "customer_deployment", DisplayName: "Customer configured deployment", Capabilities: []string{"chat", "recommendations", "summarization", "classification", "action_planning"}, DeploymentNameRequired: true},
		},
		Capabilities: cloudLLMCapabilities(false),
		Safety:       standardSafetyPosture(false),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: true,
			SupportsPrivateEndpoint:       true,
			SupportsAirGap:                false,
			RetentionNotes:                []string{"Retention and logging follow the customer's Azure OpenAI tenant configuration."},
			ResidencyNotes:                []string{"Region and private endpoint selection are customer controlled."},
			DataHandlingNotes:             []string{"Use redaction and prompt audit controls for regulated evidence."},
		},
		Support: SupportPosture{
			Owner:            "customer_azure_tenant",
			SupportLevel:     "configuration_supported",
			TestRequirements: []string{"endpoint URL", "deployment name", "API key or managed identity", "network path from Data Agent"},
		},
	}
}

func openAIAPIManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "openai-api",
		DisplayName:     "OpenAI API",
		ProviderType:    "openai_api",
		Vendor:          "OpenAI",
		Version:         "configurable",
		Status:          StatusConfigurable,
		Description:     "OpenAI-compatible cloud API configuration for customers allowed to use internet egress for AI workloads.",
		DeploymentModes: []string{"cloud_connected"},
		NetworkModes:    []string{"public_endpoint"},
		Runtime: AIProviderRuntime{
			Kind:      "http",
			Protocol:  "openai_responses_or_chat_completions",
			Endpoint:  "https://api.openai.com",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "api_key", DisplayName: "API key", SecretFields: []string{"api_key"}, OfflineReady: false},
		},
		ConfigFields: []ConfigField{
			{Name: "endpoint_url", Label: "Endpoint URL", Type: "url", Required: false},
			{Name: "model", Label: "Model", Type: "text", Required: true},
			{Name: "api_key", Label: "API key", Type: "secret", Required: true, Secret: true},
		},
		ModelOptions: []ModelOption{
			{ModelID: "customer_configured_model", DisplayName: "Customer configured model", Capabilities: []string{"chat", "recommendations", "summarization", "classification", "action_planning"}},
		},
		Capabilities: cloudLLMCapabilities(false),
		Safety:       standardSafetyPosture(false),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: true,
			SupportsPrivateEndpoint:       false,
			SupportsAirGap:                false,
			RetentionNotes:                []string{"Retention depends on the configured API account and enterprise agreement."},
			ResidencyNotes:                []string{"Use only where internet egress and data transfer policies allow it."},
			DataHandlingNotes:             []string{"Prompt redaction and evidence minimization should be enabled for sensitive workloads."},
		},
		Support: SupportPosture{
			Owner:            "customer",
			SupportLevel:     "configuration_supported",
			TestRequirements: []string{"internet egress", "model name", "API key"},
		},
	}
}

func ollamaManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "local-ollama",
		DisplayName:     "Local Ollama",
		ProviderType:    "local_ollama",
		Vendor:          "Customer",
		Version:         "configurable",
		Status:          StatusConfigurable,
		Description:     "Local Ollama runtime for private-network and air-gapped deployments where the customer supplies and operates local models.",
		DeploymentModes: []string{"private_network", "air_gapped", "hybrid"},
		NetworkModes:    []string{"localhost", "private_endpoint", "no_internet_required"},
		Runtime: AIProviderRuntime{
			Kind:      "local_http",
			Protocol:  "ollama_chat",
			Endpoint:  "http://localhost:11434",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "none", DisplayName: "No credential", OfflineReady: true},
			{Scheme: "bearer_token", DisplayName: "Bearer token", SecretFields: []string{"bearer_token"}, OfflineReady: true},
		},
		ConfigFields: []ConfigField{
			{Name: "endpoint_url", Label: "Endpoint URL", Type: "url", Required: true},
			{Name: "model", Label: "Local model", Type: "text", Required: true},
			{Name: "bearer_token", Label: "Bearer token", Type: "secret", Required: false, Secret: true},
		},
		ModelOptions: []ModelOption{
			{ModelID: "customer_local_model", DisplayName: "Customer local model", Capabilities: []string{"chat", "recommendations", "summarization", "classification"}},
		},
		Capabilities: cloudLLMCapabilities(true),
		Safety:       standardSafetyPosture(true),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: false,
			SupportsPrivateEndpoint:       true,
			SupportsAirGap:                true,
			RetentionNotes:                []string{"Model and prompt retention are controlled by the customer's local runtime."},
			ResidencyNotes:                []string{"Inference can remain inside the air-gapped environment."},
			DataHandlingNotes:             []string{"Customer is responsible for model provenance, patching, and output quality validation."},
		},
		Support: SupportPosture{
			Owner:             "customer",
			SupportLevel:      "contract_validated",
			TestRequirements:  []string{"local runtime health", "model loaded", "structured JSON response contract"},
			CustomerOwnedCode: true,
		},
	}
}

func vLLMManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "openai-compatible-local",
		DisplayName:     "OpenAI-Compatible Local Endpoint",
		ProviderType:    "openai_compatible_local",
		Vendor:          "Customer",
		Version:         "configurable",
		Status:          StatusConfigurable,
		Description:     "OpenAI-compatible local or private endpoint such as vLLM, LM Studio, or a customer-hosted gateway.",
		DeploymentModes: []string{"private_network", "air_gapped", "hybrid"},
		NetworkModes:    []string{"localhost", "private_endpoint", "no_internet_required"},
		Runtime: AIProviderRuntime{
			Kind:      "local_http",
			Protocol:  "openai_compatible_chat_completions",
			Endpoint:  "customer_configured",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "none", DisplayName: "No credential", OfflineReady: true},
			{Scheme: "api_key", DisplayName: "API key", SecretFields: []string{"api_key"}, OfflineReady: true},
			{Scheme: "bearer_token", DisplayName: "Bearer token", SecretFields: []string{"bearer_token"}, OfflineReady: true},
		},
		ConfigFields: []ConfigField{
			{Name: "endpoint_url", Label: "Endpoint URL", Type: "url", Required: true},
			{Name: "model", Label: "Model", Type: "text", Required: true},
			{Name: "api_key", Label: "API key", Type: "secret", Required: false, Secret: true},
		},
		ModelOptions: []ModelOption{
			{ModelID: "customer_openai_compatible_model", DisplayName: "Customer OpenAI-compatible model", Capabilities: []string{"chat", "recommendations", "summarization", "classification", "action_planning"}},
		},
		Capabilities: cloudLLMCapabilities(true),
		Safety:       standardSafetyPosture(true),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: false,
			SupportsPrivateEndpoint:       true,
			SupportsAirGap:                true,
			RetentionNotes:                []string{"Retention follows the customer's local serving layer."},
			ResidencyNotes:                []string{"Inference can remain inside customer-controlled network boundaries."},
			DataHandlingNotes:             []string{"Provider must return structured JSON for action planning workflows."},
		},
		Support: SupportPosture{
			Owner:             "customer",
			SupportLevel:      "contract_validated",
			TestRequirements:  []string{"endpoint health", "model selection", "structured JSON response contract"},
			CustomerOwnedCode: true,
		},
	}
}

func customHTTPManifest() AIProviderManifest {
	return AIProviderManifest{
		ProviderID:      "custom-http-ai",
		DisplayName:     "Custom HTTP AI Provider",
		ProviderType:    "custom_http",
		Vendor:          "Customer / Partner",
		Version:         "manifest-v1",
		Status:          StatusConfigurable,
		Description:     "Bring-your-own AI gateway using the Everest AI provider manifest and structured recommendation contract.",
		DeploymentModes: []string{"cloud_connected", "private_network", "air_gapped", "hybrid"},
		NetworkModes:    []string{"customer_defined"},
		Runtime: AIProviderRuntime{
			Kind:      "custom_http",
			Protocol:  "everest_ai_provider_v1",
			Endpoint:  "customer_configured",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "api_key", DisplayName: "API key", SecretFields: []string{"api_key"}, OfflineReady: true},
			{Scheme: "bearer_token", DisplayName: "Bearer token", SecretFields: []string{"bearer_token"}, OfflineReady: true},
			{Scheme: "none", DisplayName: "No credential", OfflineReady: true},
		},
		ConfigFields: []ConfigField{
			{Name: "endpoint_url", Label: "Endpoint URL", Type: "url", Required: true},
			{Name: "model", Label: "Model or profile", Type: "text", Required: false},
			{Name: "auth_scheme", Label: "Auth scheme", Type: "select", Required: true, Options: []string{"api_key", "bearer_token", "none"}},
			{Name: "api_key", Label: "API key", Type: "secret", Required: false, Secret: true},
			{Name: "bearer_token", Label: "Bearer token", Type: "secret", Required: false, Secret: true},
		},
		ModelOptions: []ModelOption{
			{ModelID: "customer_profile", DisplayName: "Customer provider profile", Capabilities: []string{"chat", "recommendations", "classification", "action_planning"}},
		},
		Capabilities: cloudLLMCapabilities(true),
		Safety:       standardSafetyPosture(true),
		DataPolicy: AIDataPolicy{
			CustomerDataLeavesEnvironment: true,
			SupportsPrivateEndpoint:       true,
			SupportsAirGap:                true,
			RetentionNotes:                []string{"Retention must be declared by the customer provider manifest."},
			ResidencyNotes:                []string{"Residency must be declared by the customer provider manifest."},
			DataHandlingNotes:             []string{"Provider must support prompt audit, evidence grounding, and blocked autonomous apply controls."},
		},
		Support: SupportPosture{
			Owner:             "customer_or_partner",
			SupportLevel:      "contract_validated",
			TestRequirements:  []string{"manifest validation", "health endpoint", "structured response contract", "redaction policy review"},
			CustomerOwnedCode: true,
		},
	}
}

func cloudLLMCapabilities(offline bool) AIProviderCapability {
	return AIProviderCapability{
		Chat:                true,
		Recommendations:     true,
		Classification:      true,
		Summarization:       true,
		PolicyExplanation:   true,
		ActionPlanning:      true,
		PromptGrounding:     true,
		StructuredJSON:      true,
		ToolCalling:         true,
		Streaming:           true,
		OfflineInference:    offline,
		SupportedTasks:      []string{"chat", "classification", "summarization", "policy_explanation", "action_recommendations", "hitl_request_prefill"},
		SupportedModalities: []string{"text", "metadata", "policy", "evidence"},
	}
}

func standardSafetyPosture(offlineReady bool) AISafetyPosture {
	return AISafetyPosture{
		AdvisoryOnlyDefault:       true,
		HITLSubmissionAllowed:     true,
		AutonomousApplyAllowed:    false,
		PromptAuditRequired:       true,
		EvidenceGroundingRequired: true,
		RedactionSupported:        true,
		AllowedAutomationModes:    []string{"recommend_only", "submit_hitl_approvals"},
		BlockedActions:            []string{"autonomous_apply", "silent_delete", "ungrounded_writeback"},
	}
}
