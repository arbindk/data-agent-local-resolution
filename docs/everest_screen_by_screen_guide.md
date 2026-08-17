# Everest Screen-by-Screen Guide

Version: Draft 1  
Product: Everest Enterprise Data Operations - Azure Blob Edition

## Purpose

This guide explains every major screen in Everest from a new joiner's point of view. For each screen it explains:

- What the screen does.
- Who uses it.
- Why each field or control exists.
- Which APIs are called.
- Which data is affected.
- Common user mistakes.
- Test scenarios.

## Navigation Overview

Everest has a left navigation menu with these major areas:

| Navigation Item | Main Purpose |
|---|---|
| Enterprise Dashboard | Overall posture and next action. |
| Source Inventory | Configure and validate Azure Blob source. |
| Azure Blob Workbench | Test blob operations, AI verbs, and DSPM scans. |
| Connector Catalog | View built-in, planned, and custom connector capabilities. |
| AI Tools | Configure AI provider runtimes and safety settings. |
| Onboarding | Create source/job/lease setup for governed execution. |
| Scans & Jobs | Start and review scan/job execution. |
| Evidence Center | Review execution evidence and exports. |
| Guardian Agent | Review safety gates and decisions. |
| Action Center | Preview and execute governed Azure Blob actions. |
| Policy Governance | Review policies, context, and artifact trust. |
| Agent Working Store | Inspect normalized local records. |
| Diagnostics | Raw support and debug output. |

## 1. Enterprise Dashboard

### Purpose

The Enterprise Dashboard gives a high-level view of the current environment. It helps users answer:

- Is the agent healthy?
- Is the control plane healthy?
- What is the latest execution?
- Is evidence available?
- What should I do next?

### Primary Users

- Data operations user
- Platform administrator
- Compliance user
- Solution architect
- QA tester

### Navigation Path

Left navigation -> Enterprise Dashboard

### Fields And Controls

| Control | Purpose | Why It Exists | API/Data Used | Validation |
|---|---|---|---|---|
| Refresh Dashboard | Reload dashboard data. | Users need current posture after scans/actions. | Platform dashboard, runtime audit, local store. | Should not fail if no execution exists. |
| Validate Source | Opens Source Inventory. | Guides user to first setup step. | UI navigation only. | Target page should open. |
| Role View Tabs | Shows role-specific summaries. | CIO, CISO, compliance, steward, AI, execution, and audit users need different views. | Dashboard summary, governance context, audit. | Tabs should switch without layout overlap. |
| Latest Execution KPI | Shows latest execution ID. | Evidence and actions depend on execution. | Platform summary. | Empty state should show `None`. |
| Governance Mode KPI | Shows dry-run/apply mode. | Users must know whether actions mutate data. | Config/runtime state. | Must be visible before actions. |
| Evidence Status KPI | Shows if evidence is ready. | Compliance needs audit proof. | Evidence APIs. | Should show pending when no run exists. |

### Common Mistakes

- Expecting object counts before running a scan.
- Not refreshing after execution.
- Confusing dashboard summary with full evidence details.

### Test Scenarios

| Test ID | Scenario | Steps | Expected Result |
|---|---|---|---|
| SCR-DASH-001 | Open dashboard fresh | Start app and open dashboard. | Page loads, no errors, empty values handled. |
| SCR-DASH-002 | Refresh dashboard | Click Refresh Dashboard. | KPIs update or show safe empty state. |
| SCR-DASH-003 | Role tabs | Click each role tab. | Correct section appears without overlap. |

## 2. Source Inventory

### Purpose

Source Inventory is used to configure and validate Azure Blob access before onboarding or scanning.

### Primary Users

- Platform administrator
- Data operations user
- QA tester

### Navigation Path

Left navigation -> Source Inventory

### Fields And Controls

