package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

const (
	defaultListLimit   = 100
	defaultPreviewSize = int64(4096)
	defaultTimeout     = 60 * time.Second
)

type config struct {
	Operation        string
	ConnectionString string
	Container        string
	ContainerURL     string
	ServiceURL       string
	BlobName         string
	Prefix           string
	Limit            int
	Content          string
	File             string
	Output           string
	ContentType      string
	Metadata         string
	Tags             string
	Overwrite        bool
	DryRun           bool
	Confirm          string
	MaxBytes         int64
	Timeout          time.Duration
}

type commandOutput struct {
	Operation  string             `json:"operation"`
	Status     string             `json:"status"`
	AuthMode   string             `json:"auth_mode,omitempty"`
	Container  string             `json:"container,omitempty"`
	Blob       string             `json:"blob,omitempty"`
	Prefix     string             `json:"prefix,omitempty"`
	Count      int                `json:"count,omitempty"`
	DryRun     bool               `json:"dry_run,omitempty"`
	Message    string             `json:"message,omitempty"`
	Items      []blobSummary      `json:"items,omitempty"`
	Containers []containerSummary `json:"containers,omitempty"`
	Details    *blobDetails       `json:"details,omitempty"`
	Applied    map[string]any     `json:"applied,omitempty"`
}

type containerSummary struct {
	Name         string            `json:"name"`
	LastModified string            `json:"last_modified,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	LeaseStatus  string            `json:"lease_status,omitempty"`
	LeaseState   string            `json:"lease_state,omitempty"`
	PublicAccess string            `json:"public_access,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type blobSummary struct {
	Name         string `json:"name"`
	SizeBytes    int64  `json:"size_bytes"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	AccessTier   string `json:"access_tier,omitempty"`
	BlobType     string `json:"blob_type,omitempty"`
	TagCount     int32  `json:"tag_count,omitempty"`
}

type blobDetails struct {
	Name          string            `json:"name"`
	SizeBytes     int64             `json:"size_bytes"`
	ContentType   string            `json:"content_type,omitempty"`
	LastModified  string            `json:"last_modified,omitempty"`
	ETag          string            `json:"etag,omitempty"`
	AccessTier    string            `json:"access_tier,omitempty"`
	BlobType      string            `json:"blob_type,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	TagsError     string            `json:"tags_error,omitempty"`
	Preview       *contentPreview   `json:"preview,omitempty"`
	PreviewError  string            `json:"preview_error,omitempty"`
	DownloadedTo  string            `json:"downloaded_to,omitempty"`
	DownloadedLen int64             `json:"downloaded_bytes,omitempty"`
}

type contentPreview struct {
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated"`
	Text      string `json:"text,omitempty"`
	Base64    string `json:"base64,omitempty"`
}

type clientScope struct {
	Client    *container.Client
	AuthMode  string
	Container string
}

type payload struct {
	Bytes       []byte
	Source      string
	ContentType string
}

