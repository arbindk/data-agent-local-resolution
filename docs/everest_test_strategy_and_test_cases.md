# Everest Test Strategy and Test Cases

Version: Draft 1  
Product: Everest Enterprise Data Operations - Azure Blob Edition

## Purpose

This document provides a QA-ready test strategy for Everest. It covers smoke tests, happy paths, negative cases, boundary cases, integration failures, role-based scenarios, performance checks, recovery tests, and regression coverage.

## Test Scope

In scope:

- Web portal navigation and layout.
- Day/night theme.
- Source configuration.
- Azure Blob Workbench.
- Azure Blob object operations.
- AI action verbs and action canvas.
- DSPM scans, keyword packs, and custom keyword patterns.
- HITL/action handoff.
- Azure Blob connector scan.
- Evidence and local working store.
- Connector catalog and AI tools.
- API validation.

Out of scope for current draft:

- Full production RBAC, because demo/local mode does not yet enforce complete authentication.
- Production database persistence for Workbench history and DSPM results, because this is planned.
- Real BERT/local model inference, because model profiles are currently catalog/contract-level unless runtime is integrated.

## Test Environment

| Item | Requirement |
|---|---|
| OS | Windows development machine or equivalent. |
| Go | Project-compatible Go version. |
| Azure | Storage account and connection string for testing. |
| Container | At least one test container. |
| Blob Permissions | List, read, write, tag, metadata, delete if delete tests are run. |
| Data Agent | `go run ./cmd/policy-agent` |
| Portal URL | `http://localhost:8080` |
| Optional runtime | `go run ./cmd/sample-plugin-runtime` |

## Test Data

Recommended test blobs:

| Blob Path | Content | Purpose |
|---|---|---|
| `everest-test/public/readme.txt` | `Public test object` | Low-risk baseline. |
| `everest-test/hr/payroll-confidential.txt` | `employee jane@example.com salary review` | Keyword and email detection. |
| `everest-test/secrets/token.txt` | `token=abcdef1234567890` | Secret detection. |
| `everest-test/cards/sample.txt` | `4111 1111 1111 1111` | Credit card Luhn detection. |
| `everest-test/india/pan.txt` | `ABCDE1234F` | India PAN detection. |
| `everest-test/custom/project-everest.txt` | `Project Everest acquisition target` | Custom keyword detection. |

## Smoke Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| SMK-001 | Start Data Agent | Runtime | Code present. | Run `go run ./cmd/policy-agent`. | None | Service starts on `:8080`. | Compile error, port error, panic. | P0 |
| SMK-002 | Open Portal | UI | Data Agent running. | Open `http://localhost:8080`. | None | Portal loads. | Blank page, JS error, 404. | P0 |
| SMK-003 | Health API | API | Data Agent running. | Open `/healthz`. | None | JSON status ok. | Non-200, no response. | P0 |
| SMK-004 | Navigation | UI | Portal open. | Click all left nav items. | None | Each screen opens. | Screen missing, overlap, JS error. | P0 |
| SMK-005 | Theme Toggle | UI | Portal open. | Toggle Day/Night. Refresh. | None | Theme changes and persists. | Button missing, theme resets unexpectedly. | P1 |

