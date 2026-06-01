package contracts

import "time"

type NormalizedBatch struct {
	BatchID     string                 `json:"batch_id"`
	LeaseID     string                 `json:"lease_id"`
	DataStoreID string                 `json:"data_store_id"`
	Records     []NormalizedFileRecord `json:"records"`
}

type NormalizedFileRecord struct {
	FileID              string    `json:"file_id"`
	DataStoreID         string    `json:"data_store_id"`
	Path                string    `json:"path"`
	Name                string    `json:"name"`
	SizeBytes           int64     `json:"size_bytes"`
	LastModified        string    `json:"last_modified"`
	LastAccessed        string    `json:"last_accessed"`
	ContentType         string    `json:"content_type"`
	Extension           string    `json:"extension"`
	SourceSystem        string    `json:"source_system"`
	PermissionClass     string    `json:"permission_class"`
	PermissionRiskScore int       `json:"permission_risk_score"`
	DDProtectionFlags   int       `json:"dd_protection_flags"`
	ScanTimestamp       time.Time `json:"scan_timestamp"`
}