| Field/Control | Purpose | Business Meaning | Mandatory | Example | API/Data Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|
| Connector | Select source type. | Tells Everest which connector runtime to use. | Yes | Azure Blob Storage | UI state, connector catalog. | Wrong connector blocks correct source flow. |
| Deployment Mode | Customer network mode. | Indicates cloud/private/air-gapped/hybrid environment. | Yes | private_network | UI/runtime context. | Wrong mode causes wrong architecture assumptions. |
| Storage Account | Display name for account. | Helps user identify Azure account. | No | hulk01 | UI context. | Low impact unless used for reporting. |
| Containers | Container names. | Defines source scope. | Yes for scan | customer-demo | Azure Blob config. | Wrong container causes empty scan or wrong scan. |
| Prefix / Scope | Blob prefix. | Narrows object scope. | No | hr/payroll | Azure Blob config. | Wrong prefix hides objects. |
| Data Store ID | Everest source ID. | Links policy, execution, evidence. | Yes | azureblob-hulkb01 | Config, execution, policy. | Evidence and policy mapping can break. |
| Credential Status | Shows if credential exists. | Indicates whether agent has secret. | Read-only | configured | Secret store status. | If not configured, Azure calls fail. |
| Source Region | Source geography. | Used for policy/governance. | No for local demo, important for enterprise. | IN | Onboarding/governance. | Wrong region may trigger wrong policy. |
| Connection String | Azure Storage connection string. | Secret used by agent to access Azure Blob. | Required initially | DefaultEndpointsProtocol=... | Secret store. | Wrong secret fails connection or scans wrong account. |
| Load Saved Config | Loads current config. | Avoids retyping. | No | Button | `GET /v1/connectors/azureblob/config` | If failed, user must enter config. |
| Save Config | Saves config and credential. | Makes source reusable. | Yes for setup | Button | `POST /v1/connectors/azureblob/config` | Invalid config blocks scan. |
| Test Connection | Validates Azure access. | Confirms credential/container/prefix. | Recommended | Button | `POST /v1/connectors/azureblob/test` | Failure means no safe scan. |
| Discover Containers | Lists containers. | Helps user choose container without Azure Portal. | Recommended | Button | `POST /v1/tools/azureblob/workbench` | Missing permissions gives empty/error. |

### APIs Triggered

- `GET /v1/connectors/azureblob/config`
- `POST /v1/connectors/azureblob/config`
- `POST /v1/connectors/azureblob/test`
- `POST /v1/tools/azureblob/workbench` with operation `list_containers`

### Data Affected

- Runtime Azure Blob config.
- Secret store for connection string.
- Shared container dropdown options.

### Common Mistakes

- Pasting the wrong account connection string.
- Entering a prefix that is too narrow.
- Trying to test before saving credential.
- Assuming a container SAS can discover sibling containers.

### Test Scenarios

| Test ID | Scenario | Steps | Expected Result |
|---|---|---|---|
| SCR-SRC-001 | Save credential only | Paste connection string and save. | Credential status configured. |
| SCR-SRC-002 | Discover containers | Click Discover Containers. | Container table/dropdown populated. |
| SCR-SRC-003 | Bad connection string | Save invalid string and test. | Clear error, no UI crash. |
| SCR-SRC-004 | Bad prefix | Use valid container and invalid prefix. | Connection may pass, scan/list returns no objects. |

## 3. Azure Blob Workbench

### Purpose

Azure Blob Workbench is a no-command-line test and operations screen. It lets users create realistic Azure Blob customer scenarios and validate actions from the portal.

### Primary Users

- QA tester
- Data operations user
- Solution architect
- Platform administrator
- Data steward
- Security officer

### Navigation Path

Left navigation -> Azure Blob Workbench

### Main Areas

| Area | Purpose |
|---|---|
| Connection and object form | Select credential, container, blob path, operation, metadata, tags, content, dry-run, overwrite, confirmation. |
| Workbench Status | Shows latest workbench operation state. |
| AI Action Verbs | Provides button-first AI recommendations. |
| Action Canvas | Editable AI/DSPM artifact for metadata, tags, and rationale. |
| DSPM Content Scan | Runs tiered DSPM classification and entity detection. |
| Containers and Objects | Shows discovered containers or listed blobs. |
| Object Details | Shows blob properties, metadata, tags, and preview. |
| Raw Result | Shows raw JSON output for support/QA. |

### Workbench Fields

