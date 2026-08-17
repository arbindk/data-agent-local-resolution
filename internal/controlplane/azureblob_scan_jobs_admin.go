// COPY THIS FILE TO: internal/controlplane/azureblob_scan_jobs_admin.go
package controlplane

// ResetAll clears every checkpointed scan job and persists the empty store.
func (s *AzureBlobScanJobStore) ResetAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = map[string]AzureBlobScanJob{}
	s.saveLocked()
}

// ImportJobs replaces the current jobs with the supplied set (used by
// POST /v1/admin/import to restore a backup).
func (s *AzureBlobScanJobStore) ImportJobs(jobs []AzureBlobScanJob) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jobs == nil {
		s.jobs = map[string]AzureBlobScanJob{}
	}
	for _, job := range jobs {
		if job.JobID == "" {
			continue
		}
		s.jobs[job.JobID] = job
	}
	s.saveLocked()
}
