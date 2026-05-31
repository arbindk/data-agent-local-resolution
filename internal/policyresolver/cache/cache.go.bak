package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

type PolicyCache struct {
	mu      sync.RWMutex
	active  map[string]*policyresolver.PolicyContext
	baseDir string
	lkgDir  string
}

func NewPolicyCache(baseDir, lkgDir string) *PolicyCache {
	_ = os.MkdirAll(baseDir, 0755)
	_ = os.MkdirAll(lkgDir, 0755)

	return &PolicyCache{
		active:  make(map[string]*policyresolver.PolicyContext),
		baseDir: baseDir,
		lkgDir:  lkgDir,
	}
}

func (c *PolicyCache) Activate(ctx *policyresolver.PolicyContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.active[ctx.DataStoreID] = ctx

	if err := c.persist(filepath.Join(c.baseDir, ctx.DataStoreID+".json"), ctx); err != nil {
		return err
	}
	if err := c.persist(filepath.Join(c.lkgDir, ctx.DataStoreID+".json"), ctx); err != nil {
		return err
	}

	return nil
}

func (c *PolicyCache) GetActive(dataStoreID string) (*policyresolver.PolicyContext, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.active[dataStoreID]
	return v, ok
}

func (c *PolicyCache) GetLastKnownGood(dataStoreID string) (*policyresolver.PolicyContext, error) {
	path := filepath.Join(c.lkgDir, dataStoreID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read last-known-good: %w", err)
	}

	var ctx policyresolver.PolicyContext
	if err := json.Unmarshal(b, &ctx); err != nil {
		return nil, fmt.Errorf("parse last-known-good: %w", err)
	}

	return &ctx, nil
}

func (c *PolicyCache) persist(path string, ctx *policyresolver.PolicyContext) error {
	tmp := path + ".tmp"

	b, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal policy context: %w", err)
	}

	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("write temp policy cache: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("atomic cache rename: %w", err)
	}

	return nil
}
