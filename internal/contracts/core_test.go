package contracts

import (
	"strings"
	"testing"
	"time"
)

func TestActionRequestEnvelopeValidateRequiresCoreFields(t *testing.T) {
	envelope := ActionRequestEnvelope{}

	err := envelope.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty envelope")
	}

	msg := err.Error()
	for _, field := range []string{"request_id", "tenant_id", "actor.id", "source_id", "connector_id", "operation", "correlation_id", "policy_context"} {
		if !strings.Contains(msg, field) {
			t.Fatalf("expected missing field %q in validation message: %s", field, msg)
		}
	}
}

func TestActionRequestEnvelopeValidateAcceptsMinimumEnterpriseEnvelope(t *testing.T) {
	envelope := ActionRequestEnvelope{
		RequestID:   "req-001",
		TenantID:    "tenant-a",
		Actor:       ActorContext{ID: "data-owner@example.com", Role: "Data Owner"},
		SourceID:    "finance-azureblob",
		ConnectorID: "azure_blob",
		Operation:   "quick_scan",
		Scope: DataScope{
			Container:    "finance",
			Prefix:       "payroll/",
			ZeroCopyMode: "reference_stream_range_or_bounded_preview",
		},
		PolicyContext: PolicyContextRef{
			PolicyID:            "finance-dspm-policy",
			PolicyBundleVersion: "2026.06.01",
			PolicyBundleHash:    "sha256:abc",
		},
		RuntimeContext: RuntimeContext{
			ActionMode:       "dry_run",
			ZeroCopyRequired: true,
		},
		CorrelationID: "corr-001",
		RequestedAt:   time.Now().UTC(),
	}

	if err := envelope.Validate(); err != nil {
		t.Fatalf("expected valid envelope, got %v", err)
	}
}

func TestStableHashIsDeterministicAndNamespaced(t *testing.T) {
	a := StableHash("receipt", "act-001", "gd-001")
	b := StableHash("receipt", "act-001", "gd-001")
	c := StableHash("receipt", "act-002", "gd-001")

	if a != b {
		t.Fatalf("expected deterministic hash, got %s and %s", a, b)
	}
	if a == c {
		t.Fatalf("expected different hash for different input")
	}
	if !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("expected sha256 prefix, got %s", a)
	}
}
