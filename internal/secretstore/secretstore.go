package secretstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/danieljoos/wincred"
)

const (
	targetPrefix = "EverestDataAgent/"

	ProviderWindowsCredentialManager = "windows_credential_manager"
	ProviderLocalFile                = "local_file"
	ProviderEnvironment              = "environment"
	ProviderAzureKeyVault            = "azure_key_vault"
	ProviderHashiCorpVault           = "hashicorp_vault"
	ProviderKubernetesSecret         = "kubernetes_secret"

	defaultFilePath  = "data/secrets/local_secret_store.json"
	defaultEnvPrefix = "EVEREST_SECRET_"
)

type Options struct {
	Provider  string
	FilePath  string
	EnvPrefix string
}

type Status struct {
	Provider   string `json:"provider"`
	Writable   bool   `json:"writable"`
	Configured bool   `json:"configured"`
	Location   string `json:"location,omitempty"`
	Note       string `json:"note,omitempty"`
}

type backend interface {
	Save(name string, value string) error
	Load(name string) (string, error)
	Delete(name string) error
	Status() Status
}

type Store struct {
	provider string
	backend  backend
}

func New() *Store {
	return NewWithOptions(Options{})
}

func NewWithOptions(opts Options) *Store {
	provider := normalizeProvider(opts.Provider)
	switch provider {
	case ProviderLocalFile:
		path := strings.TrimSpace(opts.FilePath)
		if path == "" {
			path = defaultFilePath
		}
		return &Store{provider: provider, backend: &fileBackend{path: path}}
	case ProviderEnvironment:
		prefix := strings.TrimSpace(opts.EnvPrefix)
		if prefix == "" {
			prefix = defaultEnvPrefix
		}
		return &Store{provider: provider, backend: envBackend{prefix: prefix}}
	case ProviderAzureKeyVault, ProviderHashiCorpVault, ProviderKubernetesSecret:
		return &Store{provider: provider, backend: unsupportedBackend{
			provider: provider,
			reason:   "Provider contract is available, but this local build does not include the customer vault adapter.",
		}}
	default:
		return &Store{provider: ProviderWindowsCredentialManager, backend: windowsBackend{}}
	}
}

func (s *Store) Save(name string, value string) error {
	if s == nil || s.backend == nil {
		return fmt.Errorf("secret store is not configured")
	}
	return s.backend.Save(name, value)
}

func (s *Store) Load(name string) (string, error) {
	if s == nil || s.backend == nil {
		return "", fmt.Errorf("secret store is not configured")
	}
	return s.backend.Load(name)
}

func (s *Store) Exists(name string) bool {
	_, err := s.Load(name)
	return err == nil
}

func (s *Store) Delete(name string) error {
	if s == nil || s.backend == nil {
		return fmt.Errorf("secret store is not configured")
	}
	return s.backend.Delete(name)
}

func (s *Store) Status() Status {
	if s == nil || s.backend == nil {
		return Status{Provider: "none", Writable: false, Configured: false, Note: "Secret store is not configured."}
	}
	return s.backend.Status()
}

type windowsBackend struct{}

func (windowsBackend) Save(name string, value string) error {
	if name == "" {
		return fmt.Errorf("secret name is required")
	}
	if value == "" {
		return fmt.Errorf("secret value is required")
	}

	cred := wincred.NewGenericCredential(targetPrefix + name)
	cred.CredentialBlob = []byte(value)
	cred.Persist = wincred.PersistLocalMachine

	if err := cred.Write(); err != nil {
		return fmt.Errorf("write secret to Windows Credential Manager: %w", err)
	}

	return nil
}

func (windowsBackend) Load(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name is required")
	}

	cred, err := wincred.GetGenericCredential(targetPrefix + name)
	if err != nil {
		return "", fmt.Errorf("read secret from Windows Credential Manager: %w", err)
	}

	return string(cred.CredentialBlob), nil
}

func (windowsBackend) Delete(name string) error {
	if name == "" {
		return fmt.Errorf("secret name is required")
	}

	cred, err := wincred.GetGenericCredential(targetPrefix + name)
	if err != nil {
		return nil
	}

	if cred == nil {
		return nil
	}

	if err := cred.Delete(); err != nil && !errors.Is(err, wincred.ErrElementNotFound) {
		return fmt.Errorf("delete secret from Windows Credential Manager: %w", err)
	}

	return nil
}

func (windowsBackend) Status() Status {
	return Status{
		Provider:   ProviderWindowsCredentialManager,
		Writable:   true,
		Configured: true,
		Location:   "Windows Credential Manager",
	}
}

type fileBackend struct {
	path string
	mu   sync.Mutex
}

type fileState struct {
	Version string            `json:"version"`
	Secrets map[string]string `json:"secrets"`
}

