# Everest Data Agent Core: Client Demonstration Script, Code Walkthrough, And Test Plan

## Document Purpose

This document explains what is owned by the Data Agent Core implementation, how the current Go code is structured, what can be demonstrated today, what tests can be executed now, how the current implementation aligns with the target Everest architecture, and what remains in the next development phases.

For browser-only UI testing and client demo execution, use:

- `docs/everest_core_owner_ui_testing_script.md`

It is intended for:

- Client demonstration preparation
- Engineering walkthroughs
- Manager review
- QA planning
- Architecture alignment discussions

## Executive Positioning

Everest Data Agent Core is the runtime layer that sits near customer data, receives governed work, coordinates connector access, normalizes metadata, runs local analysis, enforces policy gates before action, records provenance, and publishes evidence.

The current implementation proves the core vertical slice in Go:

```text
Portal / API Intent
  -> Data Agent Core Go service
  -> Source / Connector access
  -> Normalized batch
  -> Local working store
  -> Policy and workflow resolution
  -> Orchestrator execution
  -> Guardian/action gate contracts
  -> HITL approval integration
  -> Evidence, audit, manifest, and provenance
```

The implementation is not only an Azure Blob screen. It now includes the beginning of a reusable agent core architecture that can support future connectors and AI providers through manifest/runtime contracts.

## What I Own As Data Agent Core

Based on the attached Data Agent Cell and Guardian architecture, the Core ownership is the Data Agent Cell runtime path.

### Owned Directly By Data Agent Core

| Area | What Core Owns | Current Implementation |
|---|---|---|
| Runtime service | Starts and hosts the Data Agent backend and Portal API | `cmd/policy-agent/main.go` |
| Orchestrator | Starts executions, resolves policy/workflow, loads normalized batches, writes summary/provenance | `internal/execution/orchestrator.go` |
| Execution model | Execution request, lease, execution state, status, result | `internal/execution/models.go` |
| Workflow resolution | Loads configured workflow definitions | `internal/execution/workflow.go` |
| Policy resolver integration | Resolves signed compiled policy artifacts and last-known-good fallback | `internal/policyresolver/*` |
| Local working store | Stores normalized files, manifests, action plans, Guardian decision views | `internal/localstore/duckdbstore/*` |
| Connector integration from agent side | Azure Blob first-party connector and generic connector runtime invocation | `internal/connectors/*`, `internal/api/connector_runtime_handlers.go` |
| Azure Blob source operations | Save source, discover containers, scan, workbench operations, actions | `internal/api/*azureblob*`, `internal/connectors/azureblob/*` |
| Async scan lifecycle | Queue, lease, heartbeat, checkpoint, retry, pause, resume, cancel, complete | `internal/controlplane/azureblob_scan_jobs.go`, `internal/api/azureblob_async_scan_worker.go` |
| Zero-copy practical path | Metadata, object references, paging, bounded preview/content scan patterns | Azure Blob scanner, Workbench, DSPM request model |
| DSPM/content intelligence | Tiered metadata/rules/model/deep scan interface, keyword/entity checks, risk scoring | `internal/dspm/engine.go`, `internal/api/dspm_handlers.go` |
| AI runtime boundary | AI provider catalog/runtime invocation; AI recommendations are advisory and HITL-oriented | `internal/ai/*`, `internal/api/ai_*handlers.go` |
| Action gate | Dry-run/preview first, apply only with writeback enabled, Guardian allow, HITL, confirmation | `internal/api/azureblob_actions_handlers.go`, `internal/execution/actions.go` |
| HITL integration | Approval create/list/decide/expire support through Platform API | `internal/platform/*`, `internal/controlplane/client.go` |
| Evidence and audit | User-visible evidence, action history, audit events, portal state | `internal/portalstate/*`, `internal/api/portal_records*.go` |
| Provenance | Execution and decision proof model | `internal/provenance/models.go` |
| Extensibility | Connector manifest and AI provider manifest frameworks | `internal/connectors/catalog`, `internal/connectors/runtime`, `internal/ai/catalog`, `internal/ai/runtime` |
| Developer validation | Go unit tests, portal enterprise regression script | `*_test.go`, `scripts/verify_portal_enterprise.js` |

### Integrated But Not Fully Owned By Core

| Area | Ownership Boundary |
|---|---|
| Policy authoring UI | External governance/product area; Core consumes compiled policy bundles |
| Conductor/compiler | External policy compilation layer; Core verifies and activates compiled artifacts |
| Enterprise Guardian engine internals | Guardian is the policy decision authority; Core must call and enforce Guardian decisions |
| Central policy registry/distribution | External/control plane; Core consumes version/hash/signature |
| Production Elasticsearch/search platform | Core prepares summaries/signals; final production search pipeline remains future integration |
| Customer connector code | Customer/partner-owned; Core validates and invokes through connector runtime contract |
| Customer AI gateway/model | Customer/partner-owned; Core validates and invokes through AI provider contract |

