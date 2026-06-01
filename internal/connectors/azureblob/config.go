package azureblob

import (
	"fmt"
	"strings"
)

type Config struct {
	ConnectionString string
	ContainerName    string
	Prefix           string
	DataStoreID      string
	BatchID          string
	ActionMode       string
	WritebackEnabled string
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
