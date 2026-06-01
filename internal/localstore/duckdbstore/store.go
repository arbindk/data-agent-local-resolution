package duckdbstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("local store path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create local store directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open local sqlite store: %w", err)
	}

	store := &Store{db: db}

	if err := store.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Init(ctx context.Context) error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS normalized_files (
			batch_id TEXT,
			file_id TEXT,
			data_store_id TEXT,
			path TEXT,
			name TEXT,
			size_bytes INTEGER,
			last_modified TEXT,
			last_accessed TEXT,
			content_type TEXT,
			extension TEXT,
			source_system TEXT,
			permission_class TEXT,
			permission_risk_score INTEGER,
			dd_protection_flags INTEGER,
			scan_timestamp TEXT
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS content_findings (
			execution_id TEXT,
			batch_id TEXT,
			file_id TEXT,
			entity_type TEXT,
			matched_value TEXT,
			confidence REAL,
			source TEXT,
			created_at TEXT
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS guardian_decisions (
			execution_id TEXT,
			decision_id TEXT,
			file_id TEXT,
			proposed_action TEXT,
			decision TEXT,
			reasoning TEXT,
			created_at TEXT
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS writeback_actions (
			execution_id TEXT,
			action_id TEXT,
			decision_id TEXT,
			file_id TEXT,
			action_type TEXT,
			status TEXT,
			error TEXT,
			created_at TEXT
		);
		`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("init local store schema: %w", err)
		}
	}

	return nil
}