## Happy Path Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| HP-001 | Save Azure Credential | Source/Workbench | Valid connection string. | Paste connection string, click Save Credential. | Azure connection string | Credential saved, status configured. | Error, credential remains not configured. | P0 |
| HP-002 | Discover Containers | Workbench | Credential saved. | Click Discover Containers. | Existing account | Containers listed and dropdown populated. | Empty list with known containers, Azure error. | P0 |
| HP-003 | List Blobs | Workbench | Container selected. | Click List Blobs. | `everest-test/` prefix | Blob table shown. | No rows for known objects, API error. | P0 |
| HP-004 | Create Blob Dry-Run | Workbench | Container selected. | Choose Create Blob, set path/content, keep dry-run, run. | `everest-test/ui-sample.txt` | Status preview, no Azure write. | Blob created during dry-run. | P0 |
| HP-005 | Create Blob Apply | Workbench | Container selected. | Clear dry-run, run Create Blob. | `everest-test/ui-sample.txt` | Status ok, blob created. | Azure error, object missing. | P0 |
| HP-006 | View Blob | Workbench | Blob exists. | Select View Blob and run. | `everest-test/ui-sample.txt` | Details, metadata/tags, preview visible. | Preview missing, 404. | P0 |
| HP-007 | Set Metadata | Workbench | Blob exists. | Choose Set Metadata, enter metadata, clear dry-run, run. | `owner=qa,reviewed=true` | Metadata updated. | Metadata missing or wrong. | P1 |
| HP-008 | Set Tags | Workbench | Blob exists and tag permission exists. | Choose Set Tags, enter tags, clear dry-run, run. | `everest_test=true` | Tags updated. | Azure permission error or wrong tags. | P1 |
| HP-009 | Download Blob | Workbench | Blob exists. | Choose Download Blob, run, click Download Result. | Existing blob | Browser downloads blob. | No download payload. | P1 |
| HP-010 | Run Azure Scan | Source/Scans | Config saved. | Run scan. | Container/prefix | Records scanned/written. | Records zero unexpectedly, store error. | P0 |

## DSPM Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| DSPM-001 | Metadata Tier Path Keyword | DSPM | Blob path contains payroll. | Run metadata tier scan. | `hr/payroll.txt` | Sensitive path check warning. | No path warning. | P0 |
| DSPM-002 | Rules Tier Email | DSPM | Blob preview has email. | View blob, run rules scan. | `jane@example.com` | EMAIL entity detected. | No entity. | P0 |
| DSPM-003 | Rules Tier Secret | DSPM | Blob preview has token. | Run rules scan. | `token=abcdef1234567890` | SECRET entity, restricted/high risk. | No secret detection. | P0 |
| DSPM-004 | Credit Card Luhn | DSPM | Content has valid card test number. | Run rules scan. | `4111 1111 1111 1111` | CREDIT_CARD entity. | No entity or invalid card accepted incorrectly. | P1 |
| DSPM-005 | India PAN | DSPM | Content has PAN-like value. | Run rules scan. | `ABCDE1234F` | INDIA_PAN entity. | No entity. | P1 |
| DSPM-006 | Built-In Keyword Pack | DSPM | Content/path has confidential term. | Select All built-in packs, run scan. | `confidential salary` | BUSINESS_SENSITIVE/FINANCIAL_TERM keyword entities. | Keyword not detected. | P0 |
| DSPM-007 | Legal Contracts Pack | DSPM | Legal phrase present. | Select Legal contracts, run scan. | `legal hold` | LEGAL_TERM critical entity. | No keyword entity. | P1 |
| DSPM-008 | Custom Phrase Keyword | DSPM | Custom phrase present. | Add `Project Everest|high|CUSTOM_ENTITY|phrase`, run scan. | `Project Everest` | CUSTOM_ENTITY keyword detected. | Custom pattern ignored. | P0 |
| DSPM-009 | Custom Regex Keyword | DSPM | Custom regex present. | Add `customer-[0-9]{5}|medium|CUSTOM_ENTITY|regex`, run scan. | `customer-12345` | CUSTOM_ENTITY keyword detected. | Regex not matched. | P1 |
| DSPM-010 | Keyword Exclusion | DSPM | Exclusion matches context. | Add keyword and exclusion `sample`, run scan on sample content. | `sample Project Everest` | Keyword suppressed if exclusion applies. | False positive remains. | P2 |
| DSPM-011 | Model Profile Readiness | DSPM | Select local-bert-ner. | Run model tier. | Any sample | model_profile_ready pending_runtime. | No readiness check. | P1 |
| DSPM-012 | Use DSPM Tags | Workbench/DSPM | DSPM result exists. | Click Use DSPM Tags. | DSPM output | Metadata/tags copied to form. | Fields not populated. | P0 |
| DSPM-013 | HITL Required For Restricted | DSPM/HITL | Secret/high-risk content. | Run scan, submit HITL. | token content | HITL handoff or Action Center prefill. | No HITL indication. | P0 |

