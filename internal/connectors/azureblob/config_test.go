package azureblob

import (
	"strings"
	"testing"
)

func TestConnectionStringFromRequestBuildsFromFriendlyFields(t *testing.T) {
	got := ConnectionStringFromRequest(ConfigureRequest{
		StorageAccount: "acct01",
		AccountKey:     "secret-key",
		EndpointSuffix: "core.windows.net",
	})

	for _, want := range []string{
		"DefaultEndpointsProtocol=https",
		"AccountName=acct01",
		"AccountKey=secret-key",
		"EndpointSuffix=core.windows.net",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("connection string %q does not contain %q", got, want)
		}
	}
}

func TestConnectionStringFromRequestPrefersImportedSecret(t *testing.T) {
	got := ConnectionStringFromRequest(ConfigureRequest{
		ConnectionString: "UseDevelopmentStorage=true",
		StorageAccount:   "acct01",
		AccountKey:       "secret-key",
	})
	if got != "UseDevelopmentStorage=true" {
		t.Fatalf("ConnectionStringFromRequest() = %q, want imported secret", got)
	}
}

func TestSourceCredentialSecretName(t *testing.T) {
	if got := SourceCredentialSecretName(""); got != ConnectionStringSecretName {
		t.Fatalf("SourceCredentialSecretName(empty) = %q, want %q", got, ConnectionStringSecretName)
	}

	got := SourceCredentialSecretName("tenant/a blob source")
	want := "connectors/azureblob/sources/tenant_a_blob_source/connection_string"
	if got != want {
		t.Fatalf("SourceCredentialSecretName() = %q, want %q", got, want)
	}
}