func main() {
	cfg := parseFlags()
	if cfg.Operation == "" {
		exitWithUsage("missing required --op")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "azureblob-test-tool: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.Operation, "op", "", "operation: list-containers, list, view, create, update, download, set-metadata, set-tags, delete")
	flag.StringVar(&cfg.ConnectionString, "connection-string", "", "Azure Storage connection string. Defaults to AZURE_STORAGE_CONNECTION_STRING.")
	flag.StringVar(&cfg.Container, "container", "", "container name when using --connection-string or --service-url")
	flag.StringVar(&cfg.ContainerURL, "container-url", "", "full container URL, usually with SAS; defaults to AZURE_BLOB_CONTAINER_URL")
	flag.StringVar(&cfg.ServiceURL, "service-url", "", "account/service Blob URL, usually with SAS; requires --container")
	flag.StringVar(&cfg.BlobName, "blob", "", "blob path/name for single-blob operations")
	flag.StringVar(&cfg.Prefix, "prefix", "", "blob prefix filter for list")
	flag.IntVar(&cfg.Limit, "limit", defaultListLimit, "maximum list results")
	flag.StringVar(&cfg.Content, "content", "", "inline content for create/update")
	flag.StringVar(&cfg.File, "file", "", "source file for create/update")
	flag.StringVar(&cfg.Output, "out", "", "destination file for download")
	flag.StringVar(&cfg.ContentType, "content-type", "", "content type to apply on create/update")
	flag.StringVar(&cfg.Metadata, "metadata", "", "comma-separated metadata, for example owner=qa,purpose=everest")
	flag.StringVar(&cfg.Tags, "tags", "", "comma-separated index tags, for example everest_test=true,case=scan")
	flag.BoolVar(&cfg.Overwrite, "overwrite", false, "allow create to overwrite an existing blob")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "show intended write/delete action without changing Azure Blob")
	flag.StringVar(&cfg.Confirm, "confirm", "", "required value DELETE for delete")
	flag.Int64Var(&cfg.MaxBytes, "max-bytes", defaultPreviewSize, "maximum bytes to preview for view; set 0 to disable preview")
	flag.DurationVar(&cfg.Timeout, "timeout", defaultTimeout, "operation timeout")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run ./cmd/azureblob-test-tool --op list-containers\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run ./cmd/azureblob-test-tool --op list --container <container>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run ./cmd/azureblob-test-tool --op create --container <container> --blob everest-test/sample.txt --content \"hello\"\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run ./cmd/azureblob-test-tool --op view --container-url \"https://account.blob.core.windows.net/container?<sas>\" --blob everest-test/sample.txt\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if cfg.ConnectionString == "" {
		cfg.ConnectionString = os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	}
	if cfg.ContainerURL == "" {
		cfg.ContainerURL = os.Getenv("AZURE_BLOB_CONTAINER_URL")
	}
	cfg.Operation = strings.ToLower(strings.TrimSpace(cfg.Operation))
	cfg.Container = strings.TrimSpace(cfg.Container)
	cfg.BlobName = strings.Trim(strings.TrimSpace(cfg.BlobName), "/")
	cfg.Prefix = strings.Trim(strings.TrimSpace(cfg.Prefix), "/")

	return cfg
}

func exitWithUsage(message string) {
	fmt.Fprintf(os.Stderr, "azureblob-test-tool: %s\n\n", message)
	flag.Usage()
	os.Exit(2)
}

func run(ctx context.Context, cfg config) error {
	if isListContainersOperation(cfg.Operation) {
		return listContainers(ctx, cfg)
	}

	scope, err := newClientScope(cfg)
	if err != nil {
		return err
	}

	switch cfg.Operation {
	case "list":
		return listBlobs(ctx, scope, cfg)
	case "view", "get":
		return viewBlob(ctx, scope, cfg)
	case "create":
		return uploadBlob(ctx, scope, cfg, false)
	case "update":
		return uploadBlob(ctx, scope, cfg, true)
	case "download":
		return downloadBlob(ctx, scope, cfg)
	case "set-metadata", "metadata":
		return setMetadata(ctx, scope, cfg)
	case "set-tags", "tags":
		return setTags(ctx, scope, cfg)
	case "delete":
		return deleteBlob(ctx, scope, cfg)
	default:
		return fmt.Errorf("unsupported --op %q", cfg.Operation)
	}
}

func isListContainersOperation(operation string) bool {
	switch operation {
	case "list-containers", "containers", "container-list":
		return true
	default:
		return false
	}
}

func newAccountClient(cfg config) (*azblob.Client, string, error) {
	switch {
	case strings.TrimSpace(cfg.ContainerURL) != "":
		return nil, "", fmt.Errorf("--container-url points to one container and cannot discover sibling containers; use --connection-string or --service-url")
	case strings.TrimSpace(cfg.ServiceURL) != "":
		client, err := azblob.NewClientWithNoCredential(strings.TrimSpace(cfg.ServiceURL), nil)
		if err != nil {
			return nil, "", fmt.Errorf("create account client from --service-url: %w", err)
		}
		return client, "service_url", nil
	case strings.TrimSpace(cfg.ConnectionString) != "":
		client, err := azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
		if err != nil {
			return nil, "", fmt.Errorf("create account client from connection string: %w", err)
		}
		return client, "connection_string", nil
	default:
		return nil, "", fmt.Errorf("provide --connection-string, AZURE_STORAGE_CONNECTION_STRING, or --service-url")
	}
}

