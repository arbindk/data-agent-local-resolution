// COPY THIS FILE TO: internal/portalstate/admin.go
package portalstate

// Reset clears all runtime portal state (sources, containers, blobs, evidence,
// audit, actions, jobs, workbench history) and persists the empty state.
// Policies (config defaults) are preserved so the cell still resolves policy.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	policies := s.state.Policies
	s.state = State{
		Version:   Version,
		UpdatedAt: now(),
		Policies:  policies,
	}
	s.saveLocked()
}
