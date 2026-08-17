# Everest AI Provider Framework Architecture

Everest treats AI as a configurable capability provider rather than a hard-coded service. Customers can run with no internet, private endpoints, cloud AI, or their own AI gateway while the portal keeps the same safety posture: AI can recommend, summarize, classify, explain policy, and prepare HITL requests, but executable storage actions remain governed by Guardian and approval controls.

## Product Goals

- Support internet-connected, private-network, hybrid, and air-gapped customers.
- Let customers choose Azure OpenAI, OpenAI API, local runtimes, OpenAI-compatible private endpoints, or customer-owned AI gateways.
- Keep prompts and evidence governed through audit, redaction, and evidence-grounding policy.
- Allow AI to submit pending HITL requests only when the provider configuration permits it.
- Keep autonomous apply disabled for enterprise safety.
- Make custom provider onboarding manifest-driven so future AI tools do not require portal rewrites.

## Provider Contract

AI providers are declared with `ai_provider_manifest.schema.json`. A manifest defines:

- Provider identity, type, status, vendor, and version.
- Deployment support: `cloud_connected`, `private_network`, `air_gapped`, `hybrid`.
- Runtime shape: `builtin`, `http`, `local_http`, `custom_http`, or `external_process`.
- Auth schemes and secret fields.
- Config fields shown by the portal.
- Model/profile options.
- AI capabilities such as chat, classification, summarization, policy explanation, recommendation, and action planning.
- Safety posture for HITL, prompt audit, evidence grounding, redaction, and autonomous apply.
- Data handling posture for residency, retention, private endpoint, and air-gap support.

## Built-In Provider Options

- `everest-local-policy-copilot`: available, no external model call, air-gap ready.
- `azure-openai`: configurable for customer Azure OpenAI deployments and private endpoints.
- `openai-api`: configurable for cloud-connected environments with permitted internet egress.
- `local-ollama`: configurable for customer local model runtimes.
- `openai-compatible-local`: configurable for vLLM, LM Studio, or private OpenAI-compatible gateways.
- `custom-http-ai`: configurable bring-your-own AI provider using the Everest provider contract.

## Runtime Safety

The current implementation uses the selected provider configuration for policy, audit, and UI behavior. Azure Blob recommendations remain evidence-grounded and local in this build. This means provider readiness can be validated without sending a customer prompt to a model. Future provider adapters can call configured endpoints behind the same contract while preserving:

- `prompt_audit_required=true`
- `evidence_grounding_required=true`
- `autonomous_apply_allowed=false`
- Guardian before action execution
- HITL before restricted or destructive actions

## Customer-Owned Provider Flow

1. Customer creates an AI provider manifest.
2. Manifest is placed under `ai-providers`, `demo/ai-providers`, or a directory listed in `EVEREST_AI_PROVIDER_MANIFEST_DIRS`.
3. Portal loads the provider catalog and displays the customer provider.
4. Customer configures endpoint, auth scheme, model/profile, network mode, redaction, and HITL mode.
5. Provider readiness test validates contract posture without sending customer evidence.
6. AI Copilot uses the configured provider policy for recommendations and HITL submission behavior.

## API Surface

- `GET /v1/ai/providers/catalog`
- `POST /v1/ai/providers/manifest/validate`
- `GET /v1/ai/providers/config`
- `POST /v1/ai/providers/config`
- `POST /v1/ai/providers/test`
- `POST /v1/ai/azureblob/recommendations`

## Environment Defaults

- `EVEREST_AI_PROVIDER_ID`
- `EVEREST_AI_PROVIDER_ENDPOINT`
- `EVEREST_AI_PROVIDER_MODEL`
- `EVEREST_AI_PROVIDER_DEPLOYMENT`
- `EVEREST_AI_PROVIDER_AUTH_SCHEME`
- `EVEREST_AI_PROVIDER_API_KEY`
- `EVEREST_AI_NETWORK_MODE`
- `EVEREST_AI_AUTOMATION_MODE`
- `EVEREST_AI_REDACTION_MODE`
- `EVEREST_AI_PROMPT_AUDIT_ENABLED`
- `EVEREST_AI_EVIDENCE_GROUNDING_REQUIRED`
- `EVEREST_AI_ALLOW_HITL_SUBMISSION`
- `EVEREST_AI_ALLOW_AUTONOMOUS_APPLY`
- `EVEREST_AI_PROVIDER_MANIFEST_DIRS`
