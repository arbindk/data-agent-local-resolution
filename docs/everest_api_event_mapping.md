# Everest API and Event Mapping

Version: Draft 1  
Product: Everest Enterprise Data Operations - Azure Blob Edition

## Purpose

This document maps Everest UI actions and backend modules to APIs, payloads, outputs, errors, and test cases. It is written for developers, QA, support engineers, and new joiners.

## Base URLs

| Service | Base URL | Notes |
|---|---|---|
| Data Agent API | `http://localhost:8080` | Serves the portal and local data operations. |
| Control Plane API | `http://localhost:9090` | Platform/governance APIs when running separately. |
| Sample Plugin Runtime | `http://localhost:7071` | Demo connector/AI runtime. |

## Authentication And Authorization

Current local/demo implementation does not enforce full production authentication/RBAC. Enterprise production should add:

- User authentication.
- Role-based authorization.
- Per-action permission checks.
- Audit identity propagation.
- Secret access restrictions.

For local testing, APIs are usually called directly by the browser UI.

## API Groups

| Group | Purpose |
|---|---|
| Health | Check service availability. |
| Azure Blob Config | Save/load/test Azure Blob source settings. |
| Azure Blob Workbench | Run object test operations from UI. |
| DSPM | Run content intelligence and catalog supported checks/models/keywords. |
| Azure Blob Actions | Preview/apply governed Azure Blob actions. |
| AI Providers | Configure and invoke AI provider runtimes. |
| Connector Runtime | Catalog and invoke connector runtime contracts. |
| Execution | Start and inspect executions/jobs. |
| Evidence/Store | View normalized files, manifests, provenance, and exports. |
| Policy/Governance | Resolve policies, evaluate governance, verify artifact trust. |
| Guardian | View safety decisions. |

## 1. Health API

### `GET /healthz`

Purpose:

Checks whether the Data Agent service is running.

Request source:

- Dashboard badges.
- Diagnostics screen.
- External health monitor.

Sample response:

```json
{
  "status": "ok",
  "service": "data-agent-local-resolution"
}
```

Error cases:

- Service not running.
- Port blocked.

Retry behavior:

- Safe to retry.

Test cases:

| Test ID | Scenario | Expected |
|---|---|---|
| API-HEALTH-001 | Service running | HTTP 200 and status ok. |
| API-HEALTH-002 | Service stopped | Browser or client connection error. |

## 2. Azure Blob Configuration APIs

### `GET /v1/connectors/azureblob/config`

Purpose:

Returns public Azure Blob runtime configuration without exposing the connection string.

Request source:

- Source Inventory.
- Workbench initialization.

Sample response:

```json
{
  "container": "customer-demo",
  "prefix": "hr/payroll",
  "data_store_id": "azureblob-hulkb01",
  "batch_id": "batch-azure-001",
  "action_mode": "dry_run",
  "writeback_enabled": "false",
  "credential_set": true
}
```

Important output fields:

| Field | Meaning |
|---|---|
| container | Saved default container. |
| prefix | Saved scan prefix. |
| data_store_id | Logical source ID. |
| credential_set | Whether secret exists. |

Error cases:

- Service unavailable.

Retry/idempotency:

- Safe GET, can retry.

### `POST /v1/connectors/azureblob/config`

Purpose:

Saves Azure Blob source configuration and optionally stores connection string in secret store.

Request source:

- Source Inventory Save Config.
- Azure Blob Workbench Save Credential.

Sample request:

```json
{
  "connection_string": "DefaultEndpointsProtocol=...",
  "container": "customer-demo",
  "prefix": "hr/payroll",
  "data_store_id": "azureblob-hulkb01",
  "batch_id": "batch-azure-001",
  "action_mode": "dry_run",
  "writeback_enabled": "false"
}
```

Credential-only request:

```json
{
  "connection_string": "DefaultEndpointsProtocol=..."
}
```

Sample response:

```json
{
  "status": "saved",
  "config": {
    "container": "customer-demo",
    "credential_set": true
  }
}
```

Error cases:

| Error | Cause |
|---|---|
| `container is required` | Full config save without container. |
| `data_store_id is required` | Full config save without data store ID. |
| `batch_id is required` | Full config save without batch ID. |
| secret save error | Local secret store issue. |

