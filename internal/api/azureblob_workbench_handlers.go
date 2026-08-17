package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

const (
	workbenchDefaultLimit      = 100
	workbenchDefaultPreviewLen = int64(4096)
)

type azureBlobWorkbenchRequest struct {
	Operation         string            `json:"operation"`
	SourceID          string            `json:"source_id,omitempty"`
	ConnectionString  string            `json:"connection_string,omitempty"`
	Container         string            `json:"container,omitempty"`
	Blob              string            `json:"blob,omitempty"`
	Prefix            string            `json:"prefix,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	Content           string            `json:"content,omitempty"`
	ContentBase64     string            `json:"content_base64,omitempty"`
	ContentType       string            `json:"content_type,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	TargetContainer   string            `json:"target_container,omitempty"`
	TargetPrefix      string            `json:"target_prefix,omitempty"`
	TargetBlob        string            `json:"target_blob,omitempty"`
	TargetTier        string            `json:"target_tier,omitempty"`
	RehydratePriority string            `json:"rehydrate_priority,omitempty"`
	MaxBytes          int64             `json:"max_bytes,omitempty"`
	Overwrite         bool              `json:"overwrite,omitempty"`
	DryRun            bool              `json:"dry_run,omitempty"`
	Confirm           string            `json:"confirm,omitempty"`
}

type azureBlobWorkbenchResponse struct {
	Operation       string                            `json:"operation"`
	Status          string                            `json:"status"`
	AuthMode        string                            `json:"auth_mode,omitempty"`
	Container       string                            `json:"container,omitempty"`
	Blob            string                            `json:"blob,omitempty"`
	Prefix          string                            `json:"prefix,omitempty"`
	TargetContainer string                            `json:"target_container,omitempty"`
	TargetBlob      string                            `json:"target_blob,omitempty"`
	TargetTier      string                            `json:"target_tier,omitempty"`
	Count           int                               `json:"count,omitempty"`
	DryRun          bool                              `json:"dry_run,omitempty"`
	Message         string                            `json:"message,omitempty"`
	Containers      []azureBlobWorkbenchContainer     `json:"containers,omitempty"`
	Items           []azureBlobWorkbenchBlob          `json:"items,omitempty"`
	Details         *azureBlobWorkbenchDetails        `json:"details,omitempty"`
	DownloadName    string                            `json:"download_name,omitempty"`
	ContentBase64   string                            `json:"content_base64,omitempty"`
	ContentText     string                            `json:"content_text,omitempty"`
	Applied         map[string]any                    `json:"applied,omitempty"`
	DACIntent       *execution.IntentEvaluationResult `json:"dac_intent,omitempty"`
}

type azureBlobWorkbenchContainer struct {
	Name         string            `json:"name"`
	LastModified string            `json:"last_modified,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	LeaseStatus  string            `json:"lease_status,omitempty"`
	LeaseState   string            `json:"lease_state,omitempty"`
	PublicAccess string            `json:"public_access,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type azureBlobWorkbenchBlob struct {
	Name         string `json:"name"`
	SizeBytes    int64  `json:"size_bytes"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	AccessTier   string `json:"access_tier,omitempty"`
	BlobType     string `json:"blob_type,omitempty"`
	TagCount     int32  `json:"tag_count,omitempty"`
}

type azureBlobWorkbenchDetails struct {
	Name         string                   `json:"name"`
	SizeBytes    int64                    `json:"size_bytes"`
	ContentType  string                   `json:"content_type,omitempty"`
	LastModified string                   `json:"last_modified,omitempty"`
	ETag         string                   `json:"etag,omitempty"`
	AccessTier   string                   `json:"access_tier,omitempty"`
	BlobType     string                   `json:"blob_type,omitempty"`
	Metadata     map[string]string        `json:"metadata,omitempty"`
	Tags         map[string]string        `json:"tags,omitempty"`
	TagsError    string                   `json:"tags_error,omitempty"`
	Preview      *azureBlobContentPreview `json:"preview,omitempty"`
	PreviewError string                   `json:"preview_error,omitempty"`
}

type azureBlobContentPreview struct {
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated"`
	Text      string `json:"text,omitempty"`
	Base64    string `json:"base64,omitempty"`
}

