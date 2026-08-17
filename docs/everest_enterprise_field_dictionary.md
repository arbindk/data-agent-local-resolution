# Everest Enterprise Field Dictionary

Version: 3.0  
Scope: Enterprise-safe field definitions for portal, API, evidence, and connector workflows.

## 1. Source Fields

| Field | User-facing label | Purpose | Visibility | Notes |
| --- | --- | --- | --- | --- |
| `source_id` | Source | Stable source identity used across Workbench, scans, jobs, evidence, and actions. | Visible | Selected from dropdown where possible. |
| `data_store_id` | Data store | Enterprise data store identity. | Visible | May match source ID for Azure Blob source. |
| `storage_account` | Storage account | Azure Storage account name. | Visible | User-friendly source setup field. |
| `account_key` | Access key | Secret input used once to save account access. | Input-only | Must be cleared after save and never displayed back. |
| `endpoint_suffix` | Cloud | Azure cloud suffix. | Visible | Example: `core.windows.net`; dropdown preferred. |
| `container` | Container | Azure Blob container scope. | Visible | Dropdown/chips after discovery. |
| `prefix` | Prefix | Object path scope. | Visible | Supports targeted scans and actions. |
| `credential_set` | Account access status | Indicates whether source access exists. | Visible boolean/status | Must not reveal secret value. |

## 2. Secret Fields

| Field | Purpose | Visibility | Storage |
| --- | --- | --- | --- |
| `connection_string` | Backend-generated account access string. | Hidden/support-only | Secret provider. |
| source-specific connection string | Per-source Azure Blob account access. | Hidden | `connectors/azureblob/sources/<source-id>/connection_string`. |
| AI API key/bearer token | AI provider credential. | Hidden/input-only | Secret provider. |

Rules:

- Users should not be asked for raw connection strings in the normal portal flow.
- Backend may accept raw secret fields for migration/support APIs, but they must not appear in normal enterprise UX.
- Secret provider posture may be shown; secret values must not be shown.

## 3. Scan And Job Fields

| Field | Purpose | Visibility |
| --- | --- | --- |
| `job_id` | Scan/action job identity. | Visible |
| `batch_id` | Normalized batch identity. | Visible |
| `status` | queued/running/paused/cancelled/completed/failed. | Visible |
| `checkpoint.container` | Current container checkpoint. | Visible |
| `checkpoint.prefix` | Current prefix checkpoint. | Visible |
| `checkpoint.scope_index` | Current container scope index. | Visible |
| `checkpoint.prefix_index` | Current prefix index within the scope. | Visible |
| `checkpoint.page_marker` | Azure paging marker. | Support-visible |
| `checkpoint.objects_scanned` | Scanned object count. | Visible |
| `checkpoint.pages_scanned` | Scanned page count. | Visible |

## 4. DSPM Fields

| Field | Purpose | Visibility |
| --- | --- | --- |
| `entity_type` | PII, PHI, PCI, SECRET, LEGAL, or custom entity. | Visible |
| `severity` | low/medium/high/critical. | Visible |
| `confidence` | Detection confidence. | Visible |
| `classification` | Suggested data classification. | Visible |
| `recommended_tags` | Suggested Azure Blob index tags. | Visible |
| `recommended_metadata` | Suggested metadata. | Visible |

## 5. Action Fields

| Field | Purpose | Visibility |
| --- | --- | --- |
| `operation` | Requested action such as tag, metadata, copy, move-prep, tier, rehydrate, delete. | Visible |
| `dry_run` | Preview mode flag. | Visible |
| `guardian_decision` | allow/block/pending/HITL. | Visible |
| `approval_id` | HITL approval identity. | Visible |
| `confirm` | Explicit confirmation for restricted operations. | Input-only |
| `action_receipt` | Result from approved execution. | Visible |

## 6. Evidence Fields

| Field | Purpose | Visibility |
| --- | --- | --- |
| `evidence_id` | Evidence record identity. | Visible |
| `event_type` | Product event category. | Visible |
| `actor` | User or system actor. | Visible |
| `role` | Actor role. | Visible |
| `source_id` | Source identity. | Visible |
| `container` | Container scope. | Visible |
| `blob` | Object path where relevant. | Visible |
| `policy_id` | Governing policy. | Visible |
| `summary` | Human-readable result. | Visible |
| `details` | Structured operational detail. | Role-controlled |

## 7. UI Control Rules

- Prefer dropdowns for saved sources, containers, file IDs, policies, jobs, and providers.
- Use text input only when user must create or search a value.
- Hide fields until required by the selected operation.
- Use preview-first defaults for mutating actions.
- Display recommendations and next actions based on current source, object, risk, and approval state.