## Current Go Code Structure

### Command Entry Points

| Path | Purpose | Default Port |
|---|---|---|
| `cmd/policy-agent/main.go` | Main Data Agent Core service and Portal backend | `8080` |
| `cmd/platform-api/main.go` | Control Plane / Platform API skeleton for agents, leases, approvals, dashboard, evidence | `9090` |
| `cmd/sample-plugin-runtime/main.go` | Sample customer connector and AI runtime endpoint | `7071` |
| `cmd/azureblob-test-tool/main.go` | Standalone Azure Blob testing utility for real Azure test environments | command-line tool |
| `cmd/policy-tool/main.go` | Policy utility entry point | command-line tool |

### Internal Packages

| Package | Role |
|---|---|
| `internal/api` | HTTP handlers for Data Agent service, Portal APIs, Azure Blob, DSPM, AI, runtime, evidence |
| `internal/execution` | Core orchestrator, execution state, workflow, action constants, manifest generation |
| `internal/controlplane` | Client and models for Control Plane jobs, approvals, leases, scan jobs |
| `internal/connectors/azureblob` | First-party Azure Blob scanner, config, actions, writeback logic |
| `internal/connectors/catalog` | Connector manifest catalog |
| `internal/connectors/runtime` | Generic connector runtime invocation contract |
| `internal/ai/catalog` | AI provider manifest catalog |
| `internal/ai/runtime` | Generic AI provider runtime invocation contract |
| `internal/ai/providers` | Runtime AI provider configuration |
| `internal/dspm` | Content intelligence, keyword packs, entity detection, tiered scan logic |
| `internal/localstore/duckdbstore` | Local working store and views for normalized files, manifests, action plans |
| `internal/platform` | Platform store/API for agents, jobs, approvals, dashboard, audit, evidence |
| `internal/policyresolver` | Compiled policy resolution, signature/hash, cache, last-known-good behavior |
| `internal/portalstate` | Portal state, dropdown/options data, evidence, audit, actions |
| `internal/provenance` | Provenance data model |
| `internal/secretstore` | Secret provider abstraction: local file, environment, Windows Credential Manager contract |
| `internal/contracts` | Shared JSON schemas and normalized data contracts |

### Important Demo And Data Files

| Path | Purpose |
|---|---|
| `demo/execution/execution-standard.json` | Stable execution request for offline-safe core orchestration demonstration |
| `demo/normalized/blob-batch-001.json` | Normalized Azure Blob-like batch used by the stable execution request |
| `demo/compiled-policy-repo/policies/cpv-2026.05.29.001/compiled-policy.json` | Compiled policy artifact used by policy resolver |
| `demo/compiled-policy-repo/workflows/workflow-mvs-linear-v1.json` | Workflow resolved by Orchestrator |
| `demo/connectors/customer-filesystem.connector.json` | Sample customer connector manifest |
| `demo/ai-providers/customer-private-llm.provider.json` | Sample customer AI provider manifest |
| `web/index.html` | Current Portal UI |
| `scripts/verify_portal_enterprise.js` | Portal regression check for enterprise wording and required UI features |

## Current Implementation Capabilities

### Available Today

1. Data Agent Core service on port `8080`.
2. Platform API skeleton on port `9090`.
3. Portal UI served from the Go service.
4. Azure Blob source configuration with credential hiding.
5. Azure Blob container discovery and metadata scan when a real Azure source is configured.
6. Async Azure Blob scan jobs with job state, lease, heartbeat, checkpoint, retry, throttle counters, pause/resume/cancel.
7. Normalized metadata batch loading into local working store.
8. Orchestrator execution using policy resolver, workflow resolver, normalized batch, manifest summary, manifest details, and provenance.
9. DSPM/content intelligence endpoint with:
   - metadata tier
   - rules tier
   - model readiness tier
   - deep tier placeholder
   - keyword packs
   - custom keyword patterns
   - entity detection
   - risk score
   - HITL recommendation flag
10. AI provider catalog and runtime framework.
11. Built-in local policy copilot for evidence-grounded recommendations.
12. Custom/private AI provider manifest support.
13. Connector catalog and runtime framework for customer-built connectors.
14. HITL approval lifecycle through Platform API.
15. Azure Blob action preview/block/execute gate.
16. Evidence, audit, action history, dashboard posture, portal options/dropdowns.
17. Standalone Azure Blob test tool for real Azure environments.

### Current Guardrails

