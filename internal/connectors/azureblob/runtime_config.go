package azureblob

import "sync"

type PublicConfig struct {
	StorageAccount   string `json:"storage_account"`
	ContainerName    string `json:"container"`
	Prefix           string `json:"prefix"`
	DataStoreID      string `json:"data_store_id"`
	BatchID          string `json:"batch_id"`
	ActionMode       string `json:"action_mode"`
	WritebackEnabled string `json:"writeback_enabled"`
	CredentialSet    bool   `json:"credential_set"`
}

type RuntimeConfigStore struct {
	mu  sync.RWMutex
	cfg Config
}

func NewRuntimeConfigStore(initial Config) *RuntimeConfigStore {
	return &RuntimeConfigStore{cfg: initial}
}

func (s *RuntimeConfigStore) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *RuntimeConfigStore) Update(next Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = next
}

func (s *RuntimeConfigStore) Public() PublicConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return PublicConfig{
		StorageAccount:   s.cfg.StorageAccount,
		ContainerName:    s.cfg.ContainerName,
		Prefix:           s.cfg.Prefix,
		DataStoreID:      s.cfg.DataStoreID,
		BatchID:          s.cfg.BatchID,
		ActionMode:       s.cfg.ActionMode,
		WritebackEnabled: s.cfg.WritebackEnabled,
		CredentialSet:    s.cfg.ConnectionString != "",
	}
}
