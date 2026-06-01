package duckdbstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

func (s *Store) SaveManifest(ctx context.Context, summary contracts.ManifestSummary, details []contracts.ManifestDetail) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save manifest: %w", err)
	}
	defer rollbackIfNeeded(tx)

	if _, err := tx.ExecContext(ctx, `DELETE FROM manifest_summary WHERE execution_id = ?`, summary.ExecutionID); err != nil {
		return fmt.Errorf("delete existing manifest summary: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM manifest_details WHERE execution_id = ?`, summary.ExecutionID); err != nil {
		return fmt.Errorf("delete existing manifest details: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO manifest_summary (
			execution_id,
			job_id,
			lease_id,
			batch_id,
			data_store_id,
			source_system,
			status,
			total_records,
			processed_records,
			total_size_bytes,
			critical_count,
			high_count,
			medium_count,
			low_count,
			average_risk_score,
			actions_planned,
			actions_completed,
			actions_failed,
			compiled_policy_version,
			workflow_version,
			processing_tier,
			processing_mode,
			started_at,
			completed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		summary.ExecutionID,
		summary.JobID,
		summary.LeaseID,
		summary.BatchID,
		summary.DataStoreID,
		summary.SourceSystem,
		summary.Status,
		summary.TotalRecords,
		summary.ProcessedRecords,
		summary.TotalSizeBytes,
		summary.CriticalCount,
		summary.HighCount,
		summary.MediumCount,
		summary.LowCount,
		summary.AverageRiskScore,
		summary.ActionsPlanned,
		summary.ActionsCompleted,
		summary.ActionsFailed,
		summary.CompiledPolicyVersion,
		summary.WorkflowVersion,
		summary.ProcessingTier,
		summary.ProcessingMode,
		summary.StartedAt,
		summary.CompletedAt,
	); err != nil {
		return fmt.Errorf("insert manifest summary: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO manifest_details (
			execution_id,
			batch_id,
			data_store_id,
			file_id,
			path,
			name,
			size_bytes,
			content_type,
			extension,
			source_system,
			risk_score,
			risk_level,
			classification,
			entities_json,
			guardian_decision,
			action_requested,
			action_status,
			error,
			provenance_id,
			compiled_policy_version,
			workflow_version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert manifest details: %w", err)
	}
	defer stmt.Close()

	for _, detail := range details {
		entitiesJSON, err := json.Marshal(detail.Entities)
		if err != nil {
			return fmt.Errorf("marshal manifest entities for file %s: %w", detail.FileID, err)
		}

		if _, err := stmt.ExecContext(ctx,
			detail.ExecutionID,
			detail.BatchID,
			detail.DataStoreID,
			detail.FileID,
			detail.Path,
			detail.Name,
			detail.SizeBytes,
			detail.ContentType,
			detail.Extension,
			detail.SourceSystem,
			detail.RiskScore,
			detail.RiskLevel,
			detail.Classification,
			string(entitiesJSON),
			detail.GuardianDecision,
			detail.ActionRequested,
			detail.ActionStatus,
			detail.Error,
			detail.ProvenanceID,
			detail.CompiledPolicyVer,
			detail.WorkflowVersion,
		); err != nil {
			return fmt.Errorf("insert manifest detail %s: %w", detail.FileID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit manifest: %w", err)
	}

	return nil
}

func (s *Store) GetManifestSummary(ctx context.Context, executionID string) (*contracts.ManifestSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			execution_id,
			job_id,
			lease_id,
			batch_id,
			data_store_id,
			source_system,
			status,
			total_records,
			processed_records,
			total_size_bytes,
			critical_count,
			high_count,
			medium_count,
			low_count,
			average_risk_score,
			actions_planned,
			actions_completed,
			actions_failed,
			compiled_policy_version,
			workflow_version,
			processing_tier,
			processing_mode,
			started_at,
			completed_at
		FROM manifest_summary
		WHERE execution_id = ?
	`, executionID)

	var summary contracts.ManifestSummary

	if err := row.Scan(
		&summary.ExecutionID,
		&summary.JobID,
		&summary.LeaseID,
		&summary.BatchID,
		&summary.DataStoreID,
		&summary.SourceSystem,
		&summary.Status,
		&summary.TotalRecords,
		&summary.ProcessedRecords,
		&summary.TotalSizeBytes,
		&summary.CriticalCount,
		&summary.HighCount,
		&summary.MediumCount,
		&summary.LowCount,
		&summary.AverageRiskScore,
		&summary.ActionsPlanned,
		&summary.ActionsCompleted,
		&summary.ActionsFailed,
		&summary.CompiledPolicyVersion,
		&summary.WorkflowVersion,
		&summary.ProcessingTier,
		&summary.ProcessingMode,
		&summary.StartedAt,
		&summary.CompletedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("manifest summary not found for execution_id=%s", executionID)
		}
		return nil, fmt.Errorf("scan manifest summary: %w", err)
	}

	return &summary, nil
}

func (s *Store) ListManifestDetails(ctx context.Context, executionID string, limit int, offset int) ([]contracts.ManifestDetail, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	totalRow := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM manifest_details WHERE execution_id = ?`, executionID)

	var total int
	if err := totalRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manifest details: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			execution_id,
			batch_id,
			data_store_id,
			file_id,
			path,
			name,
			size_bytes,
			content_type,
			extension,
			source_system,
			risk_score,
			risk_level,
			classification,
			entities_json,
			guardian_decision,
			action_requested,
			action_status,
			error,
			provenance_id,
			compiled_policy_version,
			workflow_version
		FROM manifest_details
		WHERE execution_id = ?
		ORDER BY risk_score DESC, path ASC
		LIMIT ? OFFSET ?
	`, executionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query manifest details: %w", err)
	}
	defer rows.Close()

	details := make([]contracts.ManifestDetail, 0)

	for rows.Next() {
		var detail contracts.ManifestDetail
		var entitiesJSON string

		if err := rows.Scan(
			&detail.ExecutionID,
			&detail.BatchID,
			&detail.DataStoreID,
			&detail.FileID,
			&detail.Path,
			&detail.Name,
			&detail.SizeBytes,
			&detail.ContentType,
			&detail.Extension,
			&detail.SourceSystem,
			&detail.RiskScore,
			&detail.RiskLevel,
			&detail.Classification,
			&entitiesJSON,
			&detail.GuardianDecision,
			&detail.ActionRequested,
			&detail.ActionStatus,
			&detail.Error,
			&detail.ProvenanceID,
			&detail.CompiledPolicyVer,
			&detail.WorkflowVersion,
		); err != nil {
			return nil, 0, fmt.Errorf("scan manifest detail: %w", err)
		}

		if entitiesJSON != "" {
			_ = json.Unmarshal([]byte(entitiesJSON), &detail.Entities)
		}

		details = append(details, detail)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manifest details: %w", err)
	}

	return details, total, nil
}