```text
AI cannot directly mutate data.
Portal action apply requires writeback enabled.
Guardian decision must be allow before apply.
Restricted actions require HITL approval.
Restricted actions require confirmation text.
Preview/dry-run does not modify Azure Blob.
Blocked actions still produce evidence and audit records.
```

## Client Demonstration Structure

### Recommended Flow For A Client Meeting

Use this order for the demonstration:

1. Explain Data Agent Core ownership in the architecture.
2. Walk through the Go code structure.
3. Run automated tests.
4. Start the Platform API and Data Agent Core service.
5. Open the Portal.
6. Show dynamic source/job/object/evidence views.
7. Run the stable offline-safe core execution path.
8. Show summary, manifest, action plan, Guardian decision view, and provenance.
9. Run DSPM content scan on sensitive sample data.
10. Show AI recommendation boundary.
11. Show action preview and block behavior.
12. Show HITL approval lifecycle.
13. If Azure is connected, show live Azure source discovery and async scan.
14. Close with alignment and roadmap.

## Demo Talk Track

### Opening

Everest Data Agent Core is the runtime that sits close to enterprise data. It is responsible for receiving governed work, coordinating connector access, normalizing what it sees, running local analysis, enforcing policy decisions before actions, and recording evidence and provenance.

The key design point is that the Portal, AI, and Workbench do not bypass the Core runtime. Everything important flows through the Data Agent Core, and every mutation is expected to be governed before execution.

### Architecture Explanation

In the attached architecture, my area is the Data Agent Cell core:

```text
Request Intake
  -> Context / Policy Input
  -> Workflow and State
  -> Connector Adapter
  -> Zero-Copy Access
  -> Local Working Store
  -> DSPM / AI Signals
  -> Guardian Decision
  -> Action Executor or HITL
  -> Provenance
```

The current Go code has implemented this as separate packages rather than a single monolithic handler. This is important because Azure Blob is only the first connector. The same Core pattern should later support object stores, file shares, databases, SaaS systems, and customer-owned connectors.

## Pre-Demo Setup

Open PowerShell from the repository:

```powershell
cd C:\Users\arkumar\source\repos\data-agent-local-resolution
```

Optional cleanup for a clean run:

```powershell
# Use only when you want a fresh local test state.
# Do not run this if you want to preserve previous evidence/jobs.
Remove-Item -ErrorAction SilentlyContinue data\portal_state.json
Remove-Item -ErrorAction SilentlyContinue data\azureblob_scan_jobs.json
```

For the stable no-Azure path, no Azure credential is required.

For the live Azure path, configure Azure Blob source through the Portal using friendly fields, or use environment variables only for developer-side testing.

## Startup Steps

### Terminal 1: Start Platform API

```powershell
go run ./cmd/platform-api
```

Expected console output:

```text
Everest Platform API starting on :9090
Everest Platform persistent store: data/platform/everest-platform.sqlite
```

### Terminal 2: Start Data Agent Core

```powershell
go run ./cmd/policy-agent
```

Expected console output:

```text
Everest Data Agent service starting on :8080
Control Plane lease polling disabled; set EVEREST_AGENT_POLL_ENABLED=true to enable
```

### Terminal 3: Optional Sample Connector / AI Runtime

```powershell
go run ./cmd/sample-plugin-runtime
```

Expected console output:

```text
Everest sample plugin runtime listening on http://localhost:7071
connector endpoint: http://localhost:7071/connector
AI endpoint: http://localhost:7071/ai
```

Open Portal:

```text
http://localhost:8080
```

## Automated Tests To Run Right Now

### Test A1: Full Go Test Suite

Command:

```powershell
go test ./...
```

Expected result:

```text
ok everest.local/data-agent-policy-resolver/internal/api
ok everest.local/data-agent-policy-resolver/internal/connectors/azureblob
ok everest.local/data-agent-policy-resolver/internal/platform
ok everest.local/data-agent-policy-resolver/internal/secretstore
all other packages show [no test files]
```

Current verification result:

```text
Passed on 2026-06-18.
```

What this proves:

- Code compiles across all Go packages.
- Azure Blob config unit tests pass.
- Azure Blob action gate tests pass.
- Platform approval lifecycle tests pass.
- Secret store tests pass.

### Test A2: Portal Enterprise Regression

Command:

```powershell
node scripts\verify_portal_enterprise.js
```

Expected result:

```json
{"ok":true,"scripts":1,"checked":"web/index.html"}
```

Current verification result:

```text
Passed on 2026-06-18.
```

What this proves:

- Portal JavaScript parses.
- Required enterprise UI features exist.
- Banned visible demo/local wording is not present in the Portal.
- Required UI elements such as Operations Assistant, Add Azure Blob, notification drawer, selected containers, friendly action mode, and account access key flow exist.

## Unit Test Inventory

