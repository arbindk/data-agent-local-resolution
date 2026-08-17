package portalstate

import (
	"sort"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type SearchDocumentFilter struct {
	Query       string
	Kind        string
	DataStoreID string
	ExecutionID string
	Limit       int
}

func (s *Store) UpsertSearchDocuments(docs []contracts.SearchIndexDocument) []contracts.SearchIndexDocument {
	if len(docs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	cleaned := make([]contracts.SearchIndexDocument, 0, len(docs))
	for _, doc := range docs {
		doc = doc.WithDefaults(now)
		if strings.TrimSpace(doc.DocumentID) == "" {
			continue
		}
		cleaned = append(cleaned, doc)
	}
	if len(cleaned) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byID := map[string]int{}
	for i, existing := range s.state.SearchDocuments {
		byID[existing.DocumentID] = i
	}

	for _, doc := range cleaned {
		if idx, ok := byID[doc.DocumentID]; ok {
			if doc.CreatedAt.IsZero() {
				doc.CreatedAt = s.state.SearchDocuments[idx].CreatedAt
			}
			s.state.SearchDocuments[idx] = doc
			continue
		}
		s.state.SearchDocuments = append(s.state.SearchDocuments, doc)
		byID[doc.DocumentID] = len(s.state.SearchDocuments) - 1
	}

	sortSearchDocumentsLocked(s.state.SearchDocuments)
	if len(s.state.SearchDocuments) > 5000 {
		s.state.SearchDocuments = s.state.SearchDocuments[:5000]
	}
	s.saveLocked()
	return cleaned
}

func (s *Store) ListSearchDocuments(filter SearchDocumentFilter) []contracts.SearchIndexDocument {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	kind := strings.ToLower(strings.TrimSpace(filter.Kind))
	dataStoreID := strings.ToLower(strings.TrimSpace(filter.DataStoreID))
	executionID := strings.ToLower(strings.TrimSpace(filter.ExecutionID))

	out := make([]contracts.SearchIndexDocument, 0, minSearchDocumentCount(limit, len(s.state.SearchDocuments)))
	for _, doc := range s.state.SearchDocuments {
		if kind != "" && kind != "all" && strings.ToLower(doc.Kind) != kind {
			continue
		}
		if dataStoreID != "" && strings.ToLower(doc.DataStoreID) != dataStoreID {
			continue
		}
		if executionID != "" && strings.ToLower(doc.ExecutionID) != executionID {
			continue
		}
		if query != "" && !searchDocumentMatches(doc, query) {
			continue
		}
		out = append(out, doc)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func sortSearchDocumentsLocked(items []contracts.SearchIndexDocument) {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].UpdatedAt
		right := items[j].UpdatedAt
		if left.Equal(right) {
			return items[i].DocumentID < items[j].DocumentID
		}
		return left.After(right)
	})
}

func searchDocumentMatches(doc contracts.SearchIndexDocument, query string) bool {
	parts := []string{
		doc.DocumentID,
		doc.Kind,
		doc.ExecutionID,
		doc.JobID,
		doc.LeaseID,
		doc.BatchID,
		doc.DataStoreID,
		doc.SourceSystem,
		doc.SubjectType,
		doc.SubjectID,
		doc.FileID,
		doc.Container,
		doc.Path,
		doc.Title,
		doc.Summary,
		doc.Status,
		doc.Severity,
		doc.Decision,
		doc.Action,
		doc.Classification,
		doc.PolicyBundleVersion,
		doc.PolicyBundleHash,
		doc.WorkflowVersion,
	}
	for _, value := range parts {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	for _, value := range doc.Entities {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	for _, value := range doc.Tags {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	for key, value := range doc.Metadata {
		if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func minSearchDocumentCount(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