func newClientScope(cfg config) (*clientScope, error) {
	switch {
	case strings.TrimSpace(cfg.ContainerURL) != "":
		client, err := container.NewClientWithNoCredential(strings.TrimSpace(cfg.ContainerURL), nil)
		if err != nil {
			return nil, fmt.Errorf("create container client from --container-url: %w", err)
		}
		return &clientScope{
			Client:    client,
			AuthMode:  "container_url",
			Container: containerNameFromURL(cfg.ContainerURL),
		}, nil

	case strings.TrimSpace(cfg.ServiceURL) != "":
		if cfg.Container == "" {
			return nil, fmt.Errorf("--container is required with --service-url")
		}
		containerURL, err := serviceContainerURL(cfg.ServiceURL, cfg.Container)
		if err != nil {
			return nil, err
		}
		client, err := container.NewClientWithNoCredential(containerURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create container client from --service-url: %w", err)
		}
		return &clientScope{
			Client:    client,
			AuthMode:  "service_url",
			Container: cfg.Container,
		}, nil

	case strings.TrimSpace(cfg.ConnectionString) != "":
		if cfg.Container == "" {
			return nil, fmt.Errorf("--container is required with --connection-string or AZURE_STORAGE_CONNECTION_STRING")
		}
		client, err := container.NewClientFromConnectionString(cfg.ConnectionString, cfg.Container, nil)
		if err != nil {
			return nil, fmt.Errorf("create container client from connection string: %w", err)
		}
		return &clientScope{
			Client:    client,
			AuthMode:  "connection_string",
			Container: cfg.Container,
		}, nil

	default:
		return nil, fmt.Errorf("provide --connection-string, AZURE_STORAGE_CONNECTION_STRING, --container-url, AZURE_BLOB_CONTAINER_URL, or --service-url")
	}
}

func listContainers(ctx context.Context, cfg config) error {
	client, authMode, err := newAccountClient(cfg)
	if err != nil {
		return err
	}
	if cfg.Limit <= 0 {
		cfg.Limit = defaultListLimit
	}

	pager := client.NewListContainersPager(nil)
	containers := make([]containerSummary, 0, min(cfg.Limit, defaultListLimit))

	for pager.More() && len(containers) < cfg.Limit {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list containers: %w", err)
		}

		for _, item := range page.ContainerItems {
			if item == nil || item.Name == nil {
				continue
			}
			if cfg.Prefix != "" && !strings.HasPrefix(*item.Name, cfg.Prefix) {
				continue
			}
			containers = append(containers, summarizeContainerItem(item))
			if len(containers) >= cfg.Limit {
				break
			}
		}
	}

	return writeJSON(commandOutput{
		Operation:  "list-containers",
		Status:     "ok",
		AuthMode:   authMode,
		Prefix:     cfg.Prefix,
		Count:      len(containers),
		Containers: containers,
	})
}

func listBlobs(ctx context.Context, scope *clientScope, cfg config) error {
	if cfg.Limit <= 0 {
		cfg.Limit = defaultListLimit
	}

	pager := scope.Client.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix:     optionalString(cfg.Prefix),
		MaxResults: to.Ptr(int32(min(cfg.Limit, 5000))),
	})

	items := make([]blobSummary, 0, min(cfg.Limit, defaultListLimit))
	for pager.More() && len(items) < cfg.Limit {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list blobs: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			if item == nil || item.Name == nil {
				continue
			}
			items = append(items, summarizeBlobItem(item))
			if len(items) >= cfg.Limit {
				break
			}
		}
	}

	return writeJSON(commandOutput{
		Operation: "list",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Prefix:    cfg.Prefix,
		Count:     len(items),
		Items:     items,
	})
}

func viewBlob(ctx context.Context, scope *clientScope, cfg config) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}

	details, err := readBlobDetails(ctx, scope.Client.NewBlobClient(cfg.BlobName), cfg.BlobName, cfg.MaxBytes)
	if err != nil {
		return err
	}

	return writeJSON(commandOutput{
		Operation: "view",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Details:   details,
	})
}