### Azure Blob Configuration Tests

File:

```text
internal/connectors/azureblob/config_test.go
```

Test cases:

| Test | What It Proves |
|---|---|
| `TestConnectionStringFromRequestBuildsFromFriendlyFields` | Backend can build Azure connection string from friendly user fields |
| `TestConnectionStringFromRequestPrefersImportedSecret` | Imported secure connection artifact takes precedence |
| `TestSourceCredentialSecretName` | Source-specific secret names are sanitized and stable |

### Azure Blob Action Gate Tests

File:

```text
internal/api/azureblob_actions_handlers_test.go
```

Test cases:

| Test | What It Proves |
|---|---|
| `TestRestrictedActionConfirmation` | Archive requires `APPLY`, delete requires `DELETE`, tagging does not require confirmation |
| `TestNormalizeWorkbenchOperationEnterpriseAliases` | User-friendly operation aliases map to backend operation IDs |
| `TestWorkbenchTargetBlobPath` | Copy/move target path logic works correctly |

### HITL Approval Tests

File:

```text
internal/platform/approval_store_test.go
```

Test cases:

| Test | What It Proves |
|---|---|
| `TestApprovalLifecycle` | Approval can be created, governance evaluated, approved, and listed |

### Secret Store Tests

File:

```text
internal/secretstore/secretstore_test.go
```

Test cases:

| Test | What It Proves |
|---|---|
| `TestLocalFileProviderSaveLoadDelete` | Local file secret provider supports save/load/delete |
| `TestEnvironmentProviderLoad` | Environment provider can load secrets and is read-only |
| `TestUnsupportedProviderReportsContractMode` | Unsupported provider reports unconfigured status safely |
| `TestEnvSecretName` | Secret names are mapped safely to environment variable names |

## Stepwise Backend Demonstration: Stable No-Azure Path

This path is safe for client demonstration because it does not require real Azure credentials. It exercises the Core architecture using the bundled normalized batch, policy, and workflow.

### Test B1: Health Check Platform API

Command:

```powershell
Invoke-RestMethod http://localhost:9090/healthz
```

Expected output:

```text
status  = ok
service = everest-platform-api
version = mvs1-control-plane-skeleton
```

Architecture point:

```text
Platform API is available for approvals, leases, jobs, and evidence exchange.
```

### Test B2: Health Check Data Agent Core

Command:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

Expected output:

```text
status  = ok
service = data-agent-local-resolution
```

Architecture point:

```text
Data Agent Core service is running and ready to receive governed execution requests.
```

### Test B3: Show Default Execution Request

Command:

```powershell
$req = Invoke-RestMethod http://localhost:8080/v1/demo/default-request
$req | ConvertTo-Json -Depth 20
```

Expected highlights:

```text
execution_id = exec-001
lease_id = lease-001
job_id = job-001
data_store_id = blob-finance-prod
compiled_policy_version = cpv-2026.05.29.001
workflow_version = workflow-mvs-linear-v1
batch_id = batch-001
batch_input_path = demo/normalized/blob-batch-001.json
```

Architecture point:

```text
This request represents what the Orchestrator consumes: execution ID, lease, policy version, workflow version, processing tier, and normalized batch input.
```

### Test B4: Start Orchestrator Execution

Command:

```powershell
$result = Invoke-RestMethod `
  -Uri http://localhost:8080/v1/execution/start `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($req | ConvertTo-Json -Depth 20)

$result.status | ConvertTo-Json -Depth 10
```

Expected output:

```text
status = COMPLETED
execution_id = exec-001
lease_id = lease-001
job_id = job-001
data_store_id = blob-finance-prod
records_processed = 3
policy_resolved = true
workflow_resolved = true
guardian_approved = true
write_back_requested = true
```

Architecture point:

```text
Orchestrator resolved policy, resolved workflow, loaded normalized metadata, processed the batch, generated summary, and wrote provenance.
```

### Test B5: View Execution Summary

Command:

```powershell
Invoke-RestMethod http://localhost:8080/v1/execution/exec-001/summary |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
execution_id = exec-001
batch_id = batch-001
file_count = 3
processing_tier = tier_2
compiled_policy_version = cpv-2026.05.29.001
risk_summary contains critical/high/medium/low counts
actions_taken contains tagged / flagged_for_review / writeback_planned
sovereignty.local_execution_confirmed = true
metadata.guardian_evaluated = true
metadata.connector_writeback_gate = guardian_before_action
```

Architecture point:

```text
Core converts raw batch processing into enterprise-readable summary signals.
```

### Test B6: View Provenance

Command:

```powershell
Invoke-RestMethod http://localhost:8080/v1/execution/exec-001/provenance |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
decision_type = policy_activation
decision_type = guardian_before_action
compiled_policy_version = cpv-2026.05.29.001
workflow_version = workflow-mvs-linear-v1
rules_applied includes artifact_hash_verified and signature_verified
rules_applied includes guardian_approval_required_before_writeback
local_execution_confirmed = true
actor = agent-core
```

Architecture point:

```text
Every execution must produce explainable proof: which policy was used, which workflow was active, what rules were applied, and why action gating exists.
```

### Test B7: View Manifest Summary And Details

Commands:

```powershell
Invoke-RestMethod http://localhost:8080/v1/execution/exec-001/manifest/summary |
  ConvertTo-Json -Depth 20

