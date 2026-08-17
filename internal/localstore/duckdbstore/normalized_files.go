package duckdbstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

func (s *Store) ReplaceNormalizedFiles(ctx context.Context, batch contracts.NormalizedBatch) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace normalized files: %w", err)
	}
	defer rollbackIfNeeded(tx)

	if _, err := tx.ExecContext(ctx, `DELETE FROM normalized_files WHERE batch_id = ?`, batch.BatchID); err != nil {
		return fmt.Errorf("delete existing normalized files: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO normalized_files (
			batch_id,
			file_id,
			data_store_id,
			path,
			name,
			size_bytes,
			last_modified,
			last_accessed,
			content_type,
			extension,
			source_system,
			permission_class,
			permission_risk_score,
			dd_protection_flags,
			scan_timestamp
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert normalized file: %w", err)
	}
	defer stmt.Close()

	for _, record := range batch.Records {
		scanTS := record.ScanTimestamp
		if scanTS.IsZero() {
			scanTS = time.Now().UTC()
		}

		if _, err := stmt.ExecContext(
			ctx,
			batch.BatchID,
			record.FileID,
			record.DataStoreID,
			record.Path,
			record.Name,
			record.SizeBytes,
			record.LastModified,
			record.LastAccessed,
			record.ContentType,
			record.Extension,
			record.SourceSystem,
			record.PermissionClass,
			record.PermissionRiskScore,
			record.DDProtectionFlags,
			scanTS.Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("insert normalized file %s: %w", record.FileID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit normalized files: %w", err)
	}

	return nil
}

func (s *Store) LoadNormalizedBatch(ctx context.Context, batchID string) (*contracts.NormalizedBatch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			batch_id,
			file_id,
			data_store_id,
			path,
			name,
			size_bytes,
			last_modified,
			last_accessed,
			content_type,
			extension,
			source_system,
			permission_class,
			permission_risk_score,
			dd_protection_flags,
			scan_timestamp
		FROM normalized_files
		WHERE batch_id = ?
		ORDER BY path
	`, batchID)
	if err != nil {
		return nil, fmt.Errorf("query normalized files: %w", err)
	}
	defer rows.Close()

	batch := &contracts.NormalizedBatch{
		BatchID: batchID,
		Records: make([]contracts.NormalizedFileRecord, 0),
	}

	for rows.Next() {
		var record contracts.NormalizedFileRecord
		var rowBatchID string
		var scanTimestamp string

		if err := rows.Scan(
			&rowBatchID,
			&record.FileID,
			&record.DataStoreID,
			&record.Path,
			&record.Name,
			&record.SizeBytes,
			&record.LastModified,
			&record.LastAccessed,
			&record.ContentType,
			&record.Extension,
			&record.SourceSystem,
			&record.PermissionClass,
			&record.PermissionRiskScore,
			&record.DDProtectionFlags,
			&scanTimestamp,
		); err != nil {
			return nil, fmt.Errorf("scan normalized file row: %w", err)
		}

		record.ScanTimestamp = parseTimeOrZero(scanTimestamp)

		if batch.DataStoreID == "" {
			batch.DataStoreID = record.DataStoreID
		}
		if batch.LeaseID == "" {
			batch.LeaseID = ""
		}

		batch.Records = append(batch.Records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate normalized files: %w", err)
	}

	return batch, nil
}

func (s *Store) CountNormalizedFiles(ctx context.Context, batchID string) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM normalized_files WHERE batch_id = ?`, batchID)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count normalized files: %w", err)
	}

	return count, nil
}

func rollbackIfNeeded(tx *sql.Tx) {
	_ = tx.Rollback()
}

func parseTimeOrZero(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return t
}