func (s *Server) azureBlobWorkbench(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req azureBlobWorkbenchRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	req.Operation = normalizeWorkbenchOperation(req.Operation)
	if !req.DryRun && !s.authorizeWorkbenchOperation(w, r, req.Operation) {
		return
	}

	actionMode := execution.ActionModeApply
	if req.DryRun || !workbenchOperationCreatesAction(req.Operation) {
		actionMode = execution.ActionModeDryRun
	}
	intent, err := s.evaluateDACIntent(r, dacIntentInput{
		Operation:       req.Operation,
		SourceID:        req.SourceID,
		ActionMode:      actionMode,
		Container:       req.Container,
		Prefix:          req.Prefix,
		Blob:            req.Blob,
		TargetContainer: req.TargetContainer,
		TargetBlob:      req.TargetBlob,
		Limit:           req.Limit,
		Metadata: map[string]string{
			"entrypoint":        "azure_blob_workbench",
			"auth_surface":      "advanced_workbench",
			"workbench_dry_run": fmt.Sprintf("%t", req.DryRun),
		},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if intent.Status != "ok" || intent.GuardianDecision == nil || !intent.GuardianDecision.Approved {
		writeJSON(w, http.StatusOK, &azureBlobWorkbenchResponse{
			Operation: req.Operation,
			Status:    "blocked",
			DryRun:    true,
			Message:   "Workbench operation blocked by Data Agent Cell Orchestrator.",
			DACIntent: intent,
		})
		return
	}

	resp, err := s.runAzureBlobWorkbench(r, req)
	if err != nil {
		if setupResp := workbenchSetupRequiredResponse(req, err); setupResp != nil {
			setupResp.DACIntent = intent
			writeJSON(w, http.StatusOK, setupResp)
			return
		}
		writeJSON(w, azureBlobWorkbenchHTTPStatus(err), map[string]string{"error": err.Error()})
		return
	}
	resp.DACIntent = intent
	s.persistAzureWorkbenchResponse(req, resp)

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) runAzureBlobWorkbench(r *http.Request, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	req.Operation = normalizeWorkbenchOperation(req.Operation)
	if req.Operation == "" {
		return nil, fmt.Errorf("operation is required")
	}

	client, authMode, err := s.azureBlobWorkbenchClient(req)
	if err != nil {
		return nil, err
	}

	switch req.Operation {
	case "list_containers":
		return s.azureBlobWorkbenchListContainers(r, client, authMode, req)
	case "list_blobs":
		return s.azureBlobWorkbenchListBlobs(r, client, authMode, req)
	case "view_blob":
		return s.azureBlobWorkbenchViewBlob(r, client, authMode, req)
	case "create_blob":
		return s.azureBlobWorkbenchUploadBlob(r, client, authMode, req, false)
	case "update_blob":
		return s.azureBlobWorkbenchUploadBlob(r, client, authMode, req, true)
	case "download_blob":
		return s.azureBlobWorkbenchDownloadBlob(r, client, authMode, req)
	case "set_metadata":
		return s.azureBlobWorkbenchSetMetadata(r, client, authMode, req)
	case "set_tags":
		return s.azureBlobWorkbenchSetTags(r, client, authMode, req)
	case "copy_blob":
		return s.azureBlobWorkbenchCopyBlob(r, client, authMode, req, false)
	case "move_blob":
		return s.azureBlobWorkbenchCopyBlob(r, client, authMode, req, true)
	case "set_tier", "rehydrate_blob":
		return s.azureBlobWorkbenchSetTier(r, client, authMode, req)
	case "create_container":
		return s.azureBlobWorkbenchCreateContainer(r, client, authMode, req)
	case "delete_container":
		return s.azureBlobWorkbenchDeleteContainer(r, client, authMode, req)
	case "delete_blob":
		return s.azureBlobWorkbenchDeleteBlob(r, client, authMode, req)
	default:
		return nil, fmt.Errorf("unsupported Azure Blob workbench operation: %s", req.Operation)
	}
}

func (s *Server) azureBlobWorkbenchClient(req azureBlobWorkbenchRequest) (*azblob.Client, string, error) {
	connectionString := strings.TrimSpace(req.ConnectionString)
	authMode := "request_connection_string"
	sourceID := strings.TrimSpace(req.SourceID)
	if connectionString == "" && sourceID != "" {
		if secret, err := s.secrets.Load(azureblob.SourceCredentialSecretName(sourceID)); err == nil && strings.TrimSpace(secret) != "" {
			connectionString = strings.TrimSpace(secret)
			authMode = "saved_source_access"
		}
	}
	if connectionString == "" {
		cfg := s.azureBlobConfig()
		connectionString = strings.TrimSpace(cfg.ConnectionString)
		authMode = "saved_connection_string"
	}
	if connectionString == "" {
		return nil, "", fmt.Errorf("Azure Blob account access is required. Select a saved source or add an Azure Blob source in Source Inventory")
	}

	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create Azure Blob client: %w", err)
	}
	return client, authMode, nil
}

func (s *Server) azureBlobWorkbenchListContainers(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = workbenchDefaultLimit
	}

	pager := client.NewListContainersPager(nil)
	containers := make([]azureBlobWorkbenchContainer, 0, workbenchMinInt(limit, workbenchDefaultLimit))
	for pager.More() && len(containers) < limit {
		page, err := pager.NextPage(r.Context())
		if err != nil {
			return nil, fmt.Errorf("list Azure Blob containers: %w", err)
		}
		for _, item := range page.ContainerItems {
			if item == nil || item.Name == nil {
				continue
			}
			if strings.TrimSpace(req.Prefix) != "" && !strings.HasPrefix(*item.Name, strings.TrimSpace(req.Prefix)) {
				continue
			}
			containers = append(containers, summarizeWorkbenchContainer(item))
			if len(containers) >= limit {
				break
			}
		}
	}

	return &azureBlobWorkbenchResponse{
		Operation:  req.Operation,
		Status:     "ok",
		AuthMode:   authMode,
		Prefix:     strings.TrimSpace(req.Prefix),
		Count:      len(containers),
		Containers: containers,
	}, nil
}