Invoke-RestMethod http://localhost:8080/v1/execution/exec-001/manifest/details |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
manifest summary exists for exec-001
manifest details contain per-file rows
high-risk rows have action_requested and guardian_decision style fields
```

Architecture point:

```text
The manifest is the bridge between backend execution and dashboard/evidence views.
```

### Test B8: View Local Store Tables

Command:

```powershell
Invoke-RestMethod http://localhost:8080/v1/localstore/tables |
  ConvertTo-Json -Depth 20
```

Expected tables:

```text
normalized_files
content_findings
guardian_decisions
writeback_actions
manifest_summary
manifest_details
```

Architecture point:

```text
The local working store acts as transient execution memory and evidence support, not as an uncontrolled raw data copy.
```

### Test B9: View Action Plan

Command:

```powershell
Invoke-RestMethod "http://localhost:8080/v1/actions/plan?execution_id=exec-001" |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
execution_id = exec-001
items include high-risk files requiring action or review
fields include file_id, path, risk_level, classification, action_requested, guardian_decision, provenance_id
```

Architecture point:

```text
The action plan is not direct execution. It is the governed list of possible actions that must pass Guardian/HITL gates.
```

### Test B10: View Guardian Decision Rows

Command:

```powershell
Invoke-RestMethod "http://localhost:8080/v1/guardian/decisions?execution_id=exec-001" |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
items contain decision_id, file_id, path, risk_level, classification, proposed_action, decision, reasoning
reasoning explains Guardian gate requirement
```

Architecture point:

```text
Guardian is represented as a mandatory decision boundary before write-back.
```

## Stepwise DSPM Demonstration

### Test C1: View DSPM Catalog

Command:

```powershell
Invoke-RestMethod http://localhost:8080/v1/dspm/catalog |
  ConvertTo-Json -Depth 20
```

Expected output:

```text
tiers = metadata, rules, model, deep
checks include keyword_pattern, content_email, content_credit_card, content_ssn, content_secret
model_profiles include builtin-rules, local-bert-ner, domain-bert-dspm, cloud-language-service
keyword_packs include enterprise-sensitive, legal-contracts, finance-hr, technology-secrets
```

Architecture point:

```text
Core supports tiered DSPM and future model profiles while staying compatible with air-gapped/private deployments.
```

### Test C2: Run Sensitive Content Scan

Command:

```powershell
$dspmReq = @{
  source_system = "azure_blob"
  connector_id = "azureblob"
  file_id = "azureblob:hulk01:finance/payroll/jun-payroll.csv"
  container = "hulk01"
  path = "finance/payroll/jun-payroll.csv"
  content_type = "text/csv"
  size_bytes = 2048
  scan_tier = "rules"
  model_profile = "builtin-rules"
  keyword_profile = "finance-hr"
  sample_text = "employee salary record jane@example.com SSN 123-45-6789 api_key=abcdef1234567890"
}

$dspm = Invoke-RestMethod `
  -Uri http://localhost:8080/v1/dspm/content-scan `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($dspmReq | ConvertTo-Json -Depth 20)

