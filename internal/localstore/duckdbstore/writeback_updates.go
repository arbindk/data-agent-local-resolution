package duckdbstore

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) UpdateManifestActionStatus(
	ctx context.Context,
	executionID string,
	fileID string,
	status string,
	errText string,
) error {
	if executionID == "" {
		return fmt.Errorf("execution_id is required")
	}
	if fileID == "" {
		return fmt.Errorf("file_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE manifest_details
		SET action_status = ?, error = ?
		WHERE execution_id = ?
		  AND file_id = ?
	`, status, errText, executionID, fileID)
	if err != nil {
		return fmt.Errorf("update manifest action status: %w", err)
	}

	return nil
}

func (s *Store) InsertWritebackAction(
	ctx context.Context,
	executionID string,
	actionID string,
	decisionID string,
	fileID string,
	actionType string,
	status string,
	errText string,
) error {
	if executionID == "" {
		return fmt.Errorf("execution_id is required")
	}
	if actionID == "" {
		actionID = "action-" + time.Now().UTC().Format("20060102-150405.000000000")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO writeback_actions (
			execution_id,
			action_id,
			decision_id,
			file_id,
			action_type,
			status,
			error,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, executionID, actionID, decisionID, fileID, actionType, status, errText, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert writeback action: %w", err)
	}

	return nil
}
