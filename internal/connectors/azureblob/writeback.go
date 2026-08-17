package azureblob

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type WritebackRequest struct {
	FileID    string            `json:"file_id"`
	BlobPath  string            `json:"blob_path,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	IndexTags map[string]string `json:"index_tags,omitempty"`
}

type WritebackResult struct {
	Status    string            `json:"status"`
	Action    string            `json:"action"`
	FileID    string            `json:"file_id"`
	Container string            `json:"container"`
	BlobPath  string            `json:"blob_path"`
	Applied   map[string]string `json:"applied,omitempty"`
	Message   string            `json:"message,omitempty"`
}

func (s *Scanner) ApplyMetadata(ctx context.Context, req WritebackRequest) (*WritebackResult, error) {
	if err := s.cfg.ValidateForScan(); err != nil {
		return nil, err
	}

	blobPath, err := s.resolveBlobPath(req)
	if err != nil {
		return nil, err
	}

	metadataForAzure, metadataApplied := sanitizeAzureMetadata(req.Metadata)
	if len(metadataForAzure) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}

	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	blobClient := client.ServiceClient().
		NewContainerClient(s.cfg.ContainerName).
		NewBlobClient(blobPath)

	if _, err := blobClient.SetMetadata(ctx, metadataForAzure, nil); err != nil {
		return nil, fmt.Errorf("set azure blob metadata: %w", err)
	}

	return &WritebackResult{
		Status:    "completed",
		Action:    "apply_blob_metadata",
		FileID:    req.FileID,
		Container: s.cfg.ContainerName,
		BlobPath:  blobPath,
		Applied:   metadataApplied,
		Message:   "Azure Blob metadata updated successfully.",
	}, nil
}

func (s *Scanner) ApplyIndexTags(ctx context.Context, req WritebackRequest) (*WritebackResult, error) {
	if err := s.cfg.ValidateForScan(); err != nil {
		return nil, err
	}

	blobPath, err := s.resolveBlobPath(req)
	if err != nil {
		return nil, err
	}

	tags := sanitizeAzureIndexTags(req.IndexTags)
	if len(tags) == 0 {
		return nil, fmt.Errorf("index_tags are required")
	}

	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	blobClient := client.ServiceClient().
		NewContainerClient(s.cfg.ContainerName).
		NewBlobClient(blobPath)

	if _, err := blobClient.SetTags(ctx, tags, nil); err != nil {
		return nil, fmt.Errorf("set azure blob index tags: %w", err)
	}

	return &WritebackResult{
		Status:    "completed",
		Action:    "apply_blob_index_tags",
		FileID:    req.FileID,
		Container: s.cfg.ContainerName,
		BlobPath:  blobPath,
		Applied:   tags,
		Message:   "Azure Blob index tags updated successfully.",
	}, nil
}

func (s *Scanner) resolveBlobPath(req WritebackRequest) (string, error) {
	if strings.TrimSpace(req.BlobPath) != "" {
		return strings.TrimSpace(req.BlobPath), nil
	}

	blobPath, err := BlobPathFromFileID(req.FileID, s.cfg.ContainerName)
	if err != nil {
		return "", err
	}

	return blobPath, nil
}

func BlobPathFromFileID(fileID string, expectedContainer string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	expectedContainer = strings.TrimSpace(expectedContainer)

	if fileID == "" {
		return "", fmt.Errorf("file_id is required")
	}

	const prefix = "azureblob:"
	if !strings.HasPrefix(fileID, prefix) {
		return "", fmt.Errorf("unsupported azure blob file_id: %s", fileID)
	}

	rest := strings.TrimPrefix(fileID, prefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid azure blob file_id format: %s", fileID)
	}

	container := strings.TrimSpace(parts[0])
	blobPath := strings.TrimSpace(parts[1])

	if container == "" || blobPath == "" {
		return "", fmt.Errorf("invalid azure blob file_id format: %s", fileID)
	}

	if expectedContainer != "" && container != expectedContainer {
		return "", fmt.Errorf("file_id container %q does not match configured container %q", container, expectedContainer)
	}

	decoded, err := url.PathUnescape(blobPath)
	if err == nil && decoded != "" {
		blobPath = decoded
	}

	return blobPath, nil
}

func ContainerFromFileID(fileID string) (string, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return "", fmt.Errorf("file_id is required")
	}

	const prefix = "azureblob:"
	if !strings.HasPrefix(fileID, prefix) {
		return "", fmt.Errorf("unsupported azure blob file_id: %s", fileID)
	}

	rest := strings.TrimPrefix(fileID, prefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid azure blob file_id format: %s", fileID)
	}

	container := strings.TrimSpace(parts[0])
	if container == "" {
		return "", fmt.Errorf("invalid azure blob file_id format: %s", fileID)
	}

	return container, nil
}

func sanitizeAzureMetadata(input map[string]string) (map[string]*string, map[string]string) {
	forAzure := map[string]*string{}
	applied := map[string]string{}

	for key, value := range input {
		cleanKey := sanitizeAzureMetadataKey(key)
		if cleanKey == "" {
			continue
		}

		cleanValue := strings.TrimSpace(value)
		valueCopy := cleanValue

		forAzure[cleanKey] = &valueCopy
		applied[cleanKey] = cleanValue
	}

	return forAzure, applied
}

func sanitizeAzureIndexTags(input map[string]string) map[string]string {
	out := map[string]string{}

	for key, value := range input {
		cleanKey := sanitizeAzureTagKey(key)
		if cleanKey == "" {
			continue
		}

		out[cleanKey] = strings.TrimSpace(value)
	}

	return out
}

func sanitizeAzureMetadataKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, " ", "_")

	var b strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '_' {
			b.WriteRune(r)
		}
	}

	out := b.String()
	if out == "" {
		return ""
	}

	if out[0] >= '0' && out[0] <= '9' {
		out = "everest_" + out
	}

	return out
}

func sanitizeAzureTagKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, " ", "_")

	var b strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' ||
			r == '-' {
			b.WriteRune(r)
		}
	}

	return b.String()
}