func (s *Server) azureBlobWorkbenchListBlobs(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, err := requireWorkbenchContainer(req)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = workbenchDefaultLimit
	}

	pager := client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Prefix:     workbenchOptionalString(strings.Trim(strings.TrimSpace(req.Prefix), "/")),
		MaxResults: to.Ptr(int32(workbenchMinInt(limit, 5000))),
	})

	items := make([]azureBlobWorkbenchBlob, 0, workbenchMinInt(limit, workbenchDefaultLimit))
	for pager.More() && len(items) < limit {
		page, err := pager.NextPage(r.Context())
		if err != nil {
			return nil, fmt.Errorf("list Azure Blob objects: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			if item == nil || item.Name == nil {
				continue
			}
			items = append(items, summarizeWorkbenchBlob(item))
			if len(items) >= limit {
				break
			}
		}
	}

	return &azureBlobWorkbenchResponse{
		Operation: req.Operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Prefix:    strings.Trim(strings.TrimSpace(req.Prefix), "/"),
		Count:     len(items),
		Items:     items,
	}, nil
}

func (s *Server) azureBlobWorkbenchViewBlob(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}

	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = workbenchDefaultPreviewLen
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, maxBytes)
	if err != nil {
		return nil, err
	}

	return &azureBlobWorkbenchResponse{
		Operation: req.Operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		Details:   details,
	}, nil
}