Retry/idempotency:

- Safe to retry with same config.
- Replaces runtime config values.
- Stores/replaces credential if provided.

Test cases:

| Test ID | Scenario | Expected |
|---|---|---|
| API-AZCFG-001 | Save credential only | credential_set true. |
| API-AZCFG-002 | Save full config | status saved. |
| API-AZCFG-003 | Save missing container | HTTP 400. |

### `POST /v1/connectors/azureblob/test`

Purpose:

Tests saved Azure Blob config by listing from configured container/prefix.

Sample response:

```json
{
  "status": "ok",
  "source_system": "azure_blob",
  "container": "customer-demo",
  "prefix": "hr/payroll",
  "data_store_id": "azureblob-hulkb01",
  "batch_id": "batch-azure-001",
  "action_mode": "dry_run",
  "writeback_enabled": "false"
}
```

Error cases:

- Missing connection string.
- Missing container.
- Invalid Azure credential.
- Network/private endpoint unavailable.
- Insufficient Azure permission.

Retry/idempotency:

- Safe to retry.

## 3. Azure Blob Container API

### `GET /v1/connectors/azureblob/containers`

Purpose:

Lists containers using saved Azure connection string.

Sample response:

```json
{
  "status": "ok",
  "source_system": "azure_blob",
  "containers": ["customer-demo", "policy-test"],
  "count": 2
}
```

Note:

The newer Workbench endpoint also supports container discovery and returns richer metadata. Prefer Workbench for UI discovery.

## 4. Azure Blob Workbench API

### `POST /v1/tools/azureblob/workbench`

Purpose:

Runs safe portal-driven Azure Blob test operations.

Request source:

- Azure Blob Workbench.
- Source Inventory container discovery.

Supported operations:

| Operation | Purpose | Required Fields |
|---|---|---|
| list_containers | List containers. | connection string or saved credential |
| list_blobs | List blobs. | container |
| view_blob | View properties/tags/preview. | container, blob |
| create_blob | Create blob. | container, blob |
| update_blob | Replace blob content. | container, blob |
| set_metadata | Replace metadata. | container, blob, metadata |
| set_tags | Replace index tags. | container, blob, tags |
| download_blob | Return browser-download payload. | container, blob |
| delete_blob | Delete blob. | container, blob, confirm=DELETE unless dry_run |

Sample request:

```json
{
  "operation": "view_blob",
  "container": "customer-demo",
  "blob": "everest-test/sample.txt",
  "max_bytes": 4096
}
```

Sample response:

```json
{
  "operation": "view_blob",
  "status": "ok",
  "auth_mode": "saved_connection_string",
  "container": "customer-demo",
  "blob": "everest-test/sample.txt",
  "details": {
    "name": "everest-test/sample.txt",
    "size_bytes": 32,
    "content_type": "text/plain",
    "metadata": {
      "owner": "qa"
    },
    "tags": {
      "everest_test": "true"
    },
    "preview": {
      "bytes": 32,
      "truncated": false,
      "text": "Everest test object"
    }
  }
}
```

Delete dry-run request:

```json
{
  "operation": "delete_blob",
  "container": "customer-demo",
  "blob": "everest-test/sample.txt",
  "dry_run": true
}
```

Delete apply request:

```json
{
  "operation": "delete_blob",
  "container": "customer-demo",
  "blob": "everest-test/sample.txt",
  "confirm": "DELETE"
}
```

Error cases:

| Error | Cause |
|---|---|
| connection string is required | No saved/request credential. |
| container is required | Blob operation without container. |
| blob path is required | Object operation without blob. |
| delete requires confirmation text DELETE | Missing confirmation. |
| blob already exists | Create without overwrite. |
| Azure 403 | Permission missing. |
| Azure 404 | Container/blob missing. |

Retry/idempotency:

| Operation | Retry Notes |
|---|---|
| list/view/download | Safe to retry. |
| create | Not idempotent unless overwrite or same content accepted. |
| update | Replaces content on retry. |
| set_metadata/set_tags | Replaces values, retry is usually safe with same payload. |
| delete | Retrying after success may return not found. |

