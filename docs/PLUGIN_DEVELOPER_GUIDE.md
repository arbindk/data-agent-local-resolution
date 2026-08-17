# Everest Plugin Developer Guide

This guide is for customers or partners building Everest-compatible connectors and AI providers.

## Plugin Types

Everest supports two plugin families:

- Connector plugins: discover sources, scan metadata, preview actions, and later execute approved actions.
- AI provider plugins: chat, summarize, classify, explain policy, and recommend governed actions.

Both plugin families are contract-first. The portal validates manifests, invokes runtime endpoints, and runs certification checks before a plugin is treated as portal-ready.

## Connector Plugin Checklist

Required files:

- Connector manifest matching `contracts/connector_manifest.schema.json`
- Runtime endpoint or external process implementing:
  - `contracts/connector_runtime_request.schema.json`
  - `contracts/connector_runtime_response.schema.json`

Required runtime behavior:

- Return `contract_version: connector_runtime_v1`
- Return `safe_mode: true`
- Keep preview and contract-test actions dry-run
- Return `dry_run: true` for preview
- Never apply restricted or destructive actions unless Everest passes apply mode, Guardian allow, HITL approval, and confirmation text
- Do not return secrets

Recommended operations:

- `health`
- `discover_sources`
- `scan_metadata`
- `preview_action`

## AI Provider Plugin Checklist

Required files:

- AI provider manifest matching `contracts/ai_provider_manifest.schema.json`
- Runtime endpoint implementing:
  - `contracts/ai_provider_request.schema.json`
  - `contracts/ai_provider_response.schema.json`

Required runtime behavior:

- Return `contract_version: ai_provider_runtime_v1`
- Return prompt audit metadata
- Declare `autonomous_apply_blocked` in controls
- Return structured JSON
- Never execute connector actions directly
- Treat recommendations as advisory or HITL-ready only
- Do not return secrets

Recommended tasks:

- `chat`
- `summarization`
- `classification`
- `policy_explanation`
- `recommend_actions`
- `hitl_request_prefill`

## Run The Sample Runtime

```powershell
go run ./cmd/sample-plugin-runtime
```

Default endpoints:

- Connector: `http://localhost:7071/connector`
- AI provider: `http://localhost:7071/ai`
- Health: `http://localhost:7071/healthz`

## Contract Test API

Endpoint:

```text
POST /v1/plugins/contract-test
```

Connector example:

```json
{
  "plugin_type": "connector",
  "connector_id": "customer-filesystem",
  "runtime_endpoint": "http://localhost:7071/connector",
  "timeout_seconds": 30,
  "scope": {
    "data_store_id": "contract-test-datastore"
  }
}
```

AI provider example:

```json
{
  "plugin_type": "ai_provider",
  "provider_id": "customer-private-llm",
  "runtime_endpoint": "http://localhost:7071/ai",
  "model": "customer-governance-profile",
  "timeout_seconds": 30
}
```

Expected successful response:

- `status: ready_for_portal`
- `ready_for_portal: true`
- `ready_for_apply: false`
- Required checks have `result: pass`
- Apply remains disabled until production Guardian/HITL verification is configured

## Portal Flow

1. Open **Connector Catalog** or **AI Tools**.
2. Click **Use Sample Runtime** or enter the customer runtime endpoint.
3. Click **Run Contract Test**.
4. Review the readiness report.
5. Fix any failed required checks.
6. Confirm runtime history shows the certification run.

## Environment Notes

Cloud-connected customers can host plugin runtimes behind approved outbound paths.

Private-network customers should use private endpoints, internal DNS, and customer-managed credentials.

Air-gapped customers should run plugin runtimes locally with imported manifests, policies, and workflow bundles.

Everest does not require internet access for the sample runtime or local contract testing.
