package azureblob

import (
	"fmt"
	"strings"
)

const ConnectionStringSecretName = "connectors/azureblob/connection_string"

type Config struct {
	ConnectionString string
	ContainerName    string
	Prefix           string
	DataStoreID      string
	BatchID          string
	ActionMode       string
	WritebackEnabled string
}

type ConfigureRequest struct {
	ConnectionString string `json:"connection_string"`
	ContainerName    string `json:"container"`
	Prefix           string `json:"prefix"`
	DataStoreID      string `json:"data_store_id"`
	BatchID          string `json:"batch_id"`
	ActionMode       string `json:"action_mode"`
	WritebackEnabled string `json:"writeback_enabled"`
}

func (c Config) ValidateForScan() error {
	if strings.TrimSpace(c.ConnectionString) == "" {
		return fmt.Errorf("AZURE_STORAGE_CONNECTION_STRING is required")
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