## 5. DSPM APIs

### `GET /v1/dspm/catalog`

Purpose:

Returns supported DSPM tiers, checks, model profiles, keyword packs, and entity types.

Sample response:

```json
{
  "status": "ok",
  "tiers": ["metadata", "rules", "model", "deep"],
  "checks": [],
  "model_profiles": [],
  "keyword_packs": [],
  "entity_types": ["EMAIL", "SECRET", "KEYWORD"]
}
```

Retry/idempotency:

- Safe to retry.

### `POST /v1/dspm/content-scan`

Purpose:

Runs content intelligence scan for an object sample.

Sample request:

```json
{
  "source_system": "azure_blob",
  "connector_id": "azureblob",
  "file_id": "azureblob:customer-demo:hr/payroll.txt",
  "container": "customer-demo",
  "path": "hr/payroll.txt",
  "content_type": "text/plain",
  "size_bytes": 1024,
  "sample_text": "employee jane@example.com token=abcdef1234567890",
  "scan_tier": "rules",
  "model_profile": "builtin-rules",
  "keyword_profile": "builtin-default",
  "keyword_patterns": [
    {
      "pattern": "Project Everest",
      "severity": "high",
      "entity_type": "CUSTOM_ENTITY",
      "match_mode": "phrase"
    }
  ],
  "keyword_exclusions": ["sample", "template"]
}
```

Sample response:

```json
{
  "status": "completed",
  "scan_id": "dspm-abc123",
  "scan_tier": "rules",
  "model_profile": "builtin-rules",
  "classification": "restricted",
  "confidence": 0.9,
  "risk_score": 85,
  "requires_hitl": true,
  "entities": [
    {
      "type": "EMAIL",
      "value_redacted": "ja***@example.com",
      "detector": "builtin_regex_email",
      "confidence": 0.92
    }
  ],
  "recommended_tags": {
    "everest_dspm_classification": "restricted",
    "everest_dspm_hitl": "true"
  }
}
```

Error cases:

- Invalid JSON.
- Invalid regex custom keyword is ignored by current engine rather than crashing.

Retry/idempotency:

- Safe to retry.
- Current result is not persisted by default. Persistence is planned.

## 6. Azure Blob Scan API

### `POST /v1/connectors/azureblob/scan`

Purpose:

Runs Azure Blob metadata scan using saved config and writes normalized records to local working store.

Sample response:

```json
{
  "status": "completed",
  "source_system": "azure_blob",
  "container": "customer-demo",
  "prefix": "hr",
  "data_store_id": "azureblob-hulkb01",
  "batch_id": "batch-azure-001",
  "records_scanned": 42,
  "records_written": 42,
  "store": "local_ephemeral_db",
  "action_mode": "dry_run",
  "writeback": "false"
}
```

Error cases:

- Missing config.
- Azure list failure.
- Local store write failure.

Retry/idempotency:

- Replaces normalized files for batch in current local store behavior.

## 7. Azure Blob Action API

### `POST /v1/connectors/azureblob/actions/execute`

Purpose:

Previews or executes governed Azure Blob actions.

Supported operations:

- apply_blob_metadata
- apply_blob_index_tags
- archive
- retrieve
- rehydrate
- quarantine
- migration
- delete

Sample request:

```json
{
  "operation": "apply_blob_index_tags",
  "file_id": "azureblob:customer-demo:everest-test/sample.txt",
  "index_tags": {
    "classification": "restricted"
  },
  "action_mode": "dry_run",
  "writeback_enabled": false,
  "guardian_decision": "allow",
  "preview_only": true,
  "execution_id": "exec-azureblob-20260613"
}
```

Sample response:

```json
{
  "status": "completed",
  "reason": "Dry-run action plan created. Azure Blob was not modified.",
  "connector_result": {
    "operation": "apply_blob_index_tags",
    "container": "customer-demo",
    "blob_path": "everest-test/sample.txt"
  }
}
```

Error cases:

| Error | Cause |
|---|---|
| object required | Missing file_id and blob_path. |
| restricted action requires confirmation | Missing APPLY/DELETE or approval. |
| HITL required | No approved request for risky action. |
| Azure operation failure | Permission/network/object issue. |

