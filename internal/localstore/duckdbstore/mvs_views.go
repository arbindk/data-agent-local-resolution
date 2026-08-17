package duckdbstore

import (
	"context"
	"database/sql"
	"fmt"
)

type TableInfo struct {
	Name        string `json:"name"`
	RecordType  string `json:"record_type"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type NormalizedFileView struct {
	BatchID             string `json:"batch_id"`
	FileID              string `json:"file_id"`
	DataStoreID         string `json:"data_store_id"`
	Path                string `json:"path"`
	Name                string `json:"name"`
	SizeBytes           int64  `json:"size_bytes"`
	ContentType         string `json:"content_type"`
	Extension           string `json:"extension"`
	SourceSystem        string `json:"source_system"`
	PermissionClass     string `json:"permission_class"`
	PermissionRiskScore int    `json:"permission_risk_score"`
	DDProtectionFlags   int    `json:"dd_protection_flags"`
	ScanTimestamp       string `json:"scan_timestamp"`
}

type ActionPlanView struct {
	ExecutionID      string `json:"execution_id"`
	FileID           string `json:"file_id"`
	Path             string `json:"path"`
	RiskLevel        string `json:"risk_level"`
	Classification   string `json:"classification"`
	ActionRequested  string `json:"action_requested"`
	ActionStatus     string `json:"action_status"`
	GuardianDecision string `json:"guardian_decision"`
	ProvenanceID     string `json:"provenance_id"`
}

type GuardianDecisionView struct {
	ExecutionID    string `json:"execution_id"`
	DecisionID     string `json:"decision_id"`
	FileID         string `json:"file_id"`
	Path           string `json:"path"`
	RiskLevel      string `json:"risk_level"`
	Classification string `json:"classification"`
	ProposedAction string `json:"proposed_action"`
	Decision       string `json:"decision"`
	Reasoning      string `json:"reasoning"`
}

func (s *Store) ListTables(ctx context.Context) ([]TableInfo, error) {
	tables := []TableInfo{
		{
			Name:        "normalized_files",
			RecordType:  "batch working data",
			Description: "One normalized metadata record per scanned object/file.",
		},
		{
			Name:        "content_findings",
			RecordType:  "intelligence output",
			Description: "Entity/classification findings from content intelligence.",
		},
		{
			Name:        "guardian_decisions",
			RecordType:  "governance decision",
			Description: "Guardian allow/deny/review decisions before write-back.",
		},
		{
			Name:        "writeback_actions",
			RecordType:  "action execution",
			Description: "Dry-run/apply action status records.",
		},
		{
			Name:        "manifest_summary",
			RecordType:  "customer evidence",
			Description: "One summary row per execution.",
		},
		{
			Name:        "manifest_details",
			RecordType:  "customer evidence",
			Description: "Per-object manifest details.",
		},
	}

	for i := range tables {
		count, err := s.countTable(ctx, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Count = count
	}

	return tables, nil
}

func (s *Store) countTable(ctx context.Context, table string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	var count int
	if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count table %s: %w", table, err)
	}

	return count, nil
}

func (s *Store) ListNormalizedFiles(ctx context.Context, batchID string, limit int, offset int) ([]NormalizedFileView, int, error) {
	if batchID == "" {
		return nil, 0, fmt.Errorf("batch_id is required")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM normalized_files
		WHERE batch_id = ?
	`, batchID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count normalized files: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			batch_id,
			file_id,
			data_store_id,
			path,
			name,
			size_bytes,
			content_type,
			extension,
			source_system,
			permission_class,
			permission_risk_score,
			dd_protection_flags,
			scan_timestamp
		FROM normalized_files
		WHERE batch_id = ?
		ORDER BY permission_risk_score DESC, path
		LIMIT ? OFFSET ?
	`, batchID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query normalized files: %w", err)
	}
	defer rows.Close()

	items := make([]NormalizedFileView, 0)

	for rows.Next() {
		var item NormalizedFileView
		if err := rows.Scan(
			&item.BatchID,
			&item.FileID,
			&item.DataStoreID,
			&item.Path,
			&item.Name,
			&item.SizeBytes,
			&item.ContentType,
			&item.Extension,
			&item.SourceSystem,
			&item.PermissionClass,
			&item.PermissionRiskScore,
			&item.DDProtectionFlags,
			&item.ScanTimestamp,
		); err != nil {
			return nil, 0, fmt.Errorf("scan normalized file view: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate normalized file view: %w", err)
	}

	return items, total, nil
}

func (s *Store) QuickScanAnalytics(ctx context.Context, batchID string) (map[string]any, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	var totalObjects int
	var totalSize int64
	var avgRisk sql.NullFloat64

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(size_bytes), 0), AVG(permission_risk_score)
		FROM normalized_files
		WHERE batch_id = ?
	`, batchID).Scan(&totalObjects, &totalSize, &avgRisk); err != nil {
		return nil, fmt.Errorf("quick analytics summary: %w", err)
	}

	extensionCounts, err := s.groupCount(ctx, `
		SELECT COALESCE(NULLIF(extension, ''), 'unknown') AS k, COUNT(*)
		FROM normalized_files
		WHERE batch_id = ?
		GROUP BY COALESCE(NULLIF(extension, ''), 'unknown')
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`, batchID)
	if err != nil {
		return nil, err
	}

	riskCounts, err := s.groupCount(ctx, `
		SELECT
			CASE
				WHEN permission_risk_score >= 90 THEN 'critical'
				WHEN permission_risk_score >= 70 THEN 'high'
				WHEN permission_risk_score >= 40 THEN 'medium'
				ELSE 'low'
			END AS k,
			COUNT(*)
		FROM normalized_files
		WHERE batch_id = ?
		GROUP BY k
	`, batchID)
	if err != nil {
		return nil, err
	}

	permissionCounts, err := s.groupCount(ctx, `
		SELECT COALESCE(NULLIF(permission_class, ''), 'unknown') AS k, COUNT(*)
		FROM normalized_files
		WHERE batch_id = ?
		GROUP BY COALESCE(NULLIF(permission_class, ''), 'unknown')
		ORDER BY COUNT(*) DESC
	`, batchID)
	if err != nil {
		return nil, err
	}

	avg := 0.0
	if avgRisk.Valid {
		avg = avgRisk.Float64
	}

	return map[string]any{
		"status":             "ok",
		"batch_id":           batchID,
		"total_objects":      totalObjects,
		"total_size_bytes":   totalSize,
		"average_risk_score": avg,
		"by_extension":       extensionCounts,
		"by_risk_level":      riskCounts,
		"by_permission":      permissionCounts,
		"store":              "local_execution_store_ephemeral",
	}, nil
}