func (s *Server) azureBlobWorkbenchUploadBlob(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest, update bool) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}

	body, source, err := workbenchPayload(req)
	if err != nil {
		return nil, err
	}

	operation := "create_blob"
	if update {
		operation = "update_blob"
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	if !update && !req.Overwrite {
		exists, err := workbenchBlobExists(r.Context(), blobClient)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("blob already exists; use update or enable overwrite")
		}
	}

	if req.DryRun {
		return &azureBlobWorkbenchResponse{
			Operation: operation,
			Status:    "preview",
			AuthMode:  authMode,
			Container: containerName,
			Blob:      blobName,
			DryRun:    true,
			Message:   "No Azure Blob changes were made.",
			Applied: map[string]any{
				"would_upload_bytes": len(body),
				"source":             source,
				"content_type":       req.ContentType,
				"metadata":           workbenchSortedStringMap(req.Metadata),
				"tags":               workbenchSortedStringMap(req.Tags),
				"overwrite":          update || req.Overwrite,
			},
		}, nil
	}

	if _, err := client.UploadBuffer(r.Context(), containerName, blobName, body, nil); err != nil {
		return nil, fmt.Errorf("upload Azure Blob object: %w", err)
	}
	if strings.TrimSpace(req.ContentType) != "" {
		if _, err := blobClient.SetHTTPHeaders(r.Context(), blob.HTTPHeaders{BlobContentType: to.Ptr(strings.TrimSpace(req.ContentType))}, nil); err != nil {
			return nil, fmt.Errorf("set Azure Blob content type: %w", err)
		}
	}
	if len(req.Metadata) > 0 {
		if _, err := blobClient.SetMetadata(r.Context(), workbenchMetadataPointers(req.Metadata), nil); err != nil {
			return nil, fmt.Errorf("set Azure Blob metadata: %w", err)
		}
	}
	if len(req.Tags) > 0 {
		if _, err := blobClient.SetTags(r.Context(), req.Tags, nil); err != nil {
			return nil, fmt.Errorf("set Azure Blob index tags: %w", err)
		}
	}

	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, 0)
	if err != nil {
		return nil, err
	}

	return &azureBlobWorkbenchResponse{
		Operation: operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		Message:   "Blob content uploaded.",
		Details:   details,
		Applied: map[string]any{
			"uploaded_bytes": len(body),
			"source":         source,
			"content_type":   req.ContentType,
			"metadata":       workbenchSortedStringMap(req.Metadata),
			"tags":           workbenchSortedStringMap(req.Tags),
			"overwrite":      update || req.Overwrite,
		},
	}, nil
}

func (s *Server) azureBlobWorkbenchDownloadBlob(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	resp, err := blobClient.DownloadStream(r.Context(), nil)
	if err != nil {
		return nil, fmt.Errorf("download Azure Blob object: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Azure Blob object: %w", err)
	}

	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, 0)
	if err != nil {
		return nil, err
	}

	out := &azureBlobWorkbenchResponse{
		Operation:     req.Operation,
		Status:        "ok",
		AuthMode:      authMode,
		Container:     containerName,
		Blob:          blobName,
		Count:         len(data),
		Message:       "Blob downloaded into browser payload.",
		Details:       details,
		DownloadName:  filepath.Base(blobName),
		ContentBase64: base64.StdEncoding.EncodeToString(data),
	}
	if utf8.Valid(data) {
		out.ContentText = string(data)
	}
	return out, nil
}

func (s *Server) azureBlobWorkbenchSetMetadata(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}
	if len(req.Metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}

	if req.DryRun {
		return workbenchPreview(req.Operation, authMode, containerName, blobName, map[string]any{"metadata": workbenchSortedStringMap(req.Metadata)}), nil
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	if _, err := blobClient.SetMetadata(r.Context(), workbenchMetadataPointers(req.Metadata), nil); err != nil {
		return nil, fmt.Errorf("set Azure Blob metadata: %w", err)
	}

	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, 0)
	if err != nil {
		return nil, err
	}
	return &azureBlobWorkbenchResponse{
		Operation: req.Operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		Message:   "Blob metadata updated.",
		Details:   details,
		Applied:   map[string]any{"metadata": workbenchSortedStringMap(req.Metadata)},
	}, nil
}

func (s *Server) azureBlobWorkbenchSetTags(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}
	if len(req.Tags) == 0 {
		return nil, fmt.Errorf("tags are required")
	}

	if req.DryRun {
		return workbenchPreview(req.Operation, authMode, containerName, blobName, map[string]any{"tags": workbenchSortedStringMap(req.Tags)}), nil
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	if _, err := blobClient.SetTags(r.Context(), req.Tags, nil); err != nil {
		return nil, fmt.Errorf("set Azure Blob index tags: %w", err)
	}

	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, 0)
	if err != nil {
		return nil, err
	}
	return &azureBlobWorkbenchResponse{
		Operation: req.Operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		Message:   "Blob index tags updated.",
		Details:   details,
		Applied:   map[string]any{"tags": workbenchSortedStringMap(req.Tags)},
	}, nil
}