## AI Verb And Action Canvas Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| AI-001 | Recommend Tags | Workbench AI | Blob selected. | Click Recommend Tags. | Any blob | Recommendation and action canvas shown. | No result, JS error. | P0 |
| AI-002 | Complete Metadata | Workbench AI | Blob selected. | Click Complete Metadata. | Any blob | Metadata artifact shown. | Missing canvas. | P1 |
| AI-003 | Prepare Archive | Workbench AI | Blob selected. | Click Prepare Archive. | Any blob | HITL required. | Unsafe no-HITL recommendation. | P0 |
| AI-004 | Prepare Quarantine | Workbench AI | High-risk blob selected. | Click Prepare Quarantine. | confidential blob | HITL required. | No HITL. | P0 |
| AI-005 | Delegate Assessment | Workbench AI | Container selected with blobs. | Click Delegate Assessment. | Container with mixed blobs | Prioritized recommendations table. | Empty for risky test data. | P1 |
| AI-006 | Apply Canvas To Form | Workbench AI | Canvas populated. | Edit canvas, click Apply Canvas To Form. | Metadata/tags | Workbench fields updated. | Fields unchanged. | P1 |
| AI-007 | Safe Preview | Workbench AI/Actions | Recommendation for tags. | Click Preview / Apply Safe with dry-run. | apply_blob_index_tags | Preview response, no mutation. | Azure mutation. | P0 |
| AI-008 | Safe Apply | Workbench AI/Actions | Safe recommendation and permissions. | Select auto-apply safe, clear dry-run, run. | tags | Tags applied if allowed. | Blocked unexpectedly or unsafe apply. | P1 |
| AI-009 | Risky HITL | Workbench AI/HITL | Risky action selected. | Submit HITL. | quarantine/delete | HITL submitted or Action Center prefilled. | Silent failure. | P0 |

## Negative Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| NEG-001 | Missing Credential | Workbench | No saved credential. | Try list containers. | None | Clear error. | Unclear failure or crash. | P0 |
| NEG-002 | Invalid Credential | Source | Bad string. | Save/test connection. | `bad` | Validation/Azure error. | App crash. | P0 |
| NEG-003 | Missing Container | Workbench | Credential exists. | Run List Blobs without container. | None | Error: container required. | Panic or empty misleading result. | P0 |
| NEG-004 | Missing Blob Path | Workbench | Container exists. | Run View Blob with no blob. | None | Error: blob path required. | Panic. | P0 |
| NEG-005 | Create Existing Without Overwrite | Workbench | Blob exists. | Create same blob, overwrite off. | Existing path | Error asks update/overwrite. | Blob overwritten. | P0 |
| NEG-006 | Delete Without Confirmation | Workbench | Blob exists. | Delete with dry-run off and no confirm. | Existing path | Error requires DELETE. | Blob deleted. | P0 |
| NEG-007 | Tag Permission Missing | Workbench/Azure | SAS/account lacks tag permission. | Set tags. | Any tags | Azure permission error surfaced. | Silent success. | P1 |
| NEG-008 | Invalid Custom Regex | DSPM | Workbench open. | Add invalid regex, run scan. | `[` | No crash, safe handling. | Service crash. | P1 |
| NEG-009 | AI Provider Down | AI Tools | Endpoint unavailable. | Test provider/invoke. | Bad endpoint | Error or fallback. | UI crash. | P1 |
| NEG-010 | Runtime Endpoint Down | Connector Catalog | Sample runtime stopped. | Invoke runtime. | localhost:7071 | Runtime unavailable message. | UI crash. | P1 |

