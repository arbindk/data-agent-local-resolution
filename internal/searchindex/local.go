package searchindex

import (
	"context"
	"fmt"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type LocalIndex struct {
	store     *portalstate.Store
	indexName string
}

func NewLocalIndex(store *portalstate.Store, indexName string) *LocalIndex {
	return &LocalIndex{
		store:     store,
		indexName: strings.TrimSpace(indexName),
	}
}

func (i *LocalIndex) Publish(ctx context.Context, batch contracts.SearchIndexBatch) (*contracts.SearchIndexReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if i == nil || i.store == nil {
		return nil, fmt.Errorf("local search index store is not configured")
	}

	now := time.Now().UTC()
	if i.indexName == "" {
		i.indexName = "everest-summary"
	}
	batch = batch.WithDefaults(now)
	for idx := range batch.Documents {
		if strings.TrimSpace(batch.Documents[idx].IndexName) == "" {
			batch.Documents[idx].IndexName = i.indexName
		}
	}
	published := i.store.UpsertSearchDocuments(batch.Documents)

	return &contracts.SearchIndexReceipt{
		Status:             contracts.SearchIndexStatusPublished,
		Mode:               contracts.SearchIndexModeLocal,
		Backend:            "portal_state_local_index",
		IndexName:          i.indexName,
		DocumentsPublished: len(published),
		PublishedAt:        now,
	}, nil
}