| Field | Purpose | Mandatory | Valid Values / Format | Example | Impact If Wrong |
|---|---|---|---|---|---|
| Connection String | Optional one-time credential input. | Required if no saved credential exists. | Azure Storage connection string. | DefaultEndpointsProtocol=... | Azure operations fail or wrong account is used. |
| Container | Target container. | Yes for blob operations. | Azure container name. | customer-demo | Wrong container means wrong data scope. |
| Blob Path | Target blob path. | Yes for object operations. | Path-like string. | everest-test/sample.txt | Wrong object may be changed. |
| Operation | Workbench action. | Yes | list_containers, list_blobs, view_blob, create_blob, update_blob, set_metadata, set_tags, download_blob, delete_blob | view_blob | Wrong action can cause unexpected result. |
| Prefix | Limits list scope. | No | Blob prefix. | hr/payroll | Too narrow hides blobs. |
| Limit | List result limit. | No | 1 to 5000. | 100 | Too high may slow UI. |
| Content Type | Blob content type. | No | MIME type. | text/plain | Missing type may reduce downstream classification quality. |
| Metadata | Key-value metadata. | Required for set_metadata. | `key=value,key2=value2` | owner=qa | Replaces metadata set. |
| Index Tags | Azure Blob index tags. | Required for set_tags. | `key=value,key2=value2` | classification=restricted | Replaces index tag set. |
| Preview Bytes | Content preview size. | No | Number. | 4096 | Too large may slow response. |
| Confirm | Confirmation text. | Required for delete. | DELETE | DELETE | Delete blocked if missing. |
| Dry-run | Preview without changing Azure. | No, default on. | checked/unchecked | checked | If unchecked, write may occur. |
| Overwrite | Allows create to overwrite. | No | checked/unchecked | unchecked | If checked, existing object can be replaced. |
| Inline Content | Text content for create/update. | No | Any text. | Hello Everest | Used if no file selected. |
| Upload File | Browser file upload. | No | File input. | sample.pdf | Large files may be slow. |

### AI Action Verb Controls

| Control | Purpose | Why It Exists |
|---|---|---|
| Recommend Tags | AI recommends classification/index tags. | Repeated work should be a button, not a chat prompt. |
| Complete Metadata | AI recommends ownership/review metadata. | Makes safe metadata workflow quick. |
| Prepare Archive | AI prepares governed archive plan. | Archive is sensitive and needs policy context. |
| Prepare Quarantine | AI prepares quarantine plan with HITL. | High-risk objects require review. |
| Delegate Assessment | Reviews listed objects and returns prioritized recommendations. | Multi-step work should run as delegated assessment. |
| Optional Instruction | Side-channel prompt. | Used only for unusual requests or explanation. |
| Candidate Action | Action selected by AI/user. | Maps AI output to Action Center operations. |
| Automation | Controls preview, HITL, or safe apply. | Prevents accidental automation. |
| Ask AI | Calls AI recommendation path. | Generates action recommendation. |
| Preview / Apply Safe | Executes safe action preview or apply. | Keeps safe actions fast but governed. |
| Use Recommendation | Prefills Action Center. | Allows manual review. |
| Submit HITL | Routes risky action to approval/handoff. | Required for risky actions. |

### DSPM Controls

| Control | Purpose | Valid Values |
|---|---|---|
| Scan Tier | Selects scan depth. | metadata, rules, model, deep |
| Model Profile | Selects detection/model strategy. | builtin-rules, local-bert-ner, domain-bert-dspm, cloud-language-service |
| Keyword Pack | Selects built-in keyword pack. | builtin-default, enterprise-sensitive, legal-contracts, finance-hr, technology-secrets, none |
| Keyword Exclusions | Suppresses false positives. | Comma-separated strings. |
| Checks | Optional check list. | content_email, content_secret, content_credit_card, etc. |
| Custom Keyword Patterns | Customer-specific patterns. | `pattern|severity|entity_type|match_mode` |
| Run DSPM Scan | Runs DSPM API. | Button |
| Use DSPM Tags | Copies recommended metadata/tags to form. | Button |

### APIs Triggered

- `POST /v1/connectors/azureblob/config`
- `POST /v1/tools/azureblob/workbench`
- `POST /v1/dspm/content-scan`
- `POST /v1/ai/azureblob/recommendations`
- `POST /v1/connectors/azureblob/actions/execute`
- `POST /v1/approvals` or governance fallback depending platform availability

