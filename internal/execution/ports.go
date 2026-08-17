package execution

import (
	"context"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

const (
	GuardianEvaluationModeExecution = "execution"
	GuardianEvaluationModeIntent    = "intent"
)

type PolicyPort interface {
	ResolveAndActivatePolicy(context.Context, policyresolver.PolicyLease) (*policyresolver.ResolveResult, error)
}

type WorkflowPort interface {
	Resolve(workflowVersion string) (*WorkflowDefinition, error)
}

type WorkStorePort interface {
	LoadNormalizedBatch(context.Context, string) (*contracts.NormalizedBatch, error)
	SaveManifest(context.Context, contracts.ManifestSummary, []contracts.ManifestDetail) error
}

type GuardianPort interface {
	Decide(context.Context, GuardianEvaluation) (*contracts.GuardianDecision, error)
}

type SearchIndexPort interface {
	Publish(context.Context, contracts.SearchIndexBatch) (*contracts.SearchIndexReceipt, error)
}

type GuardianEvaluation struct {
	Mode            string
	ExecutionID     string
	Request         ExecutionStartRequest
	Envelope        contracts.ActionRequestEnvelope
	PolicyContext   *policyresolver.PolicyContext
	DecisionRequest *contracts.GuardianDecisionRequest
	TierBehavior    TierBehavior
}

type OrchestratorPorts struct {
	Policy   PolicyPort
	Workflow WorkflowPort
	Work     WorkStorePort
	Guardian GuardianPort
	Search   SearchIndexPort
}
