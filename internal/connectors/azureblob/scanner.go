package azureblob

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type Scanner struct {
	cfg Config
}

type ScanResult struct {
	Status         string `json:"status"`
	BatchID        string `json:"batch_id"`
	DataStoreID    string `json:"data_store_id"`
	ContainerName  string `json:"container_name"`
	Prefix         string `json:"prefix"`
	RecordsScanned int    `json:"records_scanned"`
	ActionMode     string `json:"action_mode"`
	Writeback      string `json:"writeback_enabled"`
}

func NewScanner(cfg Config) (*Scanner, error) {
	if err := cfg.ValidateForScan(); err != nil {
		return nil, err
	}
	return &Scanner{cfg: cfg}, nil
}

func (s *Scanner) TestConnection(ctx context.Context) error {
	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	pager := client.NewListBlobsFlatPager(s.cfg.ContainerName, &azblob.ListBlobsFlatOptions{
		Prefix:     to.Ptr(s.cfg.Prefix),
		MaxResults: to.Ptr[int32](1),
	})

	if pager.More() {
		if _, err := pager.NextPage(ctx); err != nil {
			return fmt.Errorf("test list azure blobs: %w", err)
		}
	}

	return nil
}

func (s *Scanner) Scan(ctx context.Context) (*contracts.NormalizedBatch, *ScanResult, error) {
	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	batch := &contracts.NormalizedBatch{
		BatchID:     s.cfg.BatchID,
		DataStoreID: s.cfg.DataStoreID,
		Records:     make([]contracts.NormalizedFileRecord, 0),
	}

	pager := client.NewListBlobsFlatPager(s.cfg.ContainerName, &azblob.ListBlobsFlatOptions{
		Prefix: to.Ptr(s.cfg.Prefix),
	})

	now := time.Now().UTC()

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("list azure blobs: %w", err)
		}

		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}

			blobName := *item.Name
			props := item.Properties

			size := int64(0)
			if props.ContentLength != nil {
				size = *props.ContentLength
			}

			lastModified := ""
			if props.LastModified != nil {
				lastModified = props.LastModified.UTC().Format(time.RFC3339)
			}

			contentType := ""
			if props.ContentType != nil {
				contentType = *props.ContentType
			}

			record := contracts.NormalizedFileRecord{
				FileID:              buildFileID(s.cfg.ContainerName, blobName),
				DataStoreID:         s.cfg.DataStoreID,
				Path:                blobName,
				Name:                filepath.Base(blobName),
				SizeBytes:           size,
				LastModified:        lastModified,
				LastAccessed:        "",
				ContentType:         contentType,
				Extension:           strings.TrimPrefix(strings.ToLower(filepath.Ext(blobName)), "."),
				SourceSystem:        "azure_blob",
				PermissionClass:     "container_scope",
				PermissionRiskScore: estimateRiskScore(blobName, contentType, size),
				DDProtectionFlags:   0,
				ScanTimestamp:       now,
			}

			batch.Records = append(batch.Records, record)
		}
	}

	result := &ScanResult{
		Status:         "completed",
		BatchID:        batch.BatchID,
		DataStoreID:    batch.DataStoreID,
		ContainerName:  s.cfg.ContainerName,
		Prefix:         s.cfg.Prefix,
		RecordsScanned: len(batch.Records),
		ActionMode:     s.cfg.ActionMode,
		Writeback:      s.cfg.WritebackEnabled,
	}

	return batch, result, nil
}

func buildFileID(containerName string, blobName string) string {
	return "azureblob:" + containerName + ":" + blobName
}

func estimateRiskScore(path string, contentType string, size int64) int {
	lowerPath := strings.ToLower(path)

	score := 30

	if strings.Contains(lowerPath, "payroll") ||
		strings.Contains(lowerPath, "salary") ||
		strings.Contains(lowerPath, "confidential") ||
		strings.Contains(lowerPath, "contract") ||
		strings.Contains(lowerPath, "acquisition") ||
		strings.Contains(lowerPath, "finance") {
		score += 40
	}

	switch strings.ToLower(contentType) {
	case "application/pdf",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-excel":
		score += 15
	}

	if size > 5*1024*1024 {
		score += 10
	}

	if score > 100 {
		return 100
	}

	return score
}

func (s *Scanner) ListContainers(ctx context.Context) ([]string, error) {
	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	pager := client.NewListContainersPager(nil)

	containers := make([]string, 0)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list azure containers: %w", err)
		}

		for _, item := range page.ContainerItems {
			if item.Name == nil {
				continue
			}
			containers = append(containers, *item.Name)
		}
	}

	return containers, nil
}