func uploadBlob(ctx context.Context, scope *clientScope, cfg config, update bool) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}

	body, err := readPayload(cfg)
	if err != nil {
		return err
	}
	metadata, err := parseKVList(cfg.Metadata)
	if err != nil {
		return fmt.Errorf("parse --metadata: %w", err)
	}
	tags, err := parseKVList(cfg.Tags)
	if err != nil {
		return fmt.Errorf("parse --tags: %w", err)
	}

	operation := "create"
	if update {
		operation = "update"
	}

	blobClient := scope.Client.NewBlobClient(cfg.BlobName)
	if !update && !cfg.Overwrite {
		exists, err := blobExists(ctx, blobClient)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("blob %q already exists; use --op update or add --overwrite", cfg.BlobName)
		}
	}

	if cfg.DryRun {
		return writeJSON(commandOutput{
			Operation: operation,
			Status:    "preview",
			AuthMode:  scope.AuthMode,
			Container: scope.Container,
			Blob:      cfg.BlobName,
			DryRun:    true,
			Message:   "No Azure Blob changes were made.",
			Applied: map[string]any{
				"would_upload_bytes": len(body.Bytes),
				"source":             body.Source,
				"content_type":       body.ContentType,
				"metadata":           metadata,
				"tags":               tags,
				"overwrite":          update || cfg.Overwrite,
			},
		})
	}

	blockClient := scope.Client.NewBlockBlobClient(cfg.BlobName)
	if _, err := blockClient.UploadBuffer(ctx, body.Bytes, nil); err != nil {
		return fmt.Errorf("upload blob content: %w", err)
	}
	if body.ContentType != "" {
		if _, err := blockClient.SetHTTPHeaders(ctx, blob.HTTPHeaders{
			BlobContentType: to.Ptr(body.ContentType),
		}, nil); err != nil {
			return fmt.Errorf("set content type: %w", err)
		}
	}
	if len(metadata) > 0 {
		if _, err := blockClient.SetMetadata(ctx, toMetadataPointers(metadata), nil); err != nil {
			return fmt.Errorf("set metadata: %w", err)
		}
	}
	if len(tags) > 0 {
		if _, err := blockClient.SetTags(ctx, tags, nil); err != nil {
			return fmt.Errorf("set tags: %w", err)
		}
	}

	details, err := readBlobDetails(ctx, blobClient, cfg.BlobName, 0)
	if err != nil {
		return err
	}

	return writeJSON(commandOutput{
		Operation: operation,
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Message:   "Blob content uploaded.",
		Details:   details,
		Applied: map[string]any{
			"uploaded_bytes": len(body.Bytes),
			"source":         body.Source,
			"content_type":   body.ContentType,
			"metadata":       metadata,
			"tags":           tags,
			"overwrite":      update || cfg.Overwrite,
		},
	})
}

func downloadBlob(ctx context.Context, scope *clientScope, cfg config) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Output) == "" {
		return fmt.Errorf("--out is required for download")
	}

	if err := os.MkdirAll(filepath.Dir(cleanOutputPath(cfg.Output)), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	blobClient := scope.Client.NewBlobClient(cfg.BlobName)
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return fmt.Errorf("download blob stream: %w", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(cleanOutputPath(cfg.Output))
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	details, err := readBlobDetails(ctx, blobClient, cfg.BlobName, 0)
	if err != nil {
		return err
	}
	details.DownloadedTo = cleanOutputPath(cfg.Output)
	details.DownloadedLen = written

	return writeJSON(commandOutput{
		Operation: "download",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Message:   "Blob downloaded.",
		Details:   details,
	})
}

func setMetadata(ctx context.Context, scope *clientScope, cfg config) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}
	metadata, err := parseKVList(cfg.Metadata)
	if err != nil {
		return fmt.Errorf("parse --metadata: %w", err)
	}
	if len(metadata) == 0 {
		return fmt.Errorf("--metadata must include at least one key=value pair")
	}

	if cfg.DryRun {
		return writeJSON(previewOutput("set-metadata", scope, cfg, map[string]any{"metadata": metadata}))
	}

	blobClient := scope.Client.NewBlobClient(cfg.BlobName)
	if _, err := blobClient.SetMetadata(ctx, toMetadataPointers(metadata), nil); err != nil {
		return fmt.Errorf("set metadata: %w", err)
	}

	details, err := readBlobDetails(ctx, blobClient, cfg.BlobName, 0)
	if err != nil {
		return err
	}

	return writeJSON(commandOutput{
		Operation: "set-metadata",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Message:   "Blob metadata updated.",
		Details:   details,
		Applied:   map[string]any{"metadata": metadata},
	})
}

