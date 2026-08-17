// COPY THIS FILE TO: internal/localstore/duckdbstore/admin.go
package duckdbstore

import "context"

// ResetRuntime clears all runtime rows from the working store while preserving
// the schema. Called by POST /v1/admin/reset (scope "duckdb").
func (s *Store) ResetRuntime(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	tables := []string{
		"normalized_files",
		"content_findings",
		"guardian_decisions",
		"writeback_actions",
		"manifest_details",
		"manifest_summary",
	}
	for _, t := range tables {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+t); err != nil {
			return err
		}
	}
	return nil
}
