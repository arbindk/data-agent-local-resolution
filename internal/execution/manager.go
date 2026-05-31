package execution

import (
	"fmt"
	"sync"
)

type Manager struct {
	mu         sync.RWMutex
	statuses   map[string]ExecutionStatus
	summaries  map[string]any
	provenance map[string][]any
}

func NewManager() *Manager {
	return &Manager{
		statuses:   make(map[string]ExecutionStatus),
		summaries:  make(map[string]any),
		provenance: make(map[string][]any),
	}
}

func (m *Manager) SaveStatus(status ExecutionStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statuses[status.ExecutionID] = status
}

func (m *Manager) GetStatus(executionID string) (ExecutionStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.statuses[executionID]
	return v, ok
}

func (m *Manager) SaveSummary(executionID string, summary any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.summaries[executionID] = summary
}

func (m *Manager) GetSummary(executionID string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.summaries[executionID]
	return v, ok
}

func (m *Manager) SaveProvenance(executionID string, records []any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.provenance[executionID] = records
}

func (m *Manager) GetProvenance(executionID string) ([]any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.provenance[executionID]
	return v, ok
}

func (m *Manager) RequireStatus(executionID string) (ExecutionStatus, error) {
	status, ok := m.GetStatus(executionID)
	if !ok {
		return ExecutionStatus{}, fmt.Errorf("execution not found: %s", executionID)
	}

	return status, nil
}
