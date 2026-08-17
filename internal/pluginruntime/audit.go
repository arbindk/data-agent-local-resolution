package pluginruntime

import (
	"sync"
	"time"
)

type AuditRecord struct {
	ID              string   `json:"id"`
	RuntimeType     string   `json:"runtime_type"`
	SubjectID       string   `json:"subject_id"`
	Operation       string   `json:"operation"`
	Status          string   `json:"status"`
	RuntimeKind     string   `json:"runtime_kind,omitempty"`
	ContractVersion string   `json:"contract_version,omitempty"`
	DryRun          bool     `json:"dry_run,omitempty"`
	SafeMode        bool     `json:"safe_mode,omitempty"`
	DurationMillis  int64    `json:"duration_ms"`
	Error           string   `json:"error,omitempty"`
	Controls        []string `json:"controls,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type Store struct {
	mu      sync.RWMutex
	records []AuditRecord
	limit   int
}

func NewStore(limit int) *Store {
	if limit <= 0 {
		limit = 500
	}
	return &Store{limit: limit}
}

func (s *Store) Add(record AuditRecord) AuditRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ID == "" {
		record.ID = "runtime-audit-" + time.Now().UTC().Format("20060102-150405.000000000")
	}
	if record.CreatedAt == "" {
		record.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	s.records = append([]AuditRecord{record}, s.records...)
	if len(s.records) > s.limit {
		s.records = s.records[:s.limit]
	}
	return record
}

func (s *Store) List(runtimeType string, limit int) []AuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > s.limit {
		limit = 100
	}

	items := make([]AuditRecord, 0, min(limit, len(s.records)))
	for _, record := range s.records {
		if runtimeType != "" && record.RuntimeType != runtimeType {
			continue
		}
		items = append(items, record)
		if len(items) >= limit {
			break
		}
	}
	return items
}
