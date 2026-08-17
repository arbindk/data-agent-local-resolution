package searchindex

import (
	"context"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type Config struct {
	Mode           string
	Endpoint       string
	IndexName      string
	APIKey         string
	Username       string
	Password       string
	TimeoutSeconds int
}

type ConfiguredIndex struct {
	cfg     Config
	local   *LocalIndex
	elastic *ElasticIndex
}

func FromAppConfig(cfg config.Config) Config {
	return Config{
		Mode:           cfg.SearchIndexMode,
		Endpoint:       cfg.SearchIndexEndpoint,
		IndexName:      cfg.SearchIndexName,
		APIKey:         cfg.SearchIndexAPIKey,
		Username:       cfg.SearchIndexUsername,
		Password:       cfg.SearchIndexPassword,
		TimeoutSeconds: cfg.SearchIndexTimeoutSeconds,
	}
}

func NewConfiguredIndex(cfg Config, portalState *portalstate.Store) *ConfiguredIndex {
	cfg.Mode = normalizeMode(cfg.Mode)
	if strings.TrimSpace(cfg.IndexName) == "" {
		cfg.IndexName = "everest-summary"
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 5
	}

	out := &ConfiguredIndex{
		cfg:   cfg,
		local: NewLocalIndex(portalState, cfg.IndexName),
	}
	if cfg.Mode == contracts.SearchIndexModeElastic {
		out.elastic = NewElasticIndex(cfg)
	}
	return out
}

func (i *ConfiguredIndex) Publish(ctx context.Context, batch contracts.SearchIndexBatch) (*contracts.SearchIndexReceipt, error) {
	if i == nil {
		return nil, nil
	}
	mode := normalizeMode(i.cfg.Mode)
	now := time.Now().UTC()
	batch = batch.WithDefaults(now)
	for idx := range batch.Documents {
		if strings.TrimSpace(batch.Documents[idx].IndexName) == "" {
			batch.Documents[idx].IndexName = i.cfg.IndexName
		}
	}

	switch mode {
	case contracts.SearchIndexModeDisabled:
		return &contracts.SearchIndexReceipt{
			Status:          contracts.SearchIndexStatusDisabled,
			Mode:            mode,
			Backend:         contracts.SearchIndexModeDisabled,
			IndexName:       i.cfg.IndexName,
			DocumentsQueued: len(batch.Documents),
			PublishedAt:     now,
		}, nil

	case contracts.SearchIndexModeElastic:
		localReceipt, _ := i.local.Publish(ctx, batch)
		receipt, err := i.elastic.Publish(ctx, batch)
		if err != nil {
			queued := len(batch.Documents)
			if localReceipt != nil {
				queued = localReceipt.DocumentsPublished
			}
			return &contracts.SearchIndexReceipt{
				Status:          contracts.SearchIndexStatusQueued,
				Mode:            mode,
				Backend:         "local_mirror_elastic",
				IndexName:       i.cfg.IndexName,
				DocumentsQueued: queued,
				Error:           err.Error(),
				PublishedAt:     now,
			}, err
		}
		return receipt, nil

	default:
		return i.local.Publish(ctx, batch)
	}
}

func (i *ConfiguredIndex) Status() map[string]any {
	if i == nil {
		return map[string]any{"mode": contracts.SearchIndexModeDisabled, "status": "not_configured"}
	}
	mode := normalizeMode(i.cfg.Mode)
	return map[string]any{
		"mode":                mode,
		"status":              "available",
		"backend":             backendName(mode),
		"index_name":          i.cfg.IndexName,
		"endpoint_configured": strings.TrimSpace(i.cfg.Endpoint) != "",
		"local_mirror":        i.local != nil,
		"external_enabled":    mode == contracts.SearchIndexModeElastic,
	}
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case contracts.SearchIndexModeDisabled, "off", "none":
		return contracts.SearchIndexModeDisabled
	case contracts.SearchIndexModeElastic, "elasticsearch", "opensearch":
		return contracts.SearchIndexModeElastic
	default:
		return contracts.SearchIndexModeLocal
	}
}

func backendName(mode string) string {
	switch normalizeMode(mode) {
	case contracts.SearchIndexModeElastic:
		return "local_mirror_plus_elastic_bulk"
	case contracts.SearchIndexModeDisabled:
		return contracts.SearchIndexModeDisabled
	default:
		return "portal_state_local_index"
	}
}
