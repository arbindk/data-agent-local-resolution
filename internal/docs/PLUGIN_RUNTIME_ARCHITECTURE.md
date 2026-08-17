# Everest Plugin Runtime Architecture

The connector and AI provider catalogs describe what a plugin can do. The runtime layer defines how Everest invokes that plugin safely. This gives customers a path to bring their own storage connectors and AI providers without changing the portal for every environment.

## Connector Runtime

Endpoint:

- `POST /v1/connectors/runtime/invoke`

Contract files:

- `contracts/connector_runtime_request.schema.json`
- `contracts/connector_runtime_response.schema.json`

Supported operations:

- `health`
- `discover_sources`
- `scan_metadata`
- `preview_action`
- `execute_action`

Supported runtime kinds:

- `builtin`
- `http`
- `external_process`
- `external_process_or_http`

For `http` plugins, Everest posts the connector runtime request as JSON to the configured runtime endpoint. For `external_process` plugins, Everest sends the same JSON request on stdin and expects a JSON response on stdout.

Apply safety:

- `dry_run` is the default.
- `execute_action` in `apply` mode is blocked unless the manifest-declared action safety controls pass.
- Guardian allow is required when the action declares `requires_guardian=true`.
- HITL approval is required when the action declares `requires_hitl=true`.
- Confirmation text must match the manifest action confirmation, for example `APPLY` or `DELETE`.
- Built-in Azure Blob continues to use its hardened first-party execution path.

## AI Provider Runtime

Endpoint:

- `POST /v1/ai/runtime/invoke`

Contract files:

- `contracts/ai_provider_request.schema.json`
- `contracts/ai_provider_response.schema.json`

Supported tasks:

- `chat`
- `classification`
- `summarization`
- `policy_explanation`
- `recommend_actions`
- `hitl_request_prefill`

Supported runtime kinds:

- `builtin`
- `http`
- `local_http`
- `custom_http`

The built-in Everest Local Policy Copilot returns evidence-grounded local responses and does not send customer prompts outside the environment. HTTP, local HTTP, and custom HTTP providers receive the runtime request as JSON and return `ai_provider_runtime_v1`.

AI safety:

- Prompt audit posture is included on every response.
- Evidence grounding posture is included on every response.
- Autonomous AI apply is blocked.
- AI can recommend or prepare HITL requests, but executable connector actions still go through Guardian and HITL.
- Secrets are loaded server-side and are not returned to the portal.

## Customer Plugin Flow

1. Add connector or AI provider manifest.
2. Validate the manifest from the portal.
3. Configure runtime endpoint or entrypoint.
4. Invoke health or test task from the portal.
5. Implement scan/recommend/preview response using the runtime schema.
6. Keep apply actions behind Everest approval and Guardian gates.

## Sample Runtime

Everest includes a local sample runtime that implements both contracts:

```powershell
go run ./cmd/sample-plugin-runtime
```

Default endpoints:

- Connector runtime: `http://localhost:7071/connector`
- AI provider runtime: `http://localhost:7071/ai`
- Health: `http://localhost:7071/healthz`

Portal test flow:

1. Start the sample runtime.
2. Open **Connector Catalog** or **AI Tools**.
3. Click **Use Sample Runtime**.
4. Invoke `discover_sources`, `scan_metadata`, `summarization`, or `recommend_actions`.
5. Refresh runtime history to verify the audit record.

The sample runtime is dry-run only and does not modify customer systems.
