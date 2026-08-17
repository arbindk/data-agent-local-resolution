package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/execution"
)

type dacIntentInput struct {
	Operation       string
	SourceID        string
	ConnectorID     string
	ActionMode      string
	Container       string
	Prefix          string
	Blob            string
	TargetContainer string
	TargetBlob      string
	Limit           int
	ObjectRefs      []contracts.ObjectRef
	TargetRefs      []contracts.ObjectRef
	Metadata        map[string]string
}

func (s *Server) evaluateDACIntent(r *http.Request, input dacIntentInput) (*execution.IntentEvaluationResult, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("data agent orchestrator is not configured")
	}
	envelope := s.buildDACActionRequestEnvelope(r, input)
	return s.orchestrator.EvaluateIntent(r.Context(), envelope)
}

func (s *Server) buildDACActionRequestEnvelope(r *http.Request, input dacIntentInput) contracts.ActionRequestEnvelope {
	actor := s.enterpriseActor(r)
	cfg := s.azureBlobConfig()
	now := time.Now().UTC()
	sourceID := firstNonBlank(input.SourceID, cfg.DataStoreID, cfg.StorageAccount, "azureblob-local")
	connectorID := firstNonBlank(input.ConnectorID, "azure_blob")
	actionMode := firstNonBlank(input.ActionMode, cfg.ActionMode, execution.ActionModeDryRun)
	operation := strings.TrimSpace(input.Operation)
	requestSeed := strings.Join([]string{
		actor.Tenant,
		actor.Actor,
		sourceID,
		connectorID,
		operation,
		now.Format("20060102150405.000000000"),
	}, "|")
	shortID := strings.TrimPrefix(contracts.StableHash(requestSeed), "sha256:")
	if len(shortID) > 20 {
		shortID = shortID[:20]
	}

	scope := contracts.DataScope{
		SourceID:     sourceID,
		Container:    strings.TrimSpace(input.Container),
		Prefix:       strings.Trim(strings.TrimSpace(input.Prefix), "/"),
		ObjectRefs:   input.ObjectRefs,
		TargetRefs:   input.TargetRefs,
		Limit:        input.Limit,
		Boundary:     "customer_controlled_boundary",
		ZeroCopyMode: "reference_stream_range_or_bounded_preview",
	}
	if scope.ObjectRefs == nil {
		scope.ObjectRefs = []contracts.ObjectRef{}
	}
	if strings.TrimSpace(input.Blob) != "" {
		scope.ObjectRefs = append(scope.ObjectRefs, contracts.ObjectRef{
			URI:         "azureblob:" + firstNonBlank(input.Container, cfg.ContainerName) + ":" + strings.Trim(strings.TrimSpace(input.Blob), "/"),
			ConnectorID: connectorID,
			SourceID:    sourceID,
			Container:   firstNonBlank(input.Container, cfg.ContainerName),
			Path:        strings.Trim(strings.TrimSpace(input.Blob), "/"),
		})
	}
	if scope.TargetRefs == nil {
		scope.TargetRefs = []contracts.ObjectRef{}
	}
	if strings.TrimSpace(input.TargetBlob) != "" || strings.TrimSpace(input.TargetContainer) != "" {
		scope.TargetRefs = append(scope.TargetRefs, contracts.ObjectRef{
			URI:         "azureblob:" + strings.TrimSpace(input.TargetContainer) + ":" + strings.Trim(strings.TrimSpace(input.TargetBlob), "/"),
			ConnectorID: connectorID,
			SourceID:    sourceID,
			Container:   strings.TrimSpace(input.TargetContainer),
			Path:        strings.Trim(strings.TrimSpace(input.TargetBlob), "/"),
		})
	}

	metadata := map[string]string{
		"api_path":          r.URL.Path,
		"api_method":        r.Method,
		"source_system":     connectorID,
		"connector_family":  "object_store",
		"zero_copy_default": "true",
	}
	for key, value := range input.Metadata {
		if strings.TrimSpace(key) != "" {
			metadata[key] = value
		}
	}

	return contracts.ActionRequestEnvelope{
		RequestID:   "req-" + shortID,
		TenantID:    actor.Tenant,
		Actor:       contracts.ActorContext{ID: actor.Actor, Email: actor.Actor, Role: actor.Role, AuthType: "portal_header"},
		SourceID:    sourceID,
		ConnectorID: connectorID,
		Operation:   operation,
		Scope:       scope,
		PolicyContext: contracts.PolicyContextRef{
			PolicyID:            firstNonBlank(s.cfg.ActiveCompiledPolicyVersion, "active-dspm-policy"),
			PolicyBundleVersion: firstNonBlank(s.cfg.ActiveCompiledPolicyVersion, "policy-local"),
			PolicyBundleHash:    firstNonBlank(s.cfg.ActiveCompiledPolicyHash, contracts.StableHash("policy", s.cfg.ActiveCompiledPolicyVersion)),
			WorkflowVersion:     firstNonBlank(s.cfg.ActiveWorkflowVersion, "workflow-local"),
		},
		RuntimeContext: contracts.RuntimeContext{
			Environment:            "enterprise_data_agent_cell",
			Region:                 firstNonBlank(metadata["source_region"], "customer_region"),
			DataBoundary:           "customer_controlled_boundary",
			ActionMode:             actionMode,
			ProcessingTier:         firstNonBlank(s.cfg.DefaultProcessingTier, "tier_2"),
			ProcessingMode:         firstNonBlank(s.cfg.DefaultProcessingMode, "standard"),
			LocalExecutionRequired: true,
			ZeroCopyRequired:       true,
			AirGapped:              strings.EqualFold(metadata["network_mode"], "air_gapped"),
		},
		CorrelationID:  "corr-" + shortID,
		IdempotencyKey: "idem-" + shortID,
		RequestedAt:    now,
		Metadata:       metadata,
	}
}

func dacDecisionString(result *execution.IntentEvaluationResult) string {
	if result == nil || result.GuardianDecision == nil {
		return execution.GuardianDecisionPending
	}
	switch result.GuardianDecision.Decision {
	case contracts.DecisionAllow:
		return execution.GuardianDecisionAllow
	case contracts.DecisionDeny:
		return execution.GuardianDecisionDeny
	case contracts.DecisionReviewRequired, contracts.DecisionFlag:
		return execution.GuardianDecisionReview
	default:
		return result.GuardianDecision.Decision
	}
}

func dacDecisionID(result *execution.IntentEvaluationResult) string {
	if result == nil || result.GuardianDecision == nil {
		return ""
	}
	return result.GuardianDecision.DecisionID
}