### Common Mistakes

- Forgetting dry-run is enabled and expecting a real write.
- Clearing dry-run without understanding action impact.
- Attempting delete without `DELETE`.
- Running DSPM before viewing/selecting a blob.
- Using keyword regex incorrectly.
- Expecting `local-bert-ner` to run real inference before model runtime is integrated.

### Test Scenarios

| Test ID | Scenario | Steps | Expected Result |
|---|---|---|---|
| SCR-WB-001 | Discover containers | Save credential, click Discover Containers. | Container table populated. |
| SCR-WB-002 | List blobs | Select container, click List Blobs. | Blob list appears and blob dropdown populated. |
| SCR-WB-003 | Create dry-run | Select Create Blob with dry-run checked. | Status preview, no Azure write. |
| SCR-WB-004 | Create apply | Clear dry-run, create blob. | Status ok and blob exists. |
| SCR-WB-005 | View blob | Select blob, run View Blob. | Properties, preview, metadata, tags shown. |
| SCR-WB-006 | Run DSPM rules | View blob, run DSPM rules scan. | Classification, checks, entities shown. |
| SCR-WB-007 | Custom keyword | Add `Project Everest|high|CUSTOM_ENTITY|phrase`. | Keyword entity appears if text/path matches. |
| SCR-WB-008 | Delete blocked | Delete without `DELETE`. | Error or blocked message. |

## 4. Connector Catalog

### Purpose

Connector Catalog explains available and planned connectors. It shows which connectors are built-in, which are contract-ready, and which can be customer-owned.

### Primary Users

- Solution architect
- Platform administrator
- Connector/plugin developer

### Fields And Controls

| Control | Purpose |
|---|---|
| Connector dropdown/table | Select connector. |
| Runtime operation | Test connector runtime contract. |
| Deployment mode | Understand cloud/private/air-gapped support. |
| Runtime endpoint | Points to sample or custom connector runtime. |
| Action ID | Selects runtime action. |
| Guardian decision | Simulates action governance. |
| Confirmation | Required for apply/delete style actions. |

### APIs Triggered

- `GET /v1/connectors/catalog`
- `POST /v1/connectors/manifest/validate`
- `POST /v1/connectors/runtime/invoke`
- `GET /v1/runtime/audit`

### Common Mistakes

- Assuming every catalog connector is fully implemented.
- Forgetting Azure Blob is the first implemented runtime.
- Not starting sample plugin runtime before invoking sample runtime.

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-CON-001 | Load catalog | Connector list appears. |
| SCR-CON-002 | Select Azure Blob | Runtime/action fields populate. |
| SCR-CON-003 | Invoke sample runtime | Runtime status shows ok if service is running. |

## 5. AI Tools

### Purpose

AI Tools configures customer AI providers and validates AI runtime behavior.

### Primary Users

- Platform administrator
- AI platform owner
- Solution architect
- QA tester

### Fields And Controls

| Field/Control | Purpose | Example |
|---|---|---|
| Provider | Selects AI provider. | customer-private-llm |
| Endpoint URL | Provider runtime endpoint. | http://localhost:7071/ai |
| Model | Model name. | local-bert-ner |
| Deployment Name | Cloud/private deployment ID. | gpt-prod |
| Auth Scheme | Authentication style. | none, api_key, bearer |
| Network Mode | cloud/private/air-gapped. | air_gapped |
| Automation Mode | How much AI can do. | hitl_required |
| Redaction Mode | Controls prompt redaction. | strict |
| Runtime Prompt | Test instruction. | Summarize evidence... |
| Save AI Config | Saves provider config. | Button |
| Test Provider | Tests configured provider. | Button |
| Invoke AI Runtime | Calls runtime. | Button |

### APIs Triggered

- `GET /v1/ai/providers/catalog`
- `GET /v1/ai/providers/config`
- `POST /v1/ai/providers/config`
- `POST /v1/ai/providers/test`
- `POST /v1/ai/runtime/invoke`
- `POST /v1/ai/providers/manifest/validate`

### Common Mistakes

