package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StatusAvailable = "available"
	StatusPlanned   = "planned"
	StatusCustom    = "custom_manifest"
)

type ConnectorManifest struct {
	ConnectorID     string               `json:"connector_id"`
	DisplayName     string               `json:"display_name"`
	ConnectorType   string               `json:"connector_type"`
	Vendor          string               `json:"vendor"`
	Version         string               `json:"version"`
	Status          string               `json:"status"`
	Description     string               `json:"description"`
	DeploymentModes []string             `json:"deployment_modes"`
	NetworkModes    []string             `json:"network_modes"`
	Runtime         ConnectorRuntime     `json:"runtime"`
	AuthSchemes     []AuthScheme         `json:"auth_schemes"`
	ConfigFields    []ConfigField        `json:"config_fields"`
	Capabilities    ConnectorCapability  `json:"capabilities"`
	Actions         []ConnectorAction    `json:"actions"`
	ObjectModel     ConnectorObjectModel `json:"object_model"`
	AI              AICapability         `json:"ai"`
	Security        SecurityPosture      `json:"security"`
	Support         SupportPosture       `json:"support"`
	LoadedFrom      string               `json:"loaded_from,omitempty"`
}

type ConnectorRuntime struct {
	Kind        string   `json:"kind"`
	Entrypoint  string   `json:"entrypoint,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	HealthCheck string   `json:"health_check,omitempty"`
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

type ConnectorCapability struct {
	DiscoverSources      bool     `json:"discover_sources"`
	ScanMetadata         bool     `json:"scan_metadata"`
	ContentSampling      bool     `json:"content_sampling"`
	PermissionsRead      bool     `json:"permissions_read"`
	ApplyMetadata        bool     `json:"apply_metadata"`
	ApplyNativeTags      bool     `json:"apply_native_tags"`
	Archive              bool     `json:"archive"`
	Retrieve             bool     `json:"retrieve"`
	Quarantine           bool     `json:"quarantine"`
	CopyOrMigrate        bool     `json:"copy_or_migrate"`
	Delete               bool     `json:"delete"`
	CustomerExtensible   bool     `json:"customer_extensible"`
	SupportedObjectTypes []string `json:"supported_object_types"`
}

type ConnectorAction struct {
	ActionID         string   `json:"action_id"`
	DisplayName      string   `json:"display_name"`
	Mode             string   `json:"mode"`
	RequiresGuardian bool     `json:"requires_guardian"`
	RequiresHITL     bool     `json:"requires_hitl"`
	Confirmation     string   `json:"confirmation,omitempty"`
	SupportedModes   []string `json:"supported_modes"`
}

type ConnectorObjectModel struct {
	FileIDFormat     string   `json:"file_id_format"`
	RequiredFields   []string `json:"required_fields"`
	OptionalFields   []string `json:"optional_fields"`
	NativeAttributes []string `json:"native_attributes"`
}

type AICapability struct {
	RecommendationSupported bool     `json:"recommendation_supported"`
	AutoSubmitHITL          bool     `json:"auto_submit_hitl"`
	AutonomousApply         bool     `json:"autonomous_apply"`
	AllowedAIActions        []string `json:"allowed_ai_actions"`
}

type SecurityPosture struct {
	SecretsInCredentialManager bool     `json:"secrets_in_credential_manager"`
	NoInboundRequired          bool     `json:"no_inbound_required"`
	AirGapNotes                []string `json:"air_gap_notes"`
	DataResidencyNotes         []string `json:"data_residency_notes"`
}

type SupportPosture struct {
	Owner             string   `json:"owner"`
	SupportLevel      string   `json:"support_level"`
	TestRequirements  []string `json:"test_requirements"`
	CustomerOwnedCode bool     `json:"customer_owned_code"`
}

func LoadCatalog(manifestDirs ...string) []ConnectorManifest {
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

func LoadManifestsFromDir(dir string) ([]ConnectorManifest, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	items := []ConnectorManifest{}
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

func LoadManifest(path string) (ConnectorManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ConnectorManifest{}, err
	}

	var manifest ConnectorManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ConnectorManifest{}, err
	}

	if err := ValidateManifest(manifest); err != nil {
		return ConnectorManifest{}, err
	}

	return manifest, nil
}

func ValidateManifest(manifest ConnectorManifest) error {
	if strings.TrimSpace(manifest.ConnectorID) == "" {
		return errors.New("connector_id is required")
	}
	if strings.TrimSpace(manifest.DisplayName) == "" {
		return errors.New("display_name is required")
	}
	if strings.TrimSpace(manifest.ConnectorType) == "" {
		return errors.New("connector_type is required")
	}
	if strings.TrimSpace(manifest.Runtime.Kind) == "" {
		return errors.New("runtime.kind is required")
	}
	if len(manifest.DeploymentModes) == 0 {
		return errors.New("deployment_modes is required")
	}
	if len(manifest.Actions) == 0 {
		return errors.New("actions is required")
	}
	return nil
}

func statusRank(status string) int {
	switch status {
	case StatusAvailable:
		return 0
	case StatusCustom:
		return 1
	default:
		return 2
	}
}

func builtinManifests() []ConnectorManifest {
	return []ConnectorManifest{
		azureBlobManifest(),
		plannedConnector("aws-s3", "Amazon S3", "s3", []string{"cloud_connected", "private_network", "air_gapped"}, []string{"bucket", "prefix", "region"}),
		plannedConnector("smb", "SMB / Windows File Share", "smb", []string{"private_network", "air_gapped"}, []string{"share", "path", "domain"}),
		plannedConnector("nfs", "NFS", "nfs", []string{"private_network", "air_gapped"}, []string{"export", "mount_path"}),
		plannedConnector("sharepoint", "SharePoint Online", "sharepoint", []string{"cloud_connected", "hybrid"}, []string{"site", "drive", "tenant"}),
		plannedConnector("onedrive", "OneDrive", "onedrive", []string{"cloud_connected", "hybrid"}, []string{"tenant", "user_scope"}),
		plannedConnector("gcs", "Google Cloud Storage", "gcs", []string{"cloud_connected", "private_network"}, []string{"bucket", "prefix", "project"}),
	}
}

func azureBlobManifest() ConnectorManifest {
	return ConnectorManifest{
		ConnectorID:     "azureblob",
		DisplayName:     "Azure Blob Storage",
		ConnectorType:   "azure_blob",
		Vendor:          "Data Dynamics",
		Version:         "1.0.0",
		Status:          StatusAvailable,
		Description:     "Enterprise Azure Blob connector with metadata scan, evidence, AI recommendations, HITL approvals, and governed action execution.",
		DeploymentModes: []string{"cloud_connected", "private_network", "air_gapped"},
		NetworkModes:    []string{"public_endpoint", "private_endpoint", "offline_policy_bundle"},
		Runtime: ConnectorRuntime{
			Kind:        "builtin",
			Protocol:    "go_sdk",
			HealthCheck: "/v1/connectors/azureblob/status",
			Platforms:   []string{"windows"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "connection_string", DisplayName: "Azure Storage connection string", SecretFields: []string{"connection_string"}, OfflineReady: true},
			{Scheme: "sas", DisplayName: "SAS token", SecretFields: []string{"sas_token"}, OfflineReady: true},
			{Scheme: "managed_identity", DisplayName: "Managed identity", OfflineReady: false},
		},
		ConfigFields: []ConfigField{
			{Name: "storage_account", Label: "Storage account", Type: "text", Required: true},
			{Name: "container", Label: "Container", Type: "multi_select_or_text", Required: true},
			{Name: "prefix", Label: "Prefix", Type: "text", Required: false},
			{Name: "connection_string", Label: "Connection string", Type: "secret", Required: false, Secret: true},
			{Name: "action_mode", Label: "Action mode", Type: "select", Required: true, Options: []string{"dry_run", "apply"}},
		},
		Capabilities: ConnectorCapability{
			DiscoverSources:      true,
			ScanMetadata:         true,
			ContentSampling:      false,
			PermissionsRead:      true,
			ApplyMetadata:        true,
			ApplyNativeTags:      true,
			Archive:              true,
			Retrieve:             true,
			Quarantine:           true,
			CopyOrMigrate:        true,
			Delete:               true,
			CustomerExtensible:   false,
			SupportedObjectTypes: []string{"blob", "container", "prefix"},
		},
		Actions: standardObjectActions(),
		ObjectModel: ConnectorObjectModel{
			FileIDFormat:     "azureblob:{container}:{blob_path}",
			RequiredFields:   []string{"file_id", "data_store_id", "path", "name", "size_bytes", "source_system", "permission_risk_score"},
			OptionalFields:   []string{"content_type", "extension", "last_modified", "last_accessed", "native_metadata", "index_tags"},
			NativeAttributes: []string{"container", "etag", "access_tier", "blob_type", "metadata", "tags"},
		},
		AI: AICapability{
			RecommendationSupported: true,
			AutoSubmitHITL:          true,
			AutonomousApply:         false,
			AllowedAIActions:        []string{"recommend", "prefill_action", "submit_hitl_approval"},
		},
		Security: SecurityPosture{
			SecretsInCredentialManager: true,
			NoInboundRequired:          true,
			AirGapNotes: []string{
				"Policy bundles, workflow bundles, and connector manifest can be imported locally.",
				"Evidence export can be moved out-of-band when no internet is available.",
			},
			DataResidencyNotes: []string{"No customer content leaves the environment in local execution mode."},
		},
		Support: SupportPosture{
			Owner:            "Data Dynamics",
			SupportLevel:     "supported",
			TestRequirements: []string{"Azure Storage account", "container read permission", "optional metadata/tag/write permissions"},
		},
	}
}

func plannedConnector(id string, displayName string, connectorType string, modes []string, configFields []string) ConnectorManifest {
	fields := make([]ConfigField, 0, len(configFields))
	for _, field := range configFields {
		fields = append(fields, ConfigField{Name: field, Label: strings.ReplaceAll(field, "_", " "), Type: "text", Required: true})
	}

	return ConnectorManifest{
		ConnectorID:     id,
		DisplayName:     displayName,
		ConnectorType:   connectorType,
		Vendor:          "Data Dynamics / Customer Plugin",
		Version:         "roadmap-" + time.Now().UTC().Format("2006"),
		Status:          StatusPlanned,
		Description:     "Connector contract is supported by the Everest connector framework; runtime implementation can be supplied by product or customer plugin.",
		DeploymentModes: modes,
		NetworkModes:    []string{"customer_defined"},
		Runtime: ConnectorRuntime{
			Kind:      "external_process_or_http",
			Protocol:  "connector_manifest_v1",
			Platforms: []string{"windows", "linux"},
		},
		AuthSchemes: []AuthScheme{
			{Scheme: "customer_defined_secret", DisplayName: "Customer-defined secret", SecretFields: []string{"credential"}, OfflineReady: true},
		},
		ConfigFields: fields,
		Capabilities: ConnectorCapability{
			DiscoverSources:      true,
			ScanMetadata:         true,
			PermissionsRead:      true,
			CustomerExtensible:   true,
			SupportedObjectTypes: []string{"object", "folder", "container"},
		},
		Actions: standardObjectActions(),
		ObjectModel: ConnectorObjectModel{
			FileIDFormat:     connectorType + ":{scope}:{native_path}",
			RequiredFields:   []string{"file_id", "data_store_id", "path", "name", "size_bytes", "source_system", "permission_risk_score"},
			OptionalFields:   []string{"content_type", "extension", "last_modified", "last_accessed", "native_metadata"},
			NativeAttributes: []string{"customer_defined"},
		},
		AI: AICapability{
			RecommendationSupported: true,
			AutoSubmitHITL:          true,
			AutonomousApply:         false,
			AllowedAIActions:        []string{"recommend", "prefill_action", "submit_hitl_approval"},
		},
		Security: SecurityPosture{
			SecretsInCredentialManager: true,
			NoInboundRequired:          true,
			AirGapNotes:                []string{"Customer plugin must run inside the customer-controlled network boundary."},
			DataResidencyNotes:         []string{"Connector output must normalize metadata locally before upload/export."},
		},
		Support: SupportPosture{
			Owner:             "customer_or_partner",
			SupportLevel:      "contract_validated",
			TestRequirements:  []string{"manifest validation", "health check", "normalized batch contract", "action dry-run contract"},
			CustomerOwnedCode: true,
		},
	}
}

func standardObjectActions() []ConnectorAction {
	return []ConnectorAction{
		{ActionID: "metadata_scan", DisplayName: "Metadata scan", Mode: "read_only", SupportedModes: []string{"dry_run", "apply"}},
		{ActionID: "apply_metadata", DisplayName: "Apply metadata", Mode: "non_destructive_writeback", RequiresGuardian: true, RequiresHITL: true, SupportedModes: []string{"apply"}},
		{ActionID: "apply_native_tags", DisplayName: "Apply native tags", Mode: "non_destructive_writeback", RequiresGuardian: true, RequiresHITL: true, SupportedModes: []string{"apply"}},
		{ActionID: "archive", DisplayName: "Archive object", Mode: "restricted_writeback", RequiresGuardian: true, RequiresHITL: true, Confirmation: "APPLY", SupportedModes: []string{"dry_run", "apply"}},
		{ActionID: "retrieve", DisplayName: "Retrieve object", Mode: "restricted_writeback", RequiresGuardian: true, RequiresHITL: true, Confirmation: "APPLY", SupportedModes: []string{"dry_run", "apply"}},
		{ActionID: "quarantine", DisplayName: "Quarantine object", Mode: "restricted_writeback", RequiresGuardian: true, RequiresHITL: true, Confirmation: "APPLY", SupportedModes: []string{"dry_run", "apply"}},
		{ActionID: "migration", DisplayName: "Copy or migrate object", Mode: "restricted_writeback", RequiresGuardian: true, RequiresHITL: true, Confirmation: "APPLY", SupportedModes: []string{"dry_run", "apply"}},
		{ActionID: "delete", DisplayName: "Delete object", Mode: "destructive_writeback", RequiresGuardian: true, RequiresHITL: true, Confirmation: "DELETE", SupportedModes: []string{"dry_run", "apply"}},
	}
}