func (s *Store) groupCount(ctx context.Context, query string, arg any) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("group count query: %w", err)
	}
	defer rows.Close()

	result := map[string]int{}

	for rows.Next() {
		var key string
		var count int

		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan group count: %w", err)
		}

		result[key] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group count: %w", err)
	}

	return result, nil
}

func (s *Store) ListActionPlan(ctx context.Context, executionID string, limit int, offset int) ([]ActionPlanView, int, error) {
	if executionID == "" {
		return nil, 0, fmt.Errorf("execution_id is required")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM manifest_details
		WHERE execution_id = ?
		  AND action_requested <> 'none'
	`, executionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count action plan: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			execution_id,
			file_id,
			path,
			risk_level,
			classification,
			action_requested,
			action_status,
			guardian_decision,
			provenance_id
		FROM manifest_details
		WHERE execution_id = ?
		  AND action_requested <> 'none'
		ORDER BY
			CASE risk_level
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				ELSE 4
			END,
			path
		LIMIT ? OFFSET ?
	`, executionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query action plan: %w", err)
	}
	defer rows.Close()

	items := make([]ActionPlanView, 0)

	for rows.Next() {
		var item ActionPlanView
		if err := rows.Scan(
			&item.ExecutionID,
			&item.FileID,
			&item.Path,
			&item.RiskLevel,
			&item.Classification,
			&item.ActionRequested,
			&item.ActionStatus,
			&item.GuardianDecision,
			&item.ProvenanceID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan action plan: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate action plan: %w", err)
	}

	return items, total, nil
}

func (s *Store) ListGuardianDecisions(ctx context.Context, executionID string, limit int, offset int) ([]GuardianDecisionView, int, error) {
	if executionID == "" {
		return nil, 0, fmt.Errorf("execution_id is required")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM manifest_details
		WHERE execution_id = ?
		  AND action_requested <> 'none'
	`, executionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count guardian decisions: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			execution_id,
			COALESCE(NULLIF(provenance_id, ''), 'decision-' || rowid) AS decision_id,
			file_id,
			path,
			risk_level,
			classification,
			action_requested,
			guardian_decision
		FROM manifest_details
		WHERE execution_id = ?
		  AND action_requested <> 'none'
		ORDER BY path
		LIMIT ? OFFSET ?
	`, executionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query guardian decisions: %w", err)
	}
	defer rows.Close()

	items := make([]GuardianDecisionView, 0)

	for rows.Next() {
		var item GuardianDecisionView
		if err := rows.Scan(
			&item.ExecutionID,
			&item.DecisionID,
			&item.FileID,
			&item.Path,
			&item.RiskLevel,
			&item.Classification,
			&item.ProposedAction,
			&item.Decision,
		); err != nil {
			return nil, 0, fmt.Errorf("scan guardian decision: %w", err)
		}

		item.Reasoning = "Guardian gate is required before real write-back. Data Agent applies actions only when the decision is allow and write-back is enabled."

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate guardian decisions: %w", err)
	}

	return items, total, nil
}
