package secretstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFileProviderSaveLoadDelete(t *testing.T) {
	store := NewWithOptions(Options{
		Provider: ProviderLocalFile,
		FilePath: filepath.Join(t.TempDir(), "secrets.json"),
	})

	if err := store.Save("connectors/azureblob/source-1", "secret-value"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load("connectors/azureblob/source-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("Load() = %q, want secret-value", got)
	}

	if !store.Exists("connectors/azureblob/source-1") {
		t.Fatalf("Exists() = false, want true")
	}

	if err := store.Delete("connectors/azureblob/source-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Exists("connectors/azureblob/source-1") {
		t.Fatalf("Exists() after Delete() = true, want false")
	}
}

func TestEnvironmentProviderLoad(t *testing.T) {
	store := NewWithOptions(Options{
		Provider:  ProviderEnvironment,
		EnvPrefix: "TEST_EVEREST_SECRET_",
	})
	t.Setenv("TEST_EVEREST_SECRET_CONNECTORS_AZUREBLOB_CONNECTION_STRING", "from-env")

	got, err := store.Load("connectors/azureblob/connection_string")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "from-env" {
		t.Fatalf("Load() = %q, want from-env", got)
	}

	if err := store.Save("connectors/azureblob/connection_string", "new-value"); err == nil {
		t.Fatalf("Save() with environment provider succeeded, want read-only error")
	}
}

func TestUnsupportedProviderReportsContractMode(t *testing.T) {
	store := NewWithOptions(Options{Provider: ProviderAzureKeyVault})
	status := store.Status()
	if status.Provider != ProviderAzureKeyVault {
		t.Fatalf("Provider = %q, want %q", status.Provider, ProviderAzureKeyVault)
	}
	if status.Configured {
		t.Fatalf("Configured = true, want false")
	}
	if err := store.Save("name", "value"); err == nil {
		t.Fatalf("Save() with unsupported provider succeeded, want error")
	}
}

func TestEnvSecretName(t *testing.T) {
	got := envSecretName("connectors/azureblob/sources/tenant-1/connection_string")
	want := "CONNECTORS_AZUREBLOB_SOURCES_TENANT_1_CONNECTION_STRING"
	if got != want {
		t.Fatalf("envSecretName() = %q, want %q", got, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