func (s *Server) azureBlobWorkbenchCopyBlob(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest, movePrep bool) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}
	targetContainer := strings.TrimSpace(req.TargetContainer)
	if targetContainer == "" {
		return nil, fmt.Errorf("target_container is required")
	}
	targetBlob := workbenchTargetBlobPath(req.TargetBlob, req.TargetPrefix, blobName)
	if targetBlob == "" {
		return nil, fmt.Errorf("target blob path could not be resolved")
	}
	if targetContainer == containerName && targetBlob == blobName {
		return nil, fmt.Errorf("target must differ from source")
	}

	operation := "copy_blob"
	if movePrep {
		operation = "move_blob"
	}
	applied := map[string]any{
		"source_container":       containerName,
		"source_blob":            blobName,
		"target_container":       targetContainer,
		"target_blob":            targetBlob,
		"source_delete_required": movePrep,
	}
	if req.DryRun {
		return &azureBlobWorkbenchResponse{
			Operation:       operation,
			Status:          "preview",
			AuthMode:        authMode,
			Container:       containerName,
			Blob:            blobName,
			TargetContainer: targetContainer,
			TargetBlob:      targetBlob,
			DryRun:          true,
			Message:         "No Azure Blob changes were made.",
			Applied:         applied,
		}, nil
	}
	if movePrep && strings.ToUpper(strings.TrimSpace(req.Confirm)) != "MOVE" {
		return nil, fmt.Errorf("move preparation requires confirmation text MOVE")
	}

	sourceBlob := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	targetBlobClient := client.ServiceClient().NewContainerClient(targetContainer).NewBlobClient(targetBlob)
	copyResp, err := targetBlobClient.StartCopyFromURL(r.Context(), sourceBlob.URL(), nil)
	if err != nil {
		return nil, fmt.Errorf("start Azure Blob copy: %w", err)
	}
	if copyResp.CopyID != nil {
		applied["copy_id"] = *copyResp.CopyID
	}
	if copyResp.CopyStatus != nil {
		applied["copy_status"] = string(*copyResp.CopyStatus)
	}
	message := "Azure Blob copy started."
	if movePrep {
		message = "Azure Blob move prepared as copy-first. Delete the source only after the copied object is verified and approved."
	}
	return &azureBlobWorkbenchResponse{
		Operation:       operation,
		Status:          "ok",
		AuthMode:        authMode,
		Container:       containerName,
		Blob:            blobName,
		TargetContainer: targetContainer,
		TargetBlob:      targetBlob,
		Message:         message,
		Applied:         applied,
	}, nil
}

func (s *Server) azureBlobWorkbenchSetTier(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}
	targetTier := strings.TrimSpace(req.TargetTier)
	if targetTier == "" {
		targetTier = "hot"
	}
	tier := workbenchAccessTier(targetTier)
	priority := workbenchRehydratePriority(req.RehydratePriority)
	applied := map[string]any{
		"target_tier":        string(tier),
		"rehydrate_priority": string(priority),
	}
	if req.DryRun {
		return &azureBlobWorkbenchResponse{
			Operation:  req.Operation,
			Status:     "preview",
			AuthMode:   authMode,
			Container:  containerName,
			Blob:       blobName,
			TargetTier: string(tier),
			DryRun:     true,
			Message:    "No Azure Blob tier changes were made.",
			Applied:    applied,
		}, nil
	}
	if strings.ToUpper(strings.TrimSpace(req.Confirm)) != "APPLY" {
		return nil, fmt.Errorf("tier and rehydrate operations require confirmation text APPLY")
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	if _, err := blobClient.SetTier(r.Context(), tier, &blob.SetTierOptions{RehydratePriority: to.Ptr(priority)}); err != nil {
		return nil, fmt.Errorf("set Azure Blob access tier: %w", err)
	}
	details, err := readWorkbenchBlobDetails(r.Context(), blobClient, blobName, 0)
	if err != nil {
		return nil, err
	}
	return &azureBlobWorkbenchResponse{
		Operation:  req.Operation,
		Status:     "ok",
		AuthMode:   authMode,
		Container:  containerName,
		Blob:       blobName,
		TargetTier: string(tier),
		Message:    "Azure Blob access tier change requested.",
		Details:    details,
		Applied:    applied,
	}, nil
}