func setTags(ctx context.Context, scope *clientScope, cfg config) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}
	tags, err := parseKVList(cfg.Tags)
	if err != nil {
		return fmt.Errorf("parse --tags: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("--tags must include at least one key=value pair")
	}

	if cfg.DryRun {
		return writeJSON(previewOutput("set-tags", scope, cfg, map[string]any{"tags": tags}))
	}

	blobClient := scope.Client.NewBlobClient(cfg.BlobName)
	if _, err := blobClient.SetTags(ctx, tags, nil); err != nil {
		return fmt.Errorf("set tags: %w", err)
	}

	details, err := readBlobDetails(ctx, blobClient, cfg.BlobName, 0)
	if err != nil {
		return err
	}

	return writeJSON(commandOutput{
		Operation: "set-tags",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Message:   "Blob index tags updated.",
		Details:   details,
		Applied:   map[string]any{"tags": tags},
	})
}

func deleteBlob(ctx context.Context, scope *clientScope, cfg config) error {
	if err := requireBlobName(cfg); err != nil {
		return err
	}

	if cfg.DryRun {
		return writeJSON(previewOutput("delete", scope, cfg, map[string]any{
			"would_delete_snapshots": true,
			"required_confirm":       "DELETE",
		}))
	}
	if cfg.Confirm != "DELETE" {
		return fmt.Errorf("delete requires --confirm DELETE")
	}

	blobClient := scope.Client.NewBlobClient(cfg.BlobName)
	if _, err := blobClient.Delete(ctx, &blob.DeleteOptions{
		DeleteSnapshots: to.Ptr(blob.DeleteSnapshotsOptionTypeInclude),
	}); err != nil {
		return fmt.Errorf("delete blob: %w", err)
	}

	return writeJSON(commandOutput{
		Operation: "delete",
		Status:    "ok",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		Message:   "Blob deleted with snapshots included.",
		Applied:   map[string]any{"deleted": true, "snapshots": "include"},
	})
}

func previewOutput(operation string, scope *clientScope, cfg config, applied map[string]any) commandOutput {
	return commandOutput{
		Operation: operation,
		Status:    "preview",
		AuthMode:  scope.AuthMode,
		Container: scope.Container,
		Blob:      cfg.BlobName,
		DryRun:    true,
		Message:   "No Azure Blob changes were made.",
		Applied:   applied,
	}
}

func readBlobDetails(ctx context.Context, blobClient *blob.Client, blobName string, maxBytes int64) (*blobDetails, error) {
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get blob properties: %w", err)
	}

	details := &blobDetails{
		Name:         blobName,
		SizeBytes:    valueInt64(props.ContentLength),
		ContentType:  valueString(props.ContentType),
		LastModified: valueTime(props.LastModified),
		ETag:         valueETag(props.ETag),
		AccessTier:   valueString(props.AccessTier),
		Metadata:     metadataStrings(props.Metadata),
	}
	if props.BlobType != nil {
		details.BlobType = string(*props.BlobType)
	}

	tags, err := blobClient.GetTags(ctx, nil)
	if err != nil {
		details.TagsError = err.Error()
	} else {
		details.Tags = blobTagsToMap(tags.BlobTagSet)
	}

	if maxBytes > 0 {
		preview, err := readPreview(ctx, blobClient, maxBytes)
		if err != nil {
			details.PreviewError = err.Error()
		} else {
			details.Preview = preview
		}
	}

	return details, nil
}

