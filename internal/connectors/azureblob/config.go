package azureblob

import (
	"fmt"
	"strings"
)

const ConnectionStringSecretName = "connectors/azureblob/connection_string"

func SourceCredentialSecretName(sourceID string) string {
	sourceID = sanitizeSecretSegment(sourceID)
	if sourceID == "" {
		return ConnectionStringSecretName
	}
	return "connectors/azureblob/sources/" + sourceID + "/connection_string"
}

type Config struct {
	ConnectionString string
	StorageAccount   string
	ContainerName    string
	Prefix           string
	DataStoreID      string
	BatchID          string
	ActionMode       string
	WritebackEnabled string
}

type ConfigureRequest struct {
	SourceID         string `json:"source_id,omitempty"`
	ConnectionString string `json:"connection_string"`
	StorageAccount   string `json:"storage_account"`
	AccountKey       string `json:"account_key,omitempty"`
	EndpointSuffix   string `json:"endpoint_suffix,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	ContainerName    string `json:"container"`
	Prefix           string `json:"prefix"`
	DataStoreID      string `json:"data_store_id"`
	BatchID          string `json:"batch_id"`
	ActionMode       string `json:"action_mode"`
	WritebackEnabled string `json:"writeback_enabled"`
}

func ConnectionStringFromRequest(req ConfigureRequest) string {
	if strings.TrimSpace(req.ConnectionString) != "" {
		return strings.TrimSpace(req.ConnectionString)
	}

	account := strings.TrimSpace(req.StorageAccount)
	key := strings.TrimSpace(req.AccountKey)
	if account == "" || key == "" {
		return ""
	}

	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = "https"
	}
	suffix := strings.TrimSpace(req.EndpointSuffix)
	if suffix == "" {
		suffix = "core.windows.net"
	}

	return "DefaultEndpointsProtocol=" + protocol +
		";AccountName=" + account +
		";AccountKey=" + key +
		";EndpointSuffix=" + suffix
}

func sanitizeSecretSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	lastSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastSeparator = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			lastSeparator = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSeparator = false
		case r == '-' || r == '_':
			b.WriteRune(r)
			lastSeparator = false
		default:
			if !lastSeparator {
				b.WriteByte('_')
				lastSeparator = true
			}
		}
	}

	return strings.Trim(b.String(), "_")
}

func (c Config) ValidateForScan() error {
	if strings.TrimSpace(c.ConnectionString) == "" {
		return fmt.Errorf("Azure Blob account access is not configured. Add account name and access key in Source Inventory")
	}
	if strings.TrimSpace(c.ContainerName) == "" {
		return fmt.Errorf("AZURE_BLOB_CONTAINER is required")
	}
	if strings.TrimSpace(c.DataStoreID) == "" {
		return fmt.Errorf("azure blob data_store_id is required")
	}
	if strings.TrimSpace(c.BatchID) == "" {
		return fmt.Errorf("azure blob batch_id is required")
	}
	return nil
}

func ConfigFromRequest(req ConfigureRequest, existing Config) Config {
	next := existing

	if strings.TrimSpace(req.ConnectionString) != "" {
		next.ConnectionString = strings.TrimSpace(req.ConnectionString)
	}

	if strings.TrimSpace(req.StorageAccount) != "" {
		next.StorageAccount = strings.TrimSpace(req.StorageAccount)
	}

	if strings.TrimSpace(req.ContainerName) != "" {
		next.ContainerName = strings.TrimSpace(req.ContainerName)
	}

	next.Prefix = strings.TrimSpace(req.Prefix)

	if strings.TrimSpace(req.DataStoreID) != "" {
		next.DataStoreID = strings.TrimSpace(req.DataStoreID)
	}

	if strings.TrimSpace(req.BatchID) != "" {
		next.BatchID = strings.TrimSpace(req.BatchID)
	}

	if strings.TrimSpace(req.ActionMode) != "" {
		next.ActionMode = strings.TrimSpace(req.ActionMode)
	}

	if strings.TrimSpace(req.WritebackEnabled) != "" {
		next.WritebackEnabled = strings.TrimSpace(req.WritebackEnabled)
	}

	return next
}