func (s *Server) azureBlobWorkbenchCreateContainer(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, err := requireWorkbenchContainer(req)
	if err != nil {
		return nil, err
	}
	if req.DryRun {
		return &azureBlobWorkbenchResponse{
			Operation: "create_container",
			Status:    "preview",
			AuthMode:  authMode,
			Container: containerName,
			DryRun:    true,
			Message:   "No Azure Blob container was created.",
			Applied:   map[string]any{"would_create_container": containerName},
		}, nil
	}
	if _, err := client.CreateContainer(r.Context(), containerName, nil); err != nil {
		return nil, fmt.Errorf("create Azure Blob container: %w", err)
	}
	return &azureBlobWorkbenchResponse{
		Operation: "create_container",
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Message:   "Azure Blob container created.",
		Containers: []azureBlobWorkbenchContainer{{
			Name: containerName,
		}},
		Applied: map[string]any{"created_container": containerName},
	}, nil
}

func (s *Server) azureBlobWorkbenchDeleteContainer(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, err := requireWorkbenchContainer(req)
	if err != nil {
		return nil, err
	}
	if req.DryRun {
		return &azureBlobWorkbenchResponse{
			Operation: "delete_container",
			Status:    "preview",
			AuthMode:  authMode,
			Container: containerName,
			DryRun:    true,
			Message:   "No Azure Blob container was deleted.",
			Applied: map[string]any{
				"would_delete_container": containerName,
				"required_confirm":       "DELETE_CONTAINER",
			},
		}, nil
	}
	if strings.ToUpper(strings.TrimSpace(req.Confirm)) != "DELETE_CONTAINER" {
		return nil, fmt.Errorf("delete container requires confirmation text DELETE_CONTAINER")
	}
	if _, err := client.DeleteContainer(r.Context(), containerName, nil); err != nil {
		return nil, fmt.Errorf("delete Azure Blob container: %w", err)
	}
	return &azureBlobWorkbenchResponse{
		Operation: "delete_container",
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Message:   "Azure Blob container marked for deletion.",
		Applied:   map[string]any{"deleted_container": containerName},
	}, nil
}

func (s *Server) azureBlobWorkbenchDeleteBlob(r *http.Request, client *azblob.Client, authMode string, req azureBlobWorkbenchRequest) (*azureBlobWorkbenchResponse, error) {
	containerName, blobName, err := requireWorkbenchBlob(req)
	if err != nil {
		return nil, err
	}

	if req.DryRun {
		return workbenchPreview(req.Operation, authMode, containerName, blobName, map[string]any{
			"would_delete_snapshots": true,
			"required_confirm":       "DELETE",
		}), nil
	}
	if req.Confirm != "DELETE" {
		return nil, fmt.Errorf("delete requires confirmation text DELETE")
	}

	blobClient := client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	if _, err := blobClient.Delete(r.Context(), &blob.DeleteOptions{
		DeleteSnapshots: to.Ptr(blob.DeleteSnapshotsOptionTypeInclude),
	}); err != nil {
		return nil, fmt.Errorf("delete Azure Blob object: %w", err)
	}

	return &azureBlobWorkbenchResponse{
		Operation: req.Operation,
		Status:    "ok",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		Message:   "Blob deleted with snapshots included.",
		Applied:   map[string]any{"deleted": true, "snapshots": "include"},
	}, nil
}

func workbenchPreview(operation, authMode, containerName, blobName string, applied map[string]any) *azureBlobWorkbenchResponse {
	return &azureBlobWorkbenchResponse{
		Operation: operation,
		Status:    "preview",
		AuthMode:  authMode,
		Container: containerName,
		Blob:      blobName,
		DryRun:    true,
		Message:   "No Azure Blob changes were made.",
		Applied:   applied,
	}
}