func readPreview(ctx context.Context, blobClient *blob.Client, maxBytes int64) (*contentPreview, error) {
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

	preview := &contentPreview{
		Bytes:     len(data),
		Truncated: truncated,
	}
	if utf8.Valid(data) {
		preview.Text = string(data)
	} else {
		preview.Base64 = base64.StdEncoding.EncodeToString(data)
	}
	return preview, nil
}

func readPayload(cfg config) (*payload, error) {
	if strings.TrimSpace(cfg.File) != "" && cfg.Content != "" {
		return nil, fmt.Errorf("use either --file or --content, not both")
	}

	switch {
	case strings.TrimSpace(cfg.File) != "":
		data, err := os.ReadFile(cfg.File)
		if err != nil {
			return nil, fmt.Errorf("read --file: %w", err)
		}
		contentType := strings.TrimSpace(cfg.ContentType)
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(cfg.File))
		}
		return &payload{
			Bytes:       data,
			Source:      cleanOutputPath(cfg.File),
			ContentType: contentType,
		}, nil

	case cfg.Content != "":
		contentType := strings.TrimSpace(cfg.ContentType)
		if contentType == "" {
			contentType = "text/plain; charset=utf-8"
		}
		return &payload{
			Bytes:       []byte(cfg.Content),
			Source:      "inline",
			ContentType: contentType,
		}, nil

	default:
		contentType := strings.TrimSpace(cfg.ContentType)
		if contentType == "" {
			contentType = "application/json"
		}
		body := map[string]string{
			"tool":       "azureblob-test-tool",
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"purpose":    "Everest Azure Blob connector validation",
		}
		data, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			return nil, err
		}
		return &payload{
			Bytes:       append(data, '\n'),
			Source:      "generated",
			ContentType: contentType,
		}, nil
	}
}

func blobExists(ctx context.Context, blobClient *blob.Client) (bool, error) {
	if _, err := blobClient.GetProperties(ctx, nil); err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("check blob existence: %w", err)
	}
	return true, nil
}

func summarizeContainerItem(item *service.ContainerItem) containerSummary {
	summary := containerSummary{
		Name:     valueString(item.Name),
		Metadata: metadataStrings(item.Metadata),
	}
	if item.Properties == nil {
		return summary
	}

	props := item.Properties
	summary.LastModified = valueTime(props.LastModified)
	summary.ETag = valueETag(props.ETag)
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

func summarizeBlobItem(item *container.BlobItem) blobSummary {
	summary := blobSummary{Name: valueString(item.Name)}
	if item.Properties == nil {
		return summary
	}
	props := item.Properties
	summary.SizeBytes = valueInt64(props.ContentLength)
	summary.ContentType = valueString(props.ContentType)
	summary.LastModified = valueTime(props.LastModified)
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

func requireBlobName(cfg config) error {
	if cfg.BlobName == "" {
		return fmt.Errorf("--blob is required for --op %s", cfg.Operation)
	}
	return nil
}

func serviceContainerURL(rawURL string, containerName string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse --service-url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("--service-url must be an absolute URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + url.PathEscape(containerName)
	return parsed.String(), nil
}

func containerNameFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func parseKVList(raw string) (map[string]string, error) {
	out := map[string]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, nil
	}

	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("%q must be key=value", item)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("%q has an empty key", item)
		}
		out[key] = value
	}
	return out, nil
}

func toMetadataPointers(metadata map[string]string) map[string]*string {
	out := make(map[string]*string, len(metadata))
	for key, value := range metadata {
		valueCopy := value
		out[key] = &valueCopy
	}
	return out
}

func metadataStrings(metadata map[string]*string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = valueString(value)
	}
	return sortedMap(out)
}

func blobTagsToMap(tags []*blob.Tags) map[string]string {
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		if tag == nil || tag.Key == nil {
			continue
		}
		out[*tag.Key] = valueString(tag.Value)
	}
	return sortedMap(out)
}

func sortedMap(input map[string]string) map[string]string {
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

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return to.Ptr(value)
}

func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func valueTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func valueETag(value *azcore.ETag) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func cleanOutputPath(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	if absolute, err := filepath.Abs(cleaned); err == nil {
		return absolute
	}
	return cleaned
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