- Choosing cloud model in air-gapped environment.
- Forgetting API key or private gateway URL.
- Expecting autonomous apply when HITL is configured.

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-AI-001 | Load provider catalog | Providers appear. |
| SCR-AI-002 | Save private provider config | Config saved. |
| SCR-AI-003 | Test provider unavailable | Clear error, no UI crash. |
| SCR-AI-004 | Invoke sample runtime | Runtime response appears. |

## 6. Onboarding

### Purpose

Onboarding creates a governed setup for a data source. It prepares source, job, policy, action mode, and write-back settings.

### Primary Users

- Data operations user
- Platform administrator
- Solution architect

### Important Fields

| Field | Purpose |
|---|---|
| Tenant ID | Customer tenant. |
| Owner | Contract/business owner. |
| Data Store ID | Source identity. |
| Storage Account | Azure account context. |
| Container | Source container. |
| Prefix | Source scope. |
| Policy | Policy assigned to source/job. |
| Action Mode | dry_run or apply. |
| Writeback | Enables/disables write-back. |

### APIs Triggered

- `POST /v1/onboarding/azureblob/quickstart`
- `POST /v1/execution/azureblob/start`

### Common Mistakes

- No container selected.
- Action mode set to apply without understanding risk.
- Writeback enabled without approval process.

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-ONB-001 | Onboard one container | Job/lease/source created. |
| SCR-ONB-002 | Missing owner for restricted action | Policy blocks or warns. |

## 7. Scans & Jobs

### Purpose

Scans & Jobs starts and reviews governed scan execution.

### Primary Users

- Data operations user
- QA tester

### Main Controls

| Control | Purpose |
|---|---|
| Start Azure Blob Execution | Starts governed execution. |
| Run Scan | Runs Azure Blob metadata scan. |
| Refresh Jobs | Loads job/execution state. |

### APIs Triggered

- `POST /v1/execution/azureblob/start`
- `POST /v1/connectors/azureblob/scan`
- `GET /v1/execution/{id}`
- `GET /v1/jobs`

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-JOB-001 | Start execution | Execution ID appears. |
| SCR-JOB-002 | Scan configured source | Records written to local store. |
| SCR-JOB-003 | Scan without config | Error shown. |

## 8. Evidence Center

### Purpose

Evidence Center shows proof of execution, manifest details, provenance, and exports.

### Primary Users

- Compliance user
- Auditor
- Security officer
- QA tester

### Controls

| Control | Purpose |
|---|---|
| Refresh Evidence | Loads latest evidence. |
| Executive Export | Downloads executive summary. |
| Manifest CSV | Downloads manifest data. |
| Guardian Export | Downloads Guardian evidence. |
| Compliance Export | Downloads compliance evidence. |
| AI Readiness Export | Downloads AI readiness view. |

### APIs Triggered

- `GET /v1/executions/{id}`
- `GET /v1/manifests/summary`
- `GET /v1/manifests/details`
- `GET /v1/provenance`
- `GET /v1/exports/azureblob/{type}`

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-EVD-001 | Open before scan | Empty state shown. |
| SCR-EVD-002 | Open after scan | Summary and details shown. |
| SCR-EVD-003 | Export manifest | CSV downloads. |

## 9. Guardian Agent

### Purpose

Guardian Agent shows safety decisions and action gate results.

### Primary Users

- Security officer
- Data steward
- Compliance user

### APIs Triggered

- `GET /v1/guardian/decisions`
- `GET /v1/actions/plan`
- `POST /v1/connectors/azureblob/writeback/apply-approved`

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-GRD-001 | Load no execution | Empty state. |
| SCR-GRD-002 | Load after action plan | Decisions visible. |
| SCR-GRD-003 | Apply approved writeback | Approved metadata/tags applied. |

## 10. Action Center

### Purpose

Action Center previews and executes governed Azure Blob actions.

### Primary Users

- Data steward
- Security officer
- Operator

### Fields