## Boundary Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| BND-001 | Empty Container | Workbench | Empty test container. | List blobs. | Empty container | Empty state. | Error unless Azure returns error. | P2 |
| BND-002 | Large Blob Preview | Workbench | Large blob exists. | View blob with 4096 max bytes. | Large file | Preview truncated, UI responsive. | Full file loaded slowly. | P1 |
| BND-003 | Zero Byte Blob | Workbench | Empty blob exists. | View/DSPM scan. | 0 bytes | Properties shown, no entities. | Crash or incorrect content. | P2 |
| BND-004 | Many Blobs | Workbench | Container has many blobs. | List with limit 5000. | Large container | UI remains usable. | Browser freeze. | P1 |
| BND-005 | Long Blob Path | Workbench | Long path blob exists. | List/view. | Deep path | Text wraps, no overlap. | Layout breaks. | P2 |
| BND-006 | Many Keyword Patterns | DSPM | Add many custom patterns. | Run scan. | 100 patterns | Scan completes reasonably. | Timeout/freeze. | P2 |
| BND-007 | Mixed Binary Content | Workbench/DSPM | Binary blob exists. | View and scan. | PDF/image | Base64 preview or no text, no crash. | Invalid text rendering crash. | P2 |

## Integration Failure Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Test Data | Expected Result | Failure Indicators | Priority |
|---|---|---|---|---|---|---|---|---|
| INT-001 | Azure Network Failure | Azure Connector | Block network/private endpoint. | List containers. | Valid credential | Clear network error. | Infinite loading. | P0 |
| INT-002 | Azure 404 Container | Azure Connector | Credential valid. | List blobs in nonexistent container. | bad-container | Azure 404 surfaced. | Misleading success. | P1 |
| INT-003 | Azure 403 Write | Workbench | Read-only credential. | Create/set tags. | Read-only SAS | Permission error. | Silent success. | P0 |
| INT-004 | Control Plane Down | Workbench/HITL | Platform not running. | Submit HITL. | Risky action | Fallback or clear warning. | UI crash. | P1 |
| INT-005 | AI Runtime Down | AI Tools/Workbench | AI endpoint down. | Ask AI. | Any blob | Local fallback or clear error. | UI crash. | P1 |
| INT-006 | Local Store Error | Scan/Store | Simulate store unavailable. | Run scan. | Valid source | Store error surfaced. | Data loss hidden. | P1 |

## Role-Based Scenario Tests

Current local demo does not enforce production RBAC. These tests define expected enterprise behavior.

| Test ID | Scenario Name | Role | Module | Expected Enterprise Behavior | Priority |
|---|---|---|---|---|---|
| RBAC-001 | Operator lists blobs | Operator | Workbench | Allowed. | P1 |
| RBAC-002 | Operator deletes blob | Operator | Workbench | Blocked or requires approval. | P0 |
| RBAC-003 | Data steward applies tags | Data steward | Action Center | Allowed after Guardian/policy. | P0 |
| RBAC-004 | Auditor applies tags | Auditor | Action Center | Blocked, read-only. | P1 |
| RBAC-005 | Security approves quarantine | Security officer | HITL | Allowed. | P0 |
| RBAC-006 | Admin configures AI provider | Platform admin | AI Tools | Allowed. | P1 |

## Performance And Basic Load Tests

| Test ID | Scenario Name | Module | Steps | Expected Result | Priority |
|---|---|---|---|---|---|
| PERF-001 | List 1000 blobs | Workbench | List prefix with 1000 blobs. | Response and UI within acceptable time. | P1 |
| PERF-002 | Scan 1000 blobs | Azure Scan | Run metadata scan. | Completes without memory spike. | P1 |
| PERF-003 | DSPM 100 samples | DSPM | Run repeated rules scans. | Stable latency, no leaks. | P2 |
| PERF-004 | Keyword 100 patterns | DSPM | Run custom keyword pack. | No timeout. | P2 |
| PERF-005 | Dashboard after scan | Dashboard | Refresh after large scan. | KPIs render quickly. | P2 |