func workbenchSetupRequiredResponse(req azureBlobWorkbenchRequest, err error) *azureBlobWorkbenchResponse {
	if err == nil {
		return nil
	}
	operation := normalizeWorkbenchOperation(req.Operation)
	if operation == "" {
		operation = "list_containers"
	}
	switch operation {
	case "list_containers", "list_blobs":
	default:
		return nil
	}
	message := err.Error()
	if !strings.Contains(message, "Azure Blob account access is required") &&
		!strings.Contains(message, "container is required") &&
		!strings.Contains(message, "operation is required") {
		return nil
	}
	return &azureBlobWorkbenchResponse{
		Operation: operation,
		Status:    "setup_required",
		Container: strings.TrimSpace(req.Container),
		Prefix:    strings.Trim(strings.TrimSpace(req.Prefix), "/"),
		DryRun:    true,
		Message:   message,
	}
}
func readWorkbenchBlobDetails(ctx anyContext, blobClient *blob.Client, blobName string, maxBytes int64) (*azureBlobWorkbenchDetails, error) {
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get Azure Blob properties: %w", err)
	}

	details := &azureBlobWorkbenchDetails{
		Name:         blobName,
		SizeBytes:    workbenchValueInt64(props.ContentLength),
		ContentType:  workbenchValueString(props.ContentType),
		LastModified: workbenchValueTime(props.LastModified),
		ETag:         workbenchValueETag(props.ETag),
		AccessTier:   workbenchValueString(props.AccessTier),
		Metadata:     workbenchPointerMapToStringMap(props.Metadata),
	}
	if props.BlobType != nil {
		details.BlobType = string(*props.BlobType)
	}

	tags, err := blobClient.GetTags(ctx, nil)
	if err != nil {
		details.TagsError = err.Error()
	} else {
		details.Tags = workbenchTagsToMap(tags.BlobTagSet)
	}

	if maxBytes > 0 {
		preview, err := readWorkbenchPreview(ctx, blobClient, maxBytes)
		if err != nil {
			details.PreviewError = err.Error()
		} else {
			details.Preview = preview
		}
	}

	return details, nil
}

type anyContext interface {
	Done() <-chan struct{}
	Err() error
	Value(key any) any
	Deadline() (deadline time.Time, ok bool)
}

func readWorkbenchPreview(ctx anyContext, blobClient *blob.Client, maxBytes int64) (*azureBlobContentPreview, error) {
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("download preview stream: %w", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	n, err := io.CopyN(&buf, resp.Body, maxBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	data := buf.Bytes()
	truncated := n > maxBytes
	if truncated {
		data = data[:maxBytes]
	}

	preview := &azureBlobContentPreview{Bytes: len(data), Truncated: truncated}
	if utf8.Valid(data) {
		preview.Text = string(data)
	} else {
		preview.Base64 = base64.StdEncoding.EncodeToString(data)
	}
	return preview, nil
}

func workbenchPayload(req azureBlobWorkbenchRequest) ([]byte, string, error) {
	if strings.TrimSpace(req.ContentBase64) != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64))
		if err != nil {
			return nil, "", fmt.Errorf("decode content_base64: %w", err)
		}
		return data, "browser_file", nil
	}
	if req.Content != "" {
		return []byte(req.Content), "inline", nil
	}
	generated := fmt.Sprintf("{\n  \"tool\": \"azure_blob_workbench\",\n  \"created_at\": %q,\n  \"purpose\": \"Everest Azure Blob UI validation\"\n}\n", time.Now().UTC().Format(time.RFC3339))
	return []byte(generated), "generated", nil
}

func workbenchBlobExists(ctx anyContext, blobClient *blob.Client) (bool, error) {
	if _, err := blobClient.GetProperties(ctx, nil); err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("check Azure Blob existence: %w", err)
	}
	return true, nil
}

func requireWorkbenchContainer(req azureBlobWorkbenchRequest) (string, error) {
	containerName := strings.TrimSpace(req.Container)
	if containerName == "" {
		return "", fmt.Errorf("container is required")
	}
	return containerName, nil
}

func requireWorkbenchBlob(req azureBlobWorkbenchRequest) (string, string, error) {
	containerName, err := requireWorkbenchContainer(req)
	if err != nil {
		return "", "", err
	}
	blobName := strings.Trim(strings.TrimSpace(req.Blob), "/")
	if blobName == "" {
		return "", "", fmt.Errorf("blob path is required")
	}
	return containerName, blobName, nil
}

func normalizeWorkbenchOperation(operation string) string {
	operation = strings.ToLower(strings.TrimSpace(operation))
	operation = strings.ReplaceAll(operation, "-", "_")
	switch operation {
	case "containers", "listcontainers", "list_container":
		return "list_containers"
	case "list", "list_objects", "list_blobs":
		return "list_blobs"
	case "view", "get", "view_object", "view_blob":
		return "view_blob"
	case "create", "upload", "create_blob":
		return "create_blob"
	case "update", "overwrite", "update_blob":
		return "update_blob"
	case "download", "download_blob":
		return "download_blob"
	case "metadata", "set_metadata":
		return "set_metadata"
	case "tags", "set_tags":
		return "set_tags"
	case "copy", "copy_object", "copy_blob":
		return "copy_blob"
	case "move", "move_object", "move_blob":
		return "move_blob"
	case "tier", "set_tier", "set_access_tier":
		return "set_tier"
	case "rehydrate", "retrieve", "rehydrate_blob":
		return "rehydrate_blob"
	case "create_container", "container_create":
		return "create_container"
	case "delete_container", "container_delete":
		return "delete_container"
	case "delete", "delete_blob":
		return "delete_blob"
	default:
		return operation
	}
}

