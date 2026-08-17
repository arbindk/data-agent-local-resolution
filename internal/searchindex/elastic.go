package searchindex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type ElasticIndex struct {
	cfg    Config
	client *http.Client
}

func NewElasticIndex(cfg Config) *ElasticIndex {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &ElasticIndex{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (i *ElasticIndex) Publish(ctx context.Context, batch contracts.SearchIndexBatch) (*contracts.SearchIndexReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if i == nil {
		return nil, fmt.Errorf("elastic search index is not configured")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(i.cfg.Endpoint), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("elastic/opensearch endpoint is not configured")
	}
	indexName := strings.TrimSpace(i.cfg.IndexName)
	if indexName == "" {
		indexName = "everest-summary"
	}

	now := time.Now().UTC()
	batch = batch.WithDefaults(now)

	var body bytes.Buffer
	for _, doc := range batch.Documents {
		doc.IndexName = indexName
		action := map[string]any{
			"index": map[string]any{
				"_index": indexName,
				"_id":    doc.DocumentID,
			},
		}
		if err := writeBulkLine(&body, action); err != nil {
			return nil, err
		}
		if err := writeBulkLine(&body, doc); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/_bulk", &body)
	if err != nil {
		return nil, fmt.Errorf("build elastic bulk request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if strings.TrimSpace(i.cfg.APIKey) != "" {
		req.Header.Set("Authorization", "ApiKey "+strings.TrimSpace(i.cfg.APIKey))
	} else if strings.TrimSpace(i.cfg.Username) != "" || strings.TrimSpace(i.cfg.Password) != "" {
		req.SetBasicAuth(i.cfg.Username, i.cfg.Password)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("publish elastic bulk index: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("elastic bulk index returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Errors bool `json:"errors"`
	}
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &parsed)
	}
	if parsed.Errors {
		return nil, fmt.Errorf("elastic bulk index completed with item errors")
	}

	return &contracts.SearchIndexReceipt{
		Status:             contracts.SearchIndexStatusPublished,
		Mode:               contracts.SearchIndexModeElastic,
		Backend:            "elastic_bulk",
		IndexName:          indexName,
		DocumentsPublished: len(batch.Documents),
		PublishedAt:        now,
	}, nil
}

func writeBulkLine(body *bytes.Buffer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal elastic bulk line: %w", err)
	}
	body.Write(data)
	body.WriteByte('\n')
	return nil
}