## Recovery Tests

| Test ID | Scenario Name | Module | Preconditions | Test Steps | Expected Result | Priority |
|---|---|---|---|---|---|---|
| REC-001 | Restart Agent | Runtime | Portal open. | Stop/start agent, refresh UI. | UI reconnects after restart. | P1 |
| REC-002 | Retry Azure List | Workbench | Temporary network issue. | Retry list containers. | Works after network returns. | P1 |
| REC-003 | Recover From Failed Create | Workbench | Invalid container. | Fix container and retry. | Create succeeds. | P1 |
| REC-004 | Recover AI Provider | AI Tools | AI down. | Start runtime and retry. | Test/invoke succeeds. | P2 |
| REC-005 | Recover Plugin Runtime | Connector Catalog | Runtime down. | Start sample runtime and refresh. | Runtime ok. | P2 |

## Data Validation Tests

| Test ID | Scenario Name | Module | Steps | Expected Result | Priority |
|---|---|---|---|---|---|
| DATA-001 | Metadata Round Trip | Workbench/Azure | Set metadata, view blob. | Metadata returned exactly. | P0 |
| DATA-002 | Tags Round Trip | Workbench/Azure | Set tags, view blob. | Tags returned exactly. | P0 |
| DATA-003 | DSPM Tags Applied | DSPM/Workbench | Run scan, use tags, set tags, view. | Azure tags match recommendation. | P0 |
| DATA-004 | Normalized Record | Scan/Store | Scan blob, open store. | Record has file_id, path, size, content_type. | P0 |
| DATA-005 | Evidence Link | Evidence | Run execution, open evidence. | Evidence references execution ID. | P1 |
| DATA-006 | Action Result Link | Action Center | Execute preview/apply. | Result has operation/blob/container. | P1 |

## Regression Checklist

Run this after major changes:

| Area | Check |
|---|---|
| UI | All nav items open. |
| UI | Day/night theme works. |
| Source | Credential save and config load work. |
| Workbench | Discover containers works. |
| Workbench | List/view/create/update/download/delete flows work. |
| Workbench | Dry-run default remains on. |
| Workbench | Delete requires `DELETE`. |
| AI | Verb buttons render recommendation/canvas. |
| AI | Risky actions route to HITL/handoff. |
| DSPM | Rules scan detects email/secret/keywords. |
| DSPM | Custom keyword patterns work. |
| Actions | Safe action preview does not mutate. |
| Actions | Apply requires correct policy/Guardian/HITL conditions. |
| Evidence | Evidence screens handle empty and populated state. |
| APIs | Invalid JSON returns 400. |
| Docs | Field/API docs match UI labels and payload names. |

## Exit Criteria For Current Build

The current build is ready for customer demo if:

- Portal starts.
- User can save credential and discover containers.
- User can perform Workbench list/view/create/tag/download/delete dry-run flow.
- DSPM scan returns classification and entities for test content.
- Keyword matching works with built-in and custom patterns.
- AI verbs produce an editable action canvas or fallback recommendation.
- Risky actions are not silently applied.
- Evidence screens do not crash when empty.
- Documentation explains current vs planned capabilities.

## Known QA Gaps

| Gap | Risk | Follow-Up |
|---|---|---|
| No full production RBAC | Cannot certify enterprise security yet. | Add auth/RBAC tests after implementation. |
| Workbench history not persisted | Audit trail incomplete. | Add persistence and tests. |
| DSPM results not persisted | No trend/reporting. | Add store and query tests. |
| BERT profiles not fully integrated | Model tests are readiness-only. | Add runtime integration tests. |
| Screenshots not captured | Training docs less visual. | Add after stable UI validation. |
