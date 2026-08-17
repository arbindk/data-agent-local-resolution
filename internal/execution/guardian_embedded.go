package execution

import (
	"context"
	"fmt"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type EmbeddedGuardian struct{}

func NewEmbeddedGuardian() EmbeddedGuardian {
	return EmbeddedGuardian{}
}

func (EmbeddedGuardian) Decide(ctx context.Context, input GuardianEvaluation) (*contracts.GuardianDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.DecisionRequest == nil {
		return nil, fmt.Errorf("guardian decision request is required")
	}
	if input.PolicyContext == nil {
		return nil, fmt.Errorf("policy context is required")
	}

	switch input.Mode {
	case GuardianEvaluationModeExecution:
		return buildGuardianDecision(
			input.Request,
			input.Envelope,
			input.PolicyContext,
			input.DecisionRequest,
			input.TierBehavior,
		), nil
	case GuardianEvaluationModeIntent:
		return buildGuardianDecisionFromEnvelope(
			input.Envelope,
			input.PolicyContext,
			input.DecisionRequest,
		), nil
	default:
		return nil, fmt.Errorf("unsupported guardian evaluation mode: %s", input.Mode)
	}
}