$dspm | ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status = completed
scan_tier = rules
classification = restricted or confidential
risk_score is high
entities include EMAIL, SSN, SECRET, and/or FINANCIAL_TERM
requires_hitl = true
recommended_metadata is populated
recommended_tags is populated
next_tier may recommend model or deep
```

Architecture point:

```text
Core can perform bounded content intelligence and produce normalized signals without directly mutating the source.
```

## Stepwise AI Demonstration

### Test D1: Built-In Local Policy Copilot

Command:

```powershell
$aiReq = @{
  provider_id = "everest-local-policy-copilot"
  task = "recommend_actions"
  prompt = "Recommend governed actions for high risk evidence"
  prompt_audit = $true
  evidence_grounding = $true
  evidence = @(
    @{
      file_id = "file-001"
      path = "finance/payroll/jan-payroll.xlsx"
      risk_level = "critical"
      classification = "confidential"
    },
    @{
      file_id = "file-002"
      path = "finance/public/vendor-list.csv"
      risk_level = "low"
      classification = "internal"
    }
  )
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/ai/runtime/invoke `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($aiReq | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status = ok
provider_id = everest-local-policy-copilot
contract_version = ai_provider_runtime_v1
controls include autonomous_apply_blocked
recommendations include high-risk action recommendations
requires_hitl = true for risky actions
prompt_audit.evidence_grounded = true
prompt_audit.customer_prompt_sent = false
```

Architecture point:

```text
AI can recommend and explain, but it cannot execute actions. This is aligned with Guardian-first architecture.
```

### Test D2: Optional Customer Private AI Runtime Contract

Requires:

```powershell
go run ./cmd/sample-plugin-runtime
```

Command:

```powershell
$privateAIReq = @{
  provider_id = "customer-private-llm"
  task = "summarization"
  runtime_endpoint = "http://localhost:7071/ai"
  model = "customer-governance-profile"
  prompt_audit = $true
  evidence_grounding = $true
  evidence = @(
    @{ file_id = "file-001"; path = "finance/payroll.xlsx"; risk_level = "high" }
  )
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/ai/runtime/invoke `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($privateAIReq | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status = ok
provider_id = customer-private-llm
runtime_kind = custom_http
contract_version = ai_provider_runtime_v1
checks include contract, prompt_audit, safety
controls include autonomous_apply_blocked and hitl_required_before_actions
```

Architecture point:

```text
Customer-owned AI can plug into Everest through a structured runtime contract without getting direct mutation authority.
```

## Stepwise Connector Runtime Demonstration

### Test E1: Customer Connector Runtime Health

Requires:

```powershell
go run ./cmd/sample-plugin-runtime
```

Command:

```powershell
$connectorReq = @{
  connector_id = "customer-filesystem"
  operation = "scan_metadata"
  runtime_endpoint = "http://localhost:7071/connector"
  data_store_id = "sample-filesystem"
  action_mode = "dry_run"
  preview_only = $true
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/connectors/runtime/invoke `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($connectorReq | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status = ok
connector_id = customer-filesystem
operation = scan_metadata
contract_version = connector_runtime_v1
dry_run = true
safe_mode = true
records include sample normalized object records
checks include contract and safety
```

Architecture point:

```text
Future connectors can be customer-owned while still using Everest's normalized object model, safety gates, and audit pattern.
```

### Test E2: Connector Apply Block Without Guardian/HITL

Command:

```powershell
$blockedConnectorReq = @{
  connector_id = "customer-filesystem"
  operation = "execute_action"
  runtime_endpoint = "http://localhost:7071/connector"
  data_store_id = "sample-filesystem"
  action_mode = "apply"
  preview_only = $false
  guardian_decision = "pending"
  hitl_approved = $false
  action = @{ action_id = "quarantine" }
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/connectors/runtime/invoke `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($blockedConnectorReq | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status = blocked
error or checks explain Guardian allow is required
applied = false
```

Architecture point:

```text
The connector runtime contract enforces Guardian/HITL requirements before customer connector actions can apply.
```

## Stepwise Azure Blob Action Gate Demonstration Without Azure Mutation

These tests do not require real Azure access because they stop before the connector write path.

### Test F1: Preview Restricted Action

Command:

```powershell
$previewAction = @{
  execution_id = "exec-001"
  decision_id = "dec-preview-001"
  file_id = "azureblob:hulk01:finance/payroll/jan-payroll.xlsx"
  blob_path = "finance/payroll/jan-payroll.xlsx"
  operation = "archive"
  action_mode = "dry_run"
  writeback_enabled = $false
  guardian_decision = "pending"
  preview_only = $true
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/connectors/azureblob/actions/execute `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($previewAction | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected output:

```text
status = planned
reason = Dry-run action plan created. Azure Blob was not modified.
guardian_enforcement = required_before_apply
approval_enforcement = not_required_for_preview
```

Architecture point:

```text
Preview creates a governed plan and evidence record. It does not modify customer data.
```

### Test F2: Delete Apply Is Blocked Without Required Gates

Command:

```powershell
$deleteAction = @{
  execution_id = "exec-001"
  decision_id = "dec-delete-001"
  file_id = "azureblob:hulk01:finance/payroll/jan-payroll.xlsx"
  blob_path = "finance/payroll/jan-payroll.xlsx"
  operation = "delete"
  action_mode = "apply"
  writeback_enabled = $true
  guardian_decision = "deny"
  confirm = "APPLY"
  preview_only = $false
}

Invoke-RestMethod `
  -Uri http://localhost:8080/v1/connectors/azureblob/actions/execute `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($deleteAction | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected output:

```text
status = blocked
reason explains apply requires writeback_enabled=true and guardian_decision=allow
guardian_enforcement = required_before_apply
Azure Blob is not modified
```

Architecture point:

```text
Destructive actions are blocked unless Guardian allow, HITL approval, and DELETE confirmation are present.
```

## Stepwise HITL Approval Demonstration

### Test G1: Create Approval

Command:

```powershell
$approvalReq = @{
  tenant_id = "tenant-local-001"
  requested_by = "client-demo-user"
  role = "data_steward"
  operation = "migration"
  action_type = "migration"
  data_store_id = "blob-finance-prod"
  execution_id = "exec-001"
  source_region = "IN"
  target_region = "AE"
  contract_owner = "owner@example.com"
  reason = "Client demonstration approval for governed migration."
}

$approval = Invoke-RestMethod `
  -Uri http://localhost:9090/v1/approvals `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($approvalReq | ConvertTo-Json -Depth 20)

$approval | ConvertTo-Json -Depth 20
```

Expected highlights:

```text
status wrapper = created or saved response depending handler path
approval.status = PENDING
approval.operation = migration
approval.governance_decision may be block due cross-region policy
required_approvers includes contract_owner/security_officer style roles
```

Architecture point:

```text
HITL tasks are formal backend objects, not informal UI comments.
```

### Test G2: Approve HITL Task

Command:

```powershell
$approvalId = $approval.approval.approval_id
if (-not $approvalId) { $approvalId = $approval.approval_id }

$decisionReq = @{
  actor = "security@example.com"
  role = "security_officer"
  decision = "approve"
  comment = "Approved for demonstration."
}

Invoke-RestMethod `
  -Uri "http://localhost:9090/v1/approvals/$approvalId/decision" `
  -Method Post `
  -ContentType 'application/json' `
  -Body ($decisionReq | ConvertTo-Json -Depth 20) |
  ConvertTo-Json -Depth 20
```

Expected highlights:

```text
approval.status = APPROVED
decision actor and role are recorded
audit event approval_decided exists
```

Architecture point:

```text
Action execution can later verify approved HITL before applying restricted operations.
```

## Optional Live Azure Blob Demonstration

Use this only when the test environment has valid Azure Blob access.

### Live Azure Path

```text
Portal
  -> Azure Blob Workbench
  -> Add Azure Blob
  -> enter friendly account/access fields
  -> save source
  -> Discover Containers
  -> select one or more containers
  -> queue async scan
  -> monitor job status
  -> view discovered objects
  -> run DSPM/content scan
  -> preview actions
  -> route restricted action to HITL
```

Expected outputs:

```text
Saved source appears in Saved Azure Blob sources.
Credential status shows saved/available, not raw secret.
Container dropdown is populated.
Selected containers are visible in Workbench status/selection summary.
Async scan job shows queued/running/completed.
Checkpoint shows container, prefix, objects scanned, pages scanned.
Evidence Center shows scan lifecycle and content scan records.
Action Center shows planned/blocked/approval-required actions.
```

### Developer-Only Azure Blob Test Tool

The tool can run these operations against a real Azure Blob endpoint:

```text
list-containers
list
view/get
create
update
download
set-metadata
set-tags
delete
```

Example:

```powershell
go run ./cmd/azureblob-test-tool --op list-containers
go run ./cmd/azureblob-test-tool --op list --container <container-name> --limit 25
go run ./cmd/azureblob-test-tool --op view --container <container-name> --blob <blob-path> --max-bytes 4096
```

Expected output:

```text
JSON response with operation, status, auth_mode, container/blob details, count, items, or preview.
```

Safety note:

```text
Use Portal/Workbench for client-facing operations.
Use this command-line tool only for developer validation in connected test environments.
```

## Current Alignment With Target Architecture

| Target Architecture Principle | Current Alignment | Gap / Next Work |
|---|---|---|
| Data Agent Cell is colocated with data | Implemented as Go service with local store and connector access | Deployment packaging and service hardening needed |
| Orchestrator-first | `internal/execution/orchestrator.go` owns execution path | Not every Portal/API operation is routed through a first-class request envelope yet |
| No queues/bus | Async scan uses local job store, leases, checkpoints, heartbeat | Need generic job model beyond Azure Blob-specific scan jobs |
| Zero-copy by default | Metadata scan, object refs, bounded content scan patterns exist | Formal zero-copy accessor abstraction still needed |
| Connector adapters | Azure Blob first-party connector plus connector runtime contract | Refactor Azure Blob behind generic connector job interface |
| Local state | Portal state, scan job state, local working store implemented | Need stronger retention/cleanup policies and production DuckDB integration naming |
| Policy resolution | Compiled policy version/hash/signature path implemented | Guardian decision contract must include policy bundle hash everywhere |
| Guardian before action | Action gates enforce Guardian allow before apply | Need real Guardian service integration, not only contract/simulated decision fields |
| AI advisory only | Built-in and custom AI runtime block autonomous apply | Need deeper evidence grounding and provider-specific validation |
| HITL | Platform approval lifecycle exists | Need tighter Orchestrator-owned HITL state and UI state transitions |
| Provenance | Execution provenance model exists | Need tamper-evident/signature chain and per-action provenance expansion |
| Evidence Center | Evidence/audit/action history exists | Need final evidence package polish and search/index integration |
| Future connectors | Manifest/runtime framework exists | Need full connector SDK, contract tests, and production plugin loading |

## What Is Production-Ready Enough To Show

1. Go backend service structure.
2. End-to-end Orchestrator execution on normalized batch.
3. Policy/workflow resolution path.
4. Manifest summary/details.
5. Provenance generation.
6. DSPM tiered scan and keyword/entity detection.
7. AI advisory contract and safety boundary.
8. Connector runtime contract for customer-owned connectors.
9. HITL approval lifecycle.
10. Azure Blob action safety gates.
11. Async scan job lifecycle design and implementation.
12. Portal enterprise regression and dynamic options.

## What Should Be Presented As In Progress

1. Real Guardian service integration.
2. Universal `ActionRequestEnvelope` across every operation.
3. Universal connector job model.
4. Full zero-copy accessor abstraction.
5. Durable production-grade DuckDB/Elasticsearch pipeline.
6. Tamper-evident provenance signing.
7. Production deployment packaging.
8. Broader automated tests for Orchestrator, async jobs, DSPM, AI, connector runtime, and evidence package.
9. More polished architecture GIF/visual sequence.
10. Real Azure live action tests in customer-like environment.

## Future Development Roadmap

### Phase 1: Architecture Alignment

Goal:

```text
Make every operation enter through Orchestrator with a standard request envelope.
```

Work items:

1. Add first-class `ActionRequestEnvelope`.
2. Add first-class `GuardianDecision` with policy bundle version/hash.
3. Add first-class `ActionReceipt`.
4. Route scans, reads, AI recommendations, exports, and actions through Orchestrator.
5. Replace route-specific action shortcuts with Orchestrator-owned routing.
6. Add enforcement tests proving no mutation bypasses Guardian.

### Phase 2: Generic Connector Core

Goal:

```text
Make Azure Blob one connector implementation behind a generic connector contract.
```

Work items:

1. Define generic connector job lifecycle.
2. Normalize source/container/object concepts into generic data source scope.
3. Move Azure Blob scan lifecycle into reusable job engine.
4. Add customer connector SDK/contract tests.
5. Add support for S3, SMB/NFS, SharePoint/OneDrive, database, and SaaS connector patterns.

### Phase 3: Guardian And Provenance Hardening

Goal:

```text
Guardian becomes the non-bypassable decision authority and provenance owner.
```

Work items:

1. Integrate real Guardian decision service.
2. Include policy hash/version in every decision.
3. Add obligations handling.
4. Record decision chain per object/action.
5. Add tamper-evident provenance hashes/signatures.
6. Add evidence package export suitable for compliance review.

### Phase 4: Autonomous MVS1 Completion

Goal:

```text
Autonomous safe auto-tagging works end to end, with HITL only where required.
```

Work items:

1. Batch grouping for safe auto-tag decisions.
2. Guardian allow/deny/review per object or policy group.
3. Auto-apply approved metadata/index tags.
4. Create HITL tasks for exceptions.
5. Publish aggregated normalized signals to search/dashboard.
6. Add real Azure validation tests.

### Phase 5: Enterprise Hardening

Goal:

```text
Make the product deployable for connected, private-network, and air-gapped customers.
```

Work items:

1. Service installation and configuration profiles.
2. Secret provider hardening.
3. Observability and health checks.
4. Retry/backoff/resume operational runbooks.
5. Role-based access control.
6. Production storage retention policy.
7. Load and scale testing.
8. Security review.

## Closing Message For Client Demonstration

The current Go implementation demonstrates the Data Agent Core foundation:

```text
Discover -> Normalize -> Analyze -> Decide -> Act or Ask -> Prove
```

The current build already shows:

- The runtime service.
- The Orchestrator.
- Policy/workflow resolution.
- Local working store.
- Azure Blob connector path.
- Async scan lifecycle.
- DSPM intelligence.
- AI advisory boundary.
- HITL approvals.
- Action safety gates.
- Evidence and provenance.
- Extensible connector/AI contracts.

The next development focus is to make the architecture invariants explicit in code:

```text
Everything through Orchestrator.
Every mutation through Guardian.
Every decision with provenance.
Every connector through a common contract.
Autonomy only within policy guardrails.
```