Retry/idempotency:

- Preview is safe.
- Metadata/tags retry is safe with same payload.
- Archive/tier actions are mostly idempotent by tier.
- Delete retry after success may fail with not found.
- Migration/quarantine copy may create duplicate/overwritten target depending target path.

## 8. AI Provider APIs

### `GET /v1/ai/providers/catalog`

Purpose:

Lists AI providers and supported deployment modes.

### `GET /v1/ai/providers/config`

Purpose:

Returns public AI provider config without exposing API key.

### `POST /v1/ai/providers/config`

Purpose:

Saves AI provider runtime settings and optional secret.

Sample request:

```json
{
  "provider_id": "customer-private-llm",
  "endpoint_url": "http://localhost:7071/ai",
  "model": "local-model",
  "auth_scheme": "none",
  "network_mode": "air_gapped",
  "automation_mode": "hitl_required",
  "redaction_mode": "strict"
}
```

### `POST /v1/ai/providers/test`

Purpose:

Validates provider configuration.

### `POST /v1/ai/runtime/invoke`

Purpose:

Invokes selected AI runtime.

Error cases:

- Provider unavailable.
- Auth failure.
- Invalid manifest/runtime response.

Retry/idempotency:

- Test/invoke can retry, but AI output may vary.

## 9. AI Azure Blob Recommendation API

### `POST /v1/ai/azureblob/recommendations`

Purpose:

Returns AI recommendation for Azure Blob evidence/object/action context.

Request source:

- Workbench AI action verbs.
- Action Center Ask AI.

Sample request:

```json
{
  "data_store_id": "azureblob-hulkb01",
  "execution_id": "exec-azureblob-20260613",
  "prompt": "Generate index tags",
  "operation": "apply_blob_index_tags",
  "file_id": "azureblob:customer-demo:hr/payroll.txt",
  "blob_path": "hr/payroll.txt",
  "container": "customer-demo",
  "policy_ids": ["azureblob-approved-tagging", "destructive-action-safety"]
}
```

Expected behavior:

- Returns recommendation if AI provider works.
- UI has local guardrail fallback if AI runtime is unavailable.

Retry/idempotency:

- Safe to retry, output may vary.

## 10. Connector Catalog And Runtime APIs

### `GET /v1/connectors/catalog`

Purpose:

Lists connector definitions and capabilities.

### `POST /v1/connectors/manifest/validate`

Purpose:

Validates connector manifest schema/contract.

### `POST /v1/connectors/runtime/invoke`

Purpose:

Invokes external connector runtime operation.

Sample request:

```json
{
  "connector_id": "customer-filesystem",
  "operation": "scan_metadata",
  "runtime_endpoint": "http://localhost:7071/connector",
  "scope": {
    "container": "customer-demo",
    "prefix": "hr"
  },
  "dry_run": true
}
```

Error cases:

- Runtime endpoint down.
- Invalid operation.
- Contract check failure.

Retry/idempotency:

- Health/discovery safe.
- Execute action depends on runtime operation.

## 11. Execution APIs

### `POST /v1/execution/azureblob/start`

Purpose:

Starts a governed Azure Blob execution using saved config.

### `POST /v1/execution/start`

Purpose:

Starts generic execution from explicit request.

### `GET /v1/execution/{execution_id}`

Purpose:

Returns execution status.

### `GET /v1/execution/{execution_id}/summary`

Purpose:

Returns execution summary.

### `GET /v1/execution/{execution_id}/provenance`

Purpose:

Returns execution provenance.

Error cases:

- Missing execution ID.
- Execution not found.
- Workflow/policy errors.

Retry/idempotency:

- GETs safe.
- Start APIs create new execution if called repeatedly.

## 12. Local Store And Evidence APIs

### `GET /v1/localstore/tables`

Purpose:

Lists local store tables/views.

### `GET /v1/localstore/normalized-files`

Purpose:

Returns normalized file records.

Query parameters:

- `batch_id`
- `limit`
- `offset`

### Manifest And Export APIs

Examples:

- `GET /v1/manifests/summary?execution_id={id}`
- `GET /v1/manifests/details?execution_id={id}`
- `GET /v1/provenance?execution_id={id}`
- `GET /v1/exports/azureblob/{type}?execution_id={id}&format=json`

Export types:

- executive-summary
- manifest
- guardian
- compliance
- ai-readiness

Retry/idempotency:

- Safe to retry.

## 13. Policy And Governance APIs

### `POST /v1/policy/resolve`

Purpose:

Resolves policy context for a lease/data store.

### `GET /v1/policy/artifact/trust`

Purpose:

Checks whether policy artifact is trusted.

### `GET /v1/governance/context`

Purpose:

Returns governance context.

### `POST /v1/governance/evaluate`

Purpose:

Evaluates a requested action against governance rules.

Error cases:

- Missing data_store_id.
- Missing role/owner for restricted action.
- Policy artifact unavailable.

Retry/idempotency:

- Evaluate is safe to retry.

## 14. Guardian APIs

### `GET /v1/guardian/decisions`

Purpose:

Returns Guardian decisions for execution/action context.

Query parameters:

- `execution_id`
- `limit`

Retry/idempotency:

- Safe to retry.

## 15. Approval/HITL APIs

Current UI attempts to submit HITL via platform approval endpoints and falls back to governance/action handoff when unavailable.

Expected production endpoints:

- `GET /v1/approvals`
- `POST /v1/approvals`
- `POST /v1/approvals/{id}/decision`

Sample HITL request:

```json
{
  "tenant_id": "local-tenant",
  "requested_by": "workbench-ai",
  "role": "data_steward",
  "operation": "quarantine",
  "source_region": "IN",
  "target_region": "IN",
  "data_store_id": "azureblob-hulkb01",
  "file_id": "azureblob:customer-demo:hr/payroll.txt",
  "reason": "DSPM classified object as restricted",
  "policy_ids": ["destructive-action-safety", "azureblob-approved-tagging"]
}
```

Production requirements:

- Persist pending approvals.
- Record reviewer identity.
- Record decision reason.
- Link approval to action result.
- Prevent apply until approval exists.

## 16. Event And Background Job Mapping

Current implementation is mostly request/response. Planned enterprise event model:

| Event/Job | Producer | Consumer | Purpose |
|---|---|---|---|
| source_config_saved | Source Inventory | Audit/config history | Track source changes. |
| container_discovered | Workbench | UI/audit | Track discovery. |
| blob_workbench_operation | Workbench API | Operation history | Audit test operations. |
| dspm_scan_completed | DSPM API | DSPM store/dashboard | Persist classification. |
| ai_recommendation_created | AI API | AI audit/HITL | Trace AI rationale. |
| hitl_requested | Workbench/Action Center | Approval queue | Human review. |
| guardian_decision_created | Guardian | Action Center/evidence | Safety gate proof. |
| azure_action_applied | Action API | Evidence/audit | Proof of mutation. |
| deep_scan_requested | DSPM | Background worker | Delegated full content scan. |

## 17. API Regression Checklist

| Check | Expected |
|---|---|
| All GET APIs handle empty state. | No UI crash, safe empty JSON. |
| All POST APIs reject invalid JSON. | HTTP 400 with error. |
| Missing Azure credential is clear. | Actionable error. |
| Delete requires confirmation. | No delete without `DELETE`. |
| Dry-run does not mutate Azure. | Status preview. |
| DSPM invalid regex does not crash service. | Pattern ignored or safe error. |
| AI unavailable has fallback or clear error. | UI remains usable. |
| Azure 403/404 errors are surfaced. | User can troubleshoot permissions/path. |

## Open API Gaps

| Gap | Required Follow-Up |
|---|---|
| Production RBAC/auth missing. | Add auth middleware and role checks. |
| Workbench history not persisted. | Add operation history API/table. |
| DSPM results not persisted. | Add DSPM store and query APIs. |
| HITL endpoint contract should be finalized. | Define approval request/decision schema. |
| Model runtime invocation is catalog/contract only. | Add model runtime API for local BERT/domain model. |
| Generic Connector Workbench not finalized. | Extract Azure Blob Workbench contract to connector-agnostic API. |
