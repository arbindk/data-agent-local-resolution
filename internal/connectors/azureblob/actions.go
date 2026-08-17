package azureblob

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

const (
	ActionArchive    = "archive"
	ActionRetrieve   = "retrieve"
	ActionRehydrate  = "rehydrate"
	ActionQuarantine = "quarantine"
	ActionMigration  = "migration"
	ActionDelete     = "delete"
)

type ActionRequest struct {
	ExecutionID       string            `json:"execution_id,omitempty"`
	FileID            string            `json:"file_id"`
	BlobPath          string            `json:"blob_path,omitempty"`
	Operation         string            `json:"operation"`
	TargetContainer   string            `json:"target_container,omitempty"`
	TargetPrefix      string            `json:"target_prefix,omitempty"`
	TargetBlobPath    string            `json:"target_blob_path,omitempty"`
	TargetTier        string            `json:"target_tier,omitempty"`
	RehydratePriority string            `json:"rehydrate_priority,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	IndexTags         map[string]string `json:"index_tags,omitempty"`
}

type ActionResult struct {
	Status            string            `json:"status"`
	Operation         string            `json:"operation"`
	FileID            string            `json:"file_id,omitempty"`
	Container         string            `json:"container"`
	BlobPath          string            `json:"blob_path"`
	TargetContainer   string            `json:"target_container,omitempty"`
	TargetBlobPath    string            `json:"target_blob_path,omitempty"`
	AccessTier        string            `json:"access_tier,omitempty"`
	RehydratePriority string            `json:"rehydrate_priority,omitempty"`
	CopyID            string            `json:"copy_id,omitempty"`
	CopyStatus        string            `json:"copy_status,omitempty"`
	Applied           map[string]string `json:"applied,omitempty"`
	Message           string            `json:"message,omitempty"`
	CompletedAt       string            `json:"completed_at"`
}

func NormalizeAction(operation string) string {
	return strings.ToLower(strings.TrimSpace(operation))
}

func (s *Scanner) ExecuteAction(ctx context.Context, req ActionRequest) (*ActionResult, error) {
	if err := s.cfg.ValidateForScan(); err != nil {
		return nil, err
	}

	req.Operation = NormalizeAction(req.Operation)
	if req.Operation == "" {
		return nil, fmt.Errorf("operation is required")
	}

	blobPath, err := s.resolveBlobPath(WritebackRequest{
		FileID:   req.FileID,
		BlobPath: req.BlobPath,
	})
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClientFromConnectionString(s.cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("create azure blob client from connection string: %w", err)
	}

	sourceBlob := client.ServiceClient().
		NewContainerClient(s.cfg.ContainerName).
		NewBlobClient(blobPath)

	result := &ActionResult{
		Status:      "completed",
		Operation:   req.Operation,
		FileID:      req.FileID,
		Container:   s.cfg.ContainerName,
		BlobPath:    blobPath,
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
	}

	switch req.Operation {
	case "apply_blob_metadata":
		writeback, err := s.ApplyMetadata(ctx, WritebackRequest{
			FileID:   req.FileID,
			BlobPath: req.BlobPath,
			Metadata: req.Metadata,
		})
		if err != nil {
			return nil, err
		}
		result.Applied = writeback.Applied
		result.Message = writeback.Message

	case "apply_blob_index_tags":
		writeback, err := s.ApplyIndexTags(ctx, WritebackRequest{
			FileID:    req.FileID,
			BlobPath:  req.BlobPath,
			IndexTags: req.IndexTags,
		})
		if err != nil {
			return nil, err
		}
		result.Applied = writeback.Applied
		result.Message = writeback.Message

	case ActionArchive:
		if _, err := sourceBlob.SetTier(ctx, blob.AccessTierArchive, nil); err != nil {
			return nil, fmt.Errorf("set azure blob archive tier: %w", err)
		}
		result.AccessTier = string(blob.AccessTierArchive)
		result.Message = "Azure Blob access tier set to Archive."

	case ActionRetrieve, ActionRehydrate:
		tier := accessTierFromRequest(req.TargetTier, blob.AccessTierHot)
		priority := rehydratePriorityFromRequest(req.RehydratePriority)
		if _, err := sourceBlob.SetTier(ctx, tier, &blob.SetTierOptions{RehydratePriority: to.Ptr(priority)}); err != nil {
			return nil, fmt.Errorf("rehydrate azure blob to %s tier: %w", tier, err)
		}
		result.AccessTier = string(tier)
		result.RehydratePriority = string(priority)
		result.Message = "Azure Blob rehydrate/tier change requested."

	case ActionQuarantine:
		applied, err := s.applyQuarantine(ctx, client, sourceBlob, blobPath, req, result)
		if err != nil {
			return nil, err
		}
		result.Applied = applied
		result.Message = "Azure Blob quarantine action completed."

	case ActionMigration:
		if err := copyBlob(ctx, client, sourceBlob, blobPath, req, result); err != nil {
			return nil, err
		}
		result.Message = "Azure Blob migration copy started."

	case ActionDelete:
		if _, err := sourceBlob.Delete(ctx, &blob.DeleteOptions{
			DeleteSnapshots: to.Ptr(blob.DeleteSnapshotsOptionTypeInclude),
		}); err != nil {
			return nil, fmt.Errorf("delete azure blob: %w", err)
		}
		result.Message = "Azure Blob deleted with snapshots included."

	default:
		return nil, fmt.Errorf("unsupported azure blob action: %s", req.Operation)
	}

	return result, nil
}

func (s *Scanner) applyQuarantine(
	ctx context.Context,
	client *azblob.Client,
	sourceBlob *blob.Client,
	blobPath string,
	req ActionRequest,
	result *ActionResult,
) (map[string]string, error) {
	if strings.TrimSpace(req.TargetContainer) != "" {
		if err := copyBlob(ctx, client, sourceBlob, blobPath, req, result); err != nil {
			return nil, err
		}
	}

	tags := map[string]string{
		"everest_quarantine":   "true",
		"everest_action":       ActionQuarantine,
		"everest_execution_id": strings.TrimSpace(req.ExecutionID),
	}
	for k, v := range req.IndexTags {
		tags[k] = v
	}
	tags = sanitizeAzureIndexTags(tags)

	if _, err := sourceBlob.SetTags(ctx, tags, nil); err != nil {
		return nil, fmt.Errorf("set azure blob quarantine tags: %w", err)
	}

	return tags, nil
}

func copyBlob(
	ctx context.Context,
	client *azblob.Client,
	sourceBlob *blob.Client,
	sourceBlobPath string,
	req ActionRequest,
	result *ActionResult,
) error {
	targetContainer := strings.TrimSpace(req.TargetContainer)
	if targetContainer == "" {
		return fmt.Errorf("target_container is required for %s", req.Operation)
	}

	targetBlobPath := targetBlobPathFromRequest(req.TargetBlobPath, req.TargetPrefix, sourceBlobPath)
	if targetBlobPath == "" {
		return fmt.Errorf("target blob path could not be resolved")
	}

	if targetContainer == result.Container && targetBlobPath == sourceBlobPath {
		return fmt.Errorf("target must differ from source for %s", req.Operation)
	}

	targetBlob := client.ServiceClient().
		NewContainerClient(targetContainer).
		NewBlobClient(targetBlobPath)

	resp, err := targetBlob.StartCopyFromURL(ctx, sourceBlob.URL(), nil)
	if err != nil {
		return fmt.Errorf("start azure blob copy: %w", err)
	}

	result.TargetContainer = targetContainer
	result.TargetBlobPath = targetBlobPath
	if resp.CopyID != nil {
		result.CopyID = *resp.CopyID
	}
	if resp.CopyStatus != nil {
		result.CopyStatus = string(*resp.CopyStatus)
	}

	return nil
}

func targetBlobPathFromRequest(targetBlobPath string, targetPrefix string, sourceBlobPath string) string {
	if strings.TrimSpace(targetBlobPath) != "" {
		return strings.Trim(strings.TrimSpace(targetBlobPath), "/")
	}
	if strings.TrimSpace(targetPrefix) == "" {
		return strings.Trim(sourceBlobPath, "/")
	}
	return strings.Trim(path.Join(strings.Trim(targetPrefix, "/"), strings.Trim(sourceBlobPath, "/")), "/")
}

func accessTierFromRequest(value string, fallback blob.AccessTier) blob.AccessTier {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "archive":
		return blob.AccessTierArchive
	case "cool":
		return blob.AccessTierCool
	case "cold":
		return blob.AccessTierCold
	case "hot", "":
		return blob.AccessTierHot
	default:
		return fallback
	}
}

func rehydratePriorityFromRequest(value string) blob.RehydratePriority {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return blob.RehydratePriorityHigh
	default:
		return blob.RehydratePriorityStandard
	}
}