func workbenchTargetBlobPath(targetBlob string, targetPrefix string, sourceBlob string) string {
	if strings.TrimSpace(targetBlob) != "" {
		return strings.Trim(strings.TrimSpace(targetBlob), "/")
	}
	sourceBlob = strings.Trim(strings.TrimSpace(sourceBlob), "/")
	if sourceBlob == "" {
		return ""
	}
	if strings.TrimSpace(targetPrefix) == "" {
		return sourceBlob
	}
	return strings.Trim(path.Join(strings.Trim(targetPrefix, "/"), sourceBlob), "/")
}

func workbenchAccessTier(value string) blob.AccessTier {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "archive":
		return blob.AccessTierArchive
	case "cool":
		return blob.AccessTierCool
	case "cold":
		return blob.AccessTierCold
	default:
		return blob.AccessTierHot
	}
}

func workbenchRehydratePriority(value string) blob.RehydratePriority {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return blob.RehydratePriorityHigh
	default:
		return blob.RehydratePriorityStandard
	}
}

func summarizeWorkbenchContainer(item *service.ContainerItem) azureBlobWorkbenchContainer {
	summary := azureBlobWorkbenchContainer{
		Name:     workbenchValueString(item.Name),
		Metadata: workbenchPointerMapToStringMap(item.Metadata),
	}
	if item.Properties == nil {
		return summary
	}
	props := item.Properties
	summary.LastModified = workbenchValueTime(props.LastModified)
	summary.ETag = workbenchValueETag(props.ETag)
	if props.LeaseStatus != nil {
		summary.LeaseStatus = string(*props.LeaseStatus)
	}
	if props.LeaseState != nil {
		summary.LeaseState = string(*props.LeaseState)
	}
	if props.PublicAccess != nil {
		summary.PublicAccess = string(*props.PublicAccess)
	}
	return summary
}

func summarizeWorkbenchBlob(item *container.BlobItem) azureBlobWorkbenchBlob {
	summary := azureBlobWorkbenchBlob{Name: workbenchValueString(item.Name)}
	if item.Properties == nil {
		return summary
	}
	props := item.Properties
	summary.SizeBytes = workbenchValueInt64(props.ContentLength)
	summary.ContentType = workbenchValueString(props.ContentType)
	summary.LastModified = workbenchValueTime(props.LastModified)
	if props.AccessTier != nil {
		summary.AccessTier = string(*props.AccessTier)
	}
	if props.BlobType != nil {
		summary.BlobType = string(*props.BlobType)
	}
	if props.TagCount != nil {
		summary.TagCount = *props.TagCount
	}
	return summary
}

func workbenchTagsToMap(tags []*blob.Tags) map[string]string {
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		if tag == nil || tag.Key == nil {
			continue
		}
		out[*tag.Key] = workbenchValueString(tag.Value)
	}
	return workbenchSortedStringMap(out)
}

func workbenchMetadataPointers(metadata map[string]string) map[string]*string {
	out := make(map[string]*string, len(metadata))
	for key, value := range metadata {
		valueCopy := value
		out[key] = &valueCopy
	}
	return out
}

func workbenchPointerMapToStringMap(input map[string]*string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = workbenchValueString(value)
	}
	return workbenchSortedStringMap(out)
}

func workbenchSortedStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(input))
	for _, key := range keys {
		out[key] = input[key]
	}
	return out
}

func workbenchOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return to.Ptr(value)
}

func workbenchValueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func workbenchValueInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func workbenchValueTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func workbenchValueETag(value *azcore.ETag) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func workbenchMinInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func azureBlobWorkbenchHTTPStatus(err error) int {
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) && responseErr.StatusCode > 0 {
		return responseErr.StatusCode
	}
	return http.StatusBadRequest
}