| Field | Purpose | Example |
|---|---|---|
| Operation | Action type. | apply_blob_metadata |
| File ID | Normalized file id. | azureblob:container:path.txt |
| Blob Path | Direct blob path fallback. | path.txt |
| Target Container | Used for migration/quarantine. | quarantine |
| Target Prefix | Used for target path. | everest/action-output |
| Target Tier | Archive/cool/hot tier. | archive |
| Metadata | Metadata to apply. | reviewed=true |
| Index Tags | Tags to apply. | classification=restricted |
| Execution ID | Evidence/action context. | exec-azureblob-... |

### APIs Triggered

- `POST /v1/connectors/azureblob/actions/execute`
- `POST /v1/ai/azureblob/recommendations`
- Platform approval APIs.

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-ACT-001 | Preview metadata action | Preview result, no Azure mutation. |
| SCR-ACT-002 | Apply safe metadata | Applies only when allowed. |
| SCR-ACT-003 | Delete without approval | Blocked. |
| SCR-ACT-004 | Migration missing target | Validation error. |

## 11. Policy Governance

### Purpose

Policy Governance helps users understand policy precedence, region rules, HITL requirements, and policy artifact trust.

### Primary Users

- Compliance user
- Security officer
- Solution architect

### APIs Triggered

- `POST /v1/policy/resolve`
- `GET /v1/policy/artifact/trust`
- `GET /v1/governance/context`
- `POST /v1/governance/evaluate`

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-POL-001 | Load policies | Policy list appears. |
| SCR-POL-002 | Evaluate restricted action | HITL requirement returned. |
| SCR-POL-003 | Artifact trust | Trust result visible. |

## 12. Agent Working Store

### Purpose

Agent Working Store shows normalized objects and local execution output.

### Primary Users

- Developer
- QA tester
- Support engineer

### APIs Triggered

- `GET /v1/localstore/tables`
- `GET /v1/localstore/normalized-files`

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-STORE-001 | Open before scan | Empty state. |
| SCR-STORE-002 | Open after scan | Normalized object rows appear. |
| SCR-STORE-003 | Filter by batch | Correct batch rows returned. |

## 13. Diagnostics

### Purpose

Diagnostics is a support-only debug area. It shows raw API payloads and health checks.

### Primary Users

- Developer
- QA tester
- Support engineer

### Controls

| Control | Purpose |
|---|---|
| Check Health | Calls health endpoints. |
| Show Last Payload | Displays most recent raw API payload. |

### APIs Triggered

- `/healthz`
- Other last-called APIs depending user actions.

### Test Scenarios

| Test ID | Scenario | Expected |
|---|---|
| SCR-DIAG-001 | Check health | Health JSON shown. |
| SCR-DIAG-002 | Show last payload | Recent response appears. |

## Cross-Screen Traceability

| Business Step | Screen | API | Data Impact |
|---|---|---|---|
| Save Azure credential | Source Inventory / Workbench | `/v1/connectors/azureblob/config` | Secret store updated. |
| Discover containers | Source Inventory / Workbench | `/v1/tools/azureblob/workbench` | UI dropdown updated. |
| View object | Workbench | `/v1/tools/azureblob/workbench` | Object details shown. |
| Run DSPM | Workbench | `/v1/dspm/content-scan` | Classification and tag recommendations returned. |
| Ask AI verb | Workbench | `/v1/ai/azureblob/recommendations` | Recommendation/canvas produced. |
| Preview action | Workbench / Action Center | `/v1/connectors/azureblob/actions/execute` | Preview response, no mutation. |
| Apply action | Action Center | `/v1/connectors/azureblob/actions/execute` | Azure metadata/tags/tier/copy/delete may change. |
| Review evidence | Evidence Center | Manifest/provenance/export APIs | Evidence visible/exported. |

## Open Screen-Level Gaps

| Gap | Impact | Recommended Action |
|---|---|---|
| Workbench operation history not persisted | Audit trace incomplete for UI test actions. | Add history store and panel. |
| DSPM results not persisted | Cannot trend DSPM posture over time. | Store scan results in local working store. |
| Full RBAC not implemented | Demo users can access all screens. | Add auth/RBAC before production. |
| Screenshots not included | Training docs less visual. | Add screenshots after stable UI validation. |
| Model runtime profiles are cataloged but not fully invoked | BERT/domain model flow is contract-ready, not complete inference. | Integrate local/private model runtime. |