func (b *fileBackend) Save(name string, value string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("secret name is required")
	}
	if value == "" {
		return fmt.Errorf("secret value is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	state, err := b.readLocked()
	if err != nil {
		return err
	}
	state.Secrets[name] = value
	return b.writeLocked(state)
}

func (b *fileBackend) Load(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("secret name is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	state, err := b.readLocked()
	if err != nil {
		return "", err
	}
	value, ok := state.Secrets[name]
	if !ok || value == "" {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return value, nil
}

func (b *fileBackend) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("secret name is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	state, err := b.readLocked()
	if err != nil {
		return err
	}
	delete(state.Secrets, name)
	return b.writeLocked(state)
}

func (b *fileBackend) Status() Status {
	return Status{
		Provider:   ProviderLocalFile,
		Writable:   true,
		Configured: strings.TrimSpace(b.path) != "",
		Location:   b.path,
		Note:       "Local file secret provider is for air-gapped labs and developer/test environments; production should use an approved vault.",
	}
}

func (b *fileBackend) readLocked() (fileState, error) {
	state := fileState{Version: "secret_store_v1", Secrets: map[string]string{}}
	data, err := os.ReadFile(b.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, fmt.Errorf("read local secret file: %w", err)
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("parse local secret file: %w", err)
	}
	if state.Secrets == nil {
		state.Secrets = map[string]string{}
	}
	if state.Version == "" {
		state.Version = "secret_store_v1"
	}
	return state, nil
}

func (b *fileBackend) writeLocked(state fileState) error {
	if state.Secrets == nil {
		state.Secrets = map[string]string{}
	}
	if state.Version == "" {
		state.Version = "secret_store_v1"
	}
	if err := os.MkdirAll(filepath.Dir(b.path), 0700); err != nil {
		return fmt.Errorf("create local secret directory: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode local secret file: %w", err)
	}
	if err := os.WriteFile(b.path, data, 0600); err != nil {
		return fmt.Errorf("write local secret file: %w", err)
	}
	return nil
}

type envBackend struct {
	prefix string
}

func (b envBackend) Save(name string, value string) error {
	return fmt.Errorf("environment secret provider is read-only; set %s instead", b.envKey(name))
}

func (b envBackend) Load(name string) (string, error) {
	key := b.envKey(name)
	if key == "" {
		return "", fmt.Errorf("secret name is required")
	}
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("environment secret %s is not set", key)
	}
	return value, nil
}

func (b envBackend) Delete(name string) error {
	return fmt.Errorf("environment secret provider is read-only; unset %s outside Everest", b.envKey(name))
}

func (b envBackend) Status() Status {
	return Status{
		Provider:   ProviderEnvironment,
		Writable:   false,
		Configured: strings.TrimSpace(b.prefix) != "",
		Location:   b.prefix + "<SECRET_NAME>",
		Note:       "Environment provider is read-only and suitable for locked-down or pre-provisioned deployments.",
	}
}

func (b envBackend) envKey(name string) string {
	return strings.TrimSpace(b.prefix) + envSecretName(name)
}

type unsupportedBackend struct {
	provider string
	reason   string
}

func (b unsupportedBackend) Save(name string, value string) error {
	return fmt.Errorf("%s secret provider is not connected: %s", b.provider, b.reason)
}

func (b unsupportedBackend) Load(name string) (string, error) {
	return "", fmt.Errorf("%s secret provider is not connected: %s", b.provider, b.reason)
}

func (b unsupportedBackend) Delete(name string) error {
	return fmt.Errorf("%s secret provider is not connected: %s", b.provider, b.reason)
}

func (b unsupportedBackend) Status() Status {
	return Status{
		Provider:   b.provider,
		Writable:   false,
		Configured: false,
		Note:       b.reason,
	}
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "windows", "wincred", "credential_manager", ProviderWindowsCredentialManager:
		return ProviderWindowsCredentialManager
	case "file", "local", "air_gapped_file", ProviderLocalFile:
		return ProviderLocalFile
	case "env", "environment_variables", ProviderEnvironment:
		return ProviderEnvironment
	case "keyvault", "key_vault", ProviderAzureKeyVault:
		return ProviderAzureKeyVault
	case "vault", "hashicorp", ProviderHashiCorpVault:
		return ProviderHashiCorpVault
	case "k8s", "kubernetes", "kubernetes_secrets", ProviderKubernetesSecret:
		return ProviderKubernetesSecret
	default:
		return ProviderWindowsCredentialManager
	}
}

func envSecretName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	lastSeparator := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
			lastSeparator = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			lastSeparator = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSeparator = false
		default:
			if !lastSeparator {
				b.WriteByte('_')
				lastSeparator = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
