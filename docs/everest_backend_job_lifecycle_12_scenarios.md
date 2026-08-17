# Everest Backend Job Lifecycle: 12 End-To-End Scenarios

## Purpose

This document explains how Everest backend work should be initiated, converted into jobs or executions, routed through the Data Agent Cell components, governed by Guardian, and closed with evidence, audit, and provenance.

It is written for developers, architects, QA engineers, and technical product owners who need a shared understanding of how Everest should behave as an enterprise product.

The key product invariant is:

```text
Portal / Intent
  -> Data Agent Cell Orchestrator
  -> Connector / Discovery / Analysis
  -> Guardian Decision
  -> Approved Action Execution or HITL Task
  -> Provenance + Evidence + Dashboard Summary
```

No user screen, AI tool, connector, workbench action, or backend shortcut should mutate customer data outside this path.

## Current Implementation Map

The current codebase already contains several parts of the enterprise architecture. Some are complete vertical slices; some are contract-shaped placeholders that should be hardened in the next refactor.

| Area | Current Files | Current Role |
|---|---|---|
| API gateway / portal backend | `internal/api/server.go` | Registers HTTP routes and connects Portal requests to backend services |
| Azure Blob source configuration | `internal/api/server.go`, `internal/connectors/azureblob/config.go`, `internal/secretstore/secretstore.go` | Saves simplified source configuration, stores credentials securely, hides raw secrets from Portal |
| Async Azure Blob scan jobs | `internal/api/azureblob_async_scan_handlers.go`, `internal/api/azureblob_async_scan_worker.go`, `internal/controlplane/azureblob_scan_jobs.go` | Queues jobs, manages state, leases, checkpoints, retry, pause, cancel, completion |
| Execution orchestration | `internal/execution/orchestrator.go`, `internal/execution/models.go` | Starts execution, resolves policy/workflow, loads normalized batch, builds summary/provenance |
| Normalized batch contract | `internal/contracts/normalized.go`, `internal/contracts/normalized_batch.schema.json` | Common file/object record model consumed by Orchestrator |
| DuckDB/local working store | `internal/localstore/duckdbstore/*` | Stores normalized metadata, manifest rows, action plan views, Guardian decision views |
| Azure Blob actions | `internal/api/azureblob_actions_handlers.go`, `internal/connectors/azureblob/actions.go` | Plans, blocks, or executes approved Azure Blob actions |
| DSPM/content intelligence | `internal/api/dspm_handlers.go`, `internal/dspm/engine.go` | Tiered scan, keyword/entity checks, model profile selection, risk output |
| AI recommendations | `internal/api/ai_advisor_handlers.go`, `internal/ai/*` | Generates recommendations and optional HITL submissions; does not mutate data |
| HITL approvals | `internal/platform/approval_store.go`, `internal/platform/approval_handlers.go` | Creates, lists, approves, rejects, expires approval requests |
| Provenance | `internal/provenance/models.go` | Records policy activation, Guardian/action gates, actor, rules, reasoning, timestamps |
| Evidence and audit | `internal/api/portal_records*.go`, `internal/portalstate/store.go` | Records user-visible evidence, action history, audit events, notifications |

## Common Backend Objects

### 1. Source Record

Represents a saved data source such as Azure Blob.

Important fields:

- `source_id`
- `data_store_id`
- `connector_id`
- `storage_account`
- `container`
- `prefix`
- `action_mode`
- `writeback_enabled`
- `credential_set`

Source records are visible to users. Raw secrets are not.

### 2. Credential Secret

The backend may build an Azure Blob connection string from simplified fields or accept a secure connection artifact from an admin flow. The Portal should not keep showing the raw secret after save.

The source points to a secret reference:

```text
source_id
  -> secretstore key
  -> connector runtime uses secret at execution time
```

### 3. Request Envelope

The target architecture should standardize every operation into an envelope before backend execution.

Recommended fields:

```text
request_id
tenant_id
actor
role
intent
source_id
data_store_id
connector_id
operation
scope
object_refs
policy_context
runtime_mode
correlation_id
requested_at
```

Today, not every API route explicitly creates this envelope. The Phase 1 refactor should make this mandatory.

### 4. Async Scan Job

Defined by `AzureBlobScanJob`.

Important fields:

- `job_id`
- `source_id`
- `data_store_id`
- `batch_id`
- `requested_by`
- `mode`
- `status`
- `dry_run`
- `limit`
- `scopes`
- `checkpoint`
- `attempts`
- `retry_count`
- `throttle_count`
- `records_written`
- `normalized_path`
- `lease_owner`
- `lease_expires_at`
- `heartbeat_at`
- `last_error`
- `cancel_requested`
- `created_at`
- `updated_at`
- `completed_at`

Scan job states:

```text
queued
  -> running
  -> paused
  -> running
  -> completed

running
  -> failed

running
  -> cancelled
```

### 5. Execution

Defined by `ExecutionStartRequest`, `ExecutionLease`, `ExecutionState`, and `ExecutionStatus`.

Execution is the Orchestrator-level unit that resolves policy, workflow, normalized batch, summary, and provenance.

Important fields:

- `execution_id`
- `lease_id`
- `job_id`
- `data_store_id`
- `batch_id`
- `compiled_policy_version`
- `compiled_artifact_hash`
- `workflow_version`
- `processing_tier`
- `processing_mode`
- `enable_content_intelligence`
- `enable_guardian_evaluation`
- `enable_write_back_actions`
- `guardian_before_action_required`
- `action_mode`

Execution states:

```text
RUNNING
  -> COMPLETED
  -> FAILED
```

### 6. Normalized Batch

Defined by `NormalizedBatch` and `NormalizedFileRecord`.

This is the connector-neutral object/file view used by Orchestrator, DuckDB, Guardian, AI, and Evidence Center.

Important fields:

- `batch_id`
- `lease_id`
- `data_store_id`
- `file_id`
- `path`
- `name`
- `size_bytes`
- `content_type`
- `source_system`
- `permission_class`
- `permission_risk_score`
- `dd_protection_flags`
- `scan_timestamp`

### 7. Guardian Decision

Defined by `guardian_decision.schema.json` today and should be expanded to include policy bundle hash/version as mandatory enterprise fields.

Required enterprise fields:

- `decision_id`
- `policy_version`
- `policy_bundle_hash`
- `approved`
- `decision`
- `approved_actions`
- `rules_applied`
- `reasoning`
- `human_review_required`
- `obligations`
- `local_execution_confirmed`

### 8. HITL Approval

Defined by `ApprovalRequest`.

Important states:

```text
PENDING
  -> APPROVED
  -> REJECTED
  -> EXPIRED
```

HITL is not a separate product path. It is a controlled branch from the same Orchestrator/Guardian flow.

### 9. Action Receipt

The action receipt is returned after approved execution.

Important fields:

- `execution_id`
- `decision_id`
- `file_id`
- `operation`
- `status`
- `connector_response`
- `completed_at`

Action states:

```text
planned
blocked
completed
failed
```

### 10. Provenance Record

Defined by `ProvenanceRecord`.

Important fields:

- `decision_id`
- `execution_id`
- `lease_id`
- `data_store_id`
- `file_id`
- `decision_type`
- `source_policy_version`
- `compiled_policy_version`
- `workflow_version`
- `processing_tier`
- `rules_applied`
- `reasoning`
- `guardian_approved`
- `human_review_required`
- `local_execution_confirmed`
- `action_requested`
- `actor`
- `timestamp`

## Shared Component Flow

All scenarios should eventually conform to this flow:

```text
User / Schedule / API Client
  -> Portal or API
  -> Authorization Guard
  -> Request Envelope
  -> Data Agent Cell Orchestrator
  -> Job Store / Execution Store
  -> Policy Resolver
  -> Workflow Resolver
  -> Connector Adapter
  -> Zero-Copy Accessor
  -> Normalized Batch
  -> DuckDB Working Store
  -> DSPM / AI Signals
  -> Guardian Decision
  -> Action Executor or HITL Task Creator
  -> Connector Action
  -> Action Receipt
  -> Provenance Writer
  -> Evidence + Audit + Dashboard Summary
```

## Scenario 1: Azure Blob Source Onboarding And Container Discovery

### Customer Goal

A platform admin adds an Azure Blob source using simple business-friendly fields. The user should not be asked to understand backend connection-string terminology.

### Initiation

```text
Portal: Source Inventory / Azure Blob Workbench
  -> user clicks Add Azure Blob
  -> user enters storage account, authentication fields, display name, optional default container/prefix
  -> Portal submits source save request
```

Current route:

```text
POST /v1/connectors/azureblob/config
```

Target route after Orchestrator refactor:

```text
POST /v1/orchestrator/intents
operation = configure_source
connector_id = azureblob
```

### Backend Flow

```text
API Handler
  -> authorizeEnterpriseAction("azureblob.source.save")
  -> validate source fields
  -> build credential material in backend
  -> secretstore.Save(source-specific credential)
  -> azureConfigStore.Update(public non-secret config)
  -> portalState.UpsertAzureSource()
  -> record audit event
  -> return public config with credential_set=true
```

### Job Creation

This scenario does not create a scan job yet. It creates or updates a source record and secret reference.

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Portal | Collect simplified fields, never show saved raw secret |
| API Handler | Validate, authorize, normalize |
| Secret Store | Persist credential material |
| Portal State | Save source display record |
| Connector Adapter | Later uses saved credential |
| Evidence/Audit | Records source save/delete events |

### Success End State

```text
source.status = saved
source.credential_set = true
raw_secret_visible_to_user = false
next_step = discover_containers
```

### Failure End States

```text
missing required fields
  -> HTTP 400
  -> no source saved

unauthorized role
  -> HTTP 403
  -> audit authorization failure

secret store failure
  -> HTTP 500
  -> source config not committed
```

### Developer Notes

- Store credentials per source, not globally only.
- The UI should show `Credential saved`, not the connection string.
- Container names should be discovered through the connector after source save.
- Raw technical payload should only appear in support/debug mode.

## Scenario 2: Async Multi-Container Metadata Discovery Job

### Customer Goal

A data steward selects one or more containers and starts a metadata scan. Everest should show job status, checkpoint progress, selected containers, retry counts, and completion output.

### Initiation

```text
Portal: Scans & Jobs or Azure Blob Workbench
  -> user selects saved source
  -> user selects containers from dropdown
  -> user optionally selects prefixes
  -> user starts scan
```

Current route:

```text
POST /v1/scan/azureblob/jobs
```

Request shape:

```text
source_id
data_store_id
batch_id
requested_by
mode = metadata_scan
scopes = [{ container, prefixes }]
limit
dry_run
```

### Backend Flow

```text
API Handler
  -> authorizeEnterpriseAction("azureblob.async_scan.queue")
  -> verify Azure Blob credential exists for source_id
  -> normalize scopes
  -> AzureBlobScanJobStore.Start()
  -> job.status = queued
  -> record lifecycle evidence: azureblob.async_scan.queued
  -> record audit event
  -> return 202 Accepted with job
```

The worker begins when the job is resumed or started by backend control:

```text
startAzureBlobAsyncWorker(job)
  -> runAzureBlobAsyncWorker(job_id, worker_id)
  -> store.Resume(job_id)
  -> store.AcquireLease(job_id, worker_id)
  -> azureBlobWorkbenchClient(source_id)
  -> record worker_started
  -> for each scope/container/prefix
       -> list Azure Blob page
       -> retry on throttling/transient errors
       -> persist portal blobs
       -> persist normalized page artifact
       -> RecordNormalizedWrite()
       -> UpdateCheckpoint()
       -> record checkpoint evidence
  -> Complete(job_id)
  -> record completed evidence
```

### Job State Sequence

```text
queued
  -> running
  -> running with lease_owner and heartbeat_at
  -> running with checkpoint updates
  -> completed
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Portal | Show saved source, selected containers, selected prefixes, job status |
| API Handler | Queue job and enforce authorization |
| Job Store | Persist job state, checkpoint, retry, lease |
| Worker | Page through Azure Blob objects safely |
| Connector Adapter | List containers/blobs using Azure SDK |
| Portal State | Store discovered containers/blobs for dropdowns |
| Evidence/Audit | Store lifecycle events |
| Normalized Artifact Writer | Write connector-neutral batch JSON |

### Success End State

```text
job.status = completed
job.records_written > 0
job.normalized_path = data/async-normalized/<job_id>.json
evidence includes queued, worker_started, checkpoint, normalized_written, completed
portal dropdowns show discovered containers and blobs
```

### Failure End States

```text
credential missing
  -> HTTP 400
  -> no job queued

container not found
  -> job.status = failed
  -> last_error contains container reason

throttling
  -> retry_count incremented
  -> throttle_count incremented
  -> retry_after set
  -> job continues or fails after max retries

cancel requested
  -> job.status = cancelled
  -> worker stops without continuing pages
```

### Developer Notes

- Multi-container selections live in `job.scopes`.
- The UI should display selected containers from `job.scopes`, not from a hidden local variable.
- Checkpoint state must be visible enough for support and QA.
- This is discovery/normalization, not final governed mutation.

## Scenario 3: Autonomous MVS1 Auto-Tagging For Safe Objects

### Customer Goal

Everest scans a batch, identifies safe low/medium-risk objects, and automatically applies approved metadata or protection flags when policy allows.

### Initiation

```text
Portal: Scans & Jobs
  -> user chooses completed normalized batch
  -> user starts MVS1 autonomous tagging execution
```

Current route:

```text
POST /v1/execution/azureblob/start
```

Target request envelope:

```text
operation = mvs1_auto_tag
source_id = selected Azure Blob source
batch_id = completed normalized batch
processing_tier = tier_2 or tier_3
action_mode = apply
writeback_enabled = true
guardian_before_action_required = true
```

### Backend Flow

```text
API Handler
  -> build ExecutionStartRequest
  -> create execution_id, lease_id, job_id
  -> Orchestrator.Start()
      -> SaveStatus(RUNNING)
      -> policyResolver.ResolveAndActivatePolicy()
      -> workflowResolver.Resolve()
      -> load normalized batch from DuckDB or batch artifact
      -> determine tier behavior
      -> build batch summary
      -> build provenance records
      -> SaveManifest(summary/details)
      -> SaveStatus(COMPLETED)
      -> SaveSummary()
      -> SaveProvenance()
```

For enterprise target behavior, after summary/action plan generation:

```text
Action Plan Rows
  -> Orchestrator sends Guardian decision request per object or per policy-safe batch group
  -> Guardian returns allow with obligations
  -> Action Executor calls Azure Blob action endpoint
  -> Connector applies metadata/index tags/dd_protection_flags
  -> Action receipt persisted
  -> Provenance links decision_id to action receipt
```

### Job / Execution State Sequence

```text
execution.status = RUNNING
  -> policy_resolved = true
  -> workflow_resolved = true
  -> records_processed = N
  -> guardian_approved = true for allowed set
  -> write_back_requested = true
  -> COMPLETED
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Orchestrator | Owns execution and action routing |
| Policy Resolver | Verifies compiled policy version/hash |
| Workflow Resolver | Loads workflow steps |
| DuckDB Store | Provides normalized batch and manifest views |
| Guardian | Allows safe auto-tagging only when policy permits |
| Action Executor | Applies approved metadata/index tags |
| Azure Blob Connector | Performs writeback |
| Provenance Writer | Records decision and action evidence |

### Success End State

```text
execution.status = COMPLETED
manifest_summary exists
manifest_details include action_requested
guardian_decisions view shows allow decisions
writeback_actions include completed rows
evidence shows actions taken
dashboard shows tagged count
```

### Failure / Block End States

```text
policy resolution failed
  -> execution.status = FAILED
  -> no action executor call

Guardian returns deny
  -> action.status = blocked
  -> no connector writeback

writeback disabled
  -> action.status = planned
  -> dashboard shows preview only

approval missing
  -> action.status = blocked
  -> reason = Approved HITL request is required
```

### Developer Notes

- Auto-tagging must not mean "skip Guardian".
- For MVS1, safe metadata/tag updates can become autonomous only when Guardian explicitly allows them.
- `dd_protection_flags` should be treated as a high-signal native tag shared by Orchestrator, Guardian, Connector, and customer tools.

## Scenario 4: Sensitive Payroll Or PII Object Requires HITL

### Customer Goal

Everest detects sensitive payroll, PII, or regulated content and routes risky remediation to a human approver.

### Initiation

```text
Portal or scheduled scan
  -> content scan or MVS execution identifies high-risk object
```

Current route for direct content scan:

```text
POST /v1/dspm/content-scan
```

Target Orchestrator path:

```text
MVS1 execution
  -> tiered content intelligence
  -> risk scoring
  -> Guardian decision
```

### Backend Flow

```text
Orchestrator
  -> Connector provides object reference
  -> Zero-Copy Accessor performs bounded preview/range read
  -> DSPM Scan evaluates:
       - keyword patterns
       - entities
       - model profile
       - classification
       - confidence
       - risk score
  -> normalized signals written to DuckDB
  -> Guardian decision request includes risk/classification/evidence
  -> Guardian returns review_required or allow_with_obligations
  -> Orchestrator creates HITL ApprovalRequest
  -> Evidence records dspm.content_scan.completed
  -> Audit records dry-run scan event
```

### Job / Task State Sequence

```text
execution = RUNNING
  -> object classified high/critical
  -> guardian.decision = review_required
  -> approval.status = PENDING
  -> execution = COMPLETED_WITH_HITL or COMPLETED
```

The current implementation may represent this through evidence/action history rather than a formal `COMPLETED_WITH_HITL` status. The target architecture should make this state explicit.

### Component Responsibilities

| Component | Responsibility |
|---|---|
| DSPM Engine | Extracts entities, keywords, classification, risk |
| AI Layer | May explain findings but cannot act |
| Guardian | Decides review is required |
| Platform Approval Store | Creates HITL task |
| Portal Action Center | Shows pending approval |
| Evidence Center | Shows why the file is sensitive |

### Success End State

```text
approval.status = PENDING
action.status = needs_approval or guardian_review
evidence includes classification, risk_score, requires_hitl=true
no mutation occurs yet
```

### Follow-Up End States

```text
approver approves
  -> approval.status = APPROVED
  -> Orchestrator re-validates Guardian decision if policy requires
  -> action may execute

approver rejects
  -> approval.status = REJECTED
  -> action remains blocked

approval expires
  -> approval.status = EXPIRED
  -> action remains blocked
```

### Developer Notes

- HITL tasks should include enough evidence for a reviewer to decide without exposing excessive raw data.
- High-risk evidence should remain behind Guardian and HITL.
- AI can prefill the task reason, but the approval decision belongs to authorized users and Guardian policy.

## Scenario 5: AI Recommendation To HITL Submission To Approved Tagging

### Customer Goal

AI recommends remediation based on current evidence. If configured, AI can prepare HITL tasks, but it cannot mutate data.

### Initiation

```text
Portal: Action Center / AI recommendation
  -> user selects execution
  -> user asks for recommendations
  -> optional: submit HITL approvals
```

Current route:

```text
POST /v1/ai/azureblob/recommendations
```

### Backend Flow

```text
API Handler
  -> validate execution_id
  -> resolve AI provider config and catalog manifest
  -> store.ListActionPlan(execution_id)
  -> build recommendations from action plan rows
  -> if AutoSubmitApprovals=true:
       -> verify provider allows HITL submission
       -> create ApprovalRequest for each recommendation needing HITL
  -> persist evidence: ai.recommendation.created
  -> persist audit event
  -> upsert action records as recommended / needs_approval / guardian_review
```

After approval:

```text
Reviewer approves HITL
  -> approval.status = APPROVED
  -> user or autonomous controller invokes action execution
  -> Azure Blob action handler verifies:
       action_mode = apply
       writeback_enabled = true
       guardian_decision = allow
       approved HITL exists for execution_id and operation
       restricted confirmation if required
  -> connector executes
  -> action receipt + evidence + audit
```

### Job / Task State Sequence

```text
recommendation.status = needs_approval
  -> approval.status = PENDING
  -> approval.status = APPROVED
  -> action.status = completed
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| AI Provider | Recommend/explain using evidence |
| AI Advisor Handler | Enforce provider automation policy |
| Platform Approval Store | Create HITL task |
| Guardian | Must still allow executable action |
| Action Executor | Runs only after gates pass |
| Evidence/Audit | Records recommendation and outcome |

### Success End State

```text
AI recommendation exists
HITL approval exists if needed
approved action completed
provenance links recommendation -> approval -> Guardian decision -> action receipt
```

### Block End States

```text
provider is recommend-only
  -> approval_submission.provider_policy = blocked
  -> no HITL task auto-created

Guardian decision is pending/review/deny
  -> action not executed

approval not found
  -> action.status = blocked
```

### Developer Notes

- Never let AI set `writeback_enabled=true`.
- Never let AI create an action receipt directly.
- AI-generated recommendations should be evidence-grounded and auditable.

## Scenario 6: Guardian Denies Delete Request

### Customer Goal

A destructive delete request is blocked unless strict policy, approval, and confirmation rules allow it.

### Initiation

```text
Portal: Action Center / Workbench advanced action
  -> user selects object
  -> operation = delete
  -> user previews or attempts apply
```

Current route:

```text
POST /v1/connectors/azureblob/actions/execute
```

### Backend Flow

```text
Action Handler
  -> validate operation is supported
  -> validate file_id or blob_path exists
  -> if preview_only or action_mode != apply:
       -> status = planned
       -> no connector call
  -> if action_mode=apply:
       -> check writeback_enabled
       -> check guardian_decision == allow
       -> check confirmation text DELETE
       -> check approved HITL for execution_id and delete operation
       -> only then call connector
```

For denied Guardian decision:

```text
Guardian decision = deny
  -> CanApplyAzureBlobPortalAction() returns false
  -> response.status = blocked
  -> persist action execution evidence
  -> record audit event
  -> no Azure Blob scanner.ExecuteAction()
```

### Job / Action State Sequence

```text
action.requested
  -> guardian.decision = deny
  -> action.status = blocked
  -> evidence.status = blocked
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Portal | Shows preview and blocked reason |
| Action Handler | Enforces all gates before connector call |
| Guardian | Denies destructive request |
| Connector | Must not receive delete when denied |
| Evidence/Audit | Records blocked attempt |

### Success End State

For this scenario, success means safe block:

```text
Azure Blob object remains unchanged
action.status = blocked
reason explains Guardian/apply/approval requirement
evidence severity = high
audit event exists
```

### Developer Notes

- A denied delete is not an error; it is a governed outcome.
- Blocked attempts must still be visible in Evidence Center.
- Confirmation text alone is not enough. Guardian and approval are mandatory.

## Scenario 7: Restricted Archive, Rehydrate, Quarantine, Or Migration

### Customer Goal

A user wants to perform a restricted but non-delete operation, such as archive, retrieve/rehydrate, quarantine, or migration.

### Initiation

```text
Portal: Action Center or Workbench
  -> operation = archive | retrieve | rehydrate | quarantine | migration
  -> user starts with Preview
  -> user requests approval
```

Current route:

```text
POST /v1/connectors/azureblob/actions/execute
```

### Backend Flow

```text
Preview Request
  -> action_mode = dry_run or preview_only = true
  -> response.status = planned
  -> reason = no Azure Blob modification
  -> evidence/action history record created

Apply Request
  -> action_mode = apply
  -> writeback_enabled = true
  -> guardian_decision = allow
  -> confirmation = APPLY
  -> approved HITL verified from Control Plane
  -> connector ExecuteAction()
  -> action receipt persisted
  -> manifest action status updated
```

### Job / Action State Sequence

```text
planned
  -> needs_approval
  -> approved
  -> completed
```

or:

```text
planned
  -> blocked
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Portal | Promotes preview-first interaction |
| Guardian | Allows, denies, modifies, or requires HITL |
| Approval Store | Confirms authorized human approval |
| Action Handler | Requires `APPLY` confirmation |
| Azure Blob Connector | Executes only after gates pass |
| Provenance | Links decision and action receipt |

### Success End State

```text
action.status = completed
connector_response includes operation result
manifest_details.action_status = completed
evidence event = action.<operation>.completed
audit DryRun=false
```

### Block End States

```text
missing APPLY confirmation
  -> action.status = blocked

missing approved HITL
  -> action.status = blocked

Guardian decision not allow
  -> action.status = blocked
```

### Developer Notes

- Restricted operations may be supported by the UI, but they should not become casual one-click mutations.
- If the connector cannot guarantee the action safely, Guardian should return deny or HITL.
- The final architecture should centralize these gates in Orchestrator, with the current handler acting as a temporary enforcement layer.

## Scenario 8: Full Content Export Or Download For Investigation

### Customer Goal

An authorized user needs a full content export for investigation, legal review, or compliance evidence.

### Initiation

```text
Portal: Evidence Center / Workbench advanced export
  -> user selects object or evidence package
  -> user requests full export
```

### Backend Flow

Target enterprise flow:

```text
Portal
  -> Request Envelope(operation = export_content)
  -> Orchestrator validates scope and actor
  -> Guardian evaluates export policy
  -> if deny:
       -> block and record provenance
  -> if HITL:
       -> create approval task
  -> if allow:
       -> Connector streams content by reference
       -> export artifact is created under controlled retention
       -> action receipt records export
       -> evidence/audit/provenance updated
```

### Zero-Copy Boundary

Default zero-copy behavior allows:

```text
metadata
object reference
range read
bounded preview
ephemeral feature extraction
```

Full content export is outside the default path and requires explicit governance:

```text
full duplicate content
  -> Guardian approval
  -> audit
  -> retention policy
  -> access control
  -> provenance
```

### Job / Action State Sequence

```text
export.requested
  -> guardian.decision = allow | deny | review_required
  -> approval.status = PENDING if required
  -> export.status = completed | blocked
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Orchestrator | Owns export request and decision chain |
| Guardian | Enforces export policy |
| Connector | Streams only approved content |
| Evidence Store | Records export evidence |
| Audit Store | Records actor, time, target, purpose |

### Success End State

```text
export completed under controlled retention
provenance includes reason and policy hash
evidence package references export action
dashboard shows export count/risk
```

### Developer Notes

- Do not treat export like ordinary scan.
- Export should have stronger approvals than metadata scan.
- Raw export data should not be indexed into dashboard by default.

## Scenario 9: Batch Failure, Throttling, Pause, Resume, And Recovery

### Customer Goal

A scan is interrupted by throttling, transient network failure, worker crash, or operator pause. Everest should resume from checkpoint without duplicate uncontrolled action.

### Initiation

```text
Async scan is running
  -> Azure returns throttling or transient failure
  -> worker crashes or heartbeat expires
  -> operator pauses/resumes
```

Current routes:

```text
POST /v1/scan/azureblob/jobs/{job_id}/pause
POST /v1/scan/azureblob/jobs/{job_id}/resume
POST /v1/scan/azureblob/jobs/{job_id}/cancel
POST /v1/scan/azureblob/jobs/{job_id}/fail
POST /v1/scan/azureblob/jobs/reclaim-stale
```

### Backend Flow

```text
Worker
  -> AcquireLease(job_id, worker_id)
  -> HeartbeatLease()
  -> list page
  -> if retryable Azure error:
       -> RecordRetry(reason, retry_after, throttled)
       -> exponential backoff
       -> retry page
  -> after successful page:
       -> persist page
       -> write normalized records
       -> UpdateCheckpoint()
```

If worker heartbeat expires:

```text
Admin or scheduled maintenance
  -> ReclaimStaleLeases()
  -> job.status = paused
  -> lease fields cleared
  -> last_error = heartbeat expired
  -> operator resumes
  -> new worker starts from checkpoint
```

### Job State Sequence

```text
running
  -> retry_count incremented
  -> running
```

or:

```text
running
  -> paused
  -> running
  -> completed
```

or:

```text
running
  -> failed
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Job Store | Lease, heartbeat, checkpoint, retry metadata |
| Worker | Retry, backoff, page resume |
| Portal | Show paused/failed/retry status clearly |
| Evidence/Audit | Record failure and recovery lifecycle |
| Orchestrator | Must avoid duplicate action execution after resume |

### Success End State

```text
job.status = completed
checkpoint.objects_scanned = expected count
records_written = unique normalized records
retry_count/throttle_count visible
evidence shows recovery path
```

### Failure End State

```text
job.status = failed
last_error populated
lease released
operator can create new job or resume if state allows
```

### Developer Notes

- The current normalized writer de-duplicates by `file_id`; keep that invariant.
- Any mutation after recovery must still go through Guardian.
- Do not use an external queue to hide state. The no-queue model still allows local durable state, leases, and checkpoints.

## Scenario 10: Policy Bundle Update, Cache, And Air-Gapped Execution

### Customer Goal

Security publishes a new policy. Data Agent Cells enforce the correct signed version even in private or air-gapped environments.

### Initiation

```text
Policy authoring pipeline
  -> compiled policy bundle produced
  -> version/hash/signature available
  -> Data Agent receives lease referencing policy version/hash
```

Current execution fields:

```text
compiled_policy_version
compiled_git_tag
compiled_git_commit
compiled_artifact_hash
workflow_version
workflow_git_tag
```

### Backend Flow

```text
Orchestrator.Start()
  -> build PolicyLease from ExecutionLease
  -> policyResolver.ResolveAndActivatePolicy()
       -> verify artifact hash
       -> verify signature
       -> activate local policy context
       -> optionally use last-known-good policy if allowed
  -> workflowResolver.Resolve()
  -> continue execution only if LocalExecutionReady=true
```

Air-gapped mode:

```text
Local policy cache
  -> local workflow cache
  -> local connector
  -> local AI provider optional
  -> local DuckDB/state
  -> evidence export generated locally
```

### Execution State Sequence

```text
RUNNING
  -> policy_resolved = true
  -> workflow_resolved = true
  -> COMPLETED
```

or:

```text
RUNNING
  -> FAILED
  -> reason = policy resolution failed
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Policy Resolver | Verify hash/signature/version |
| Workflow Resolver | Resolve workflow version |
| Guardian | Uses active policy bundle for decisions |
| Provenance Writer | Records policy version/hash and rules |
| Portal | Shows active policy and cache health |

### Success End State

```text
active policy context exists
provenance records compiled policy version/hash
decisions are repeatable
air-gapped environment does not require public internet
```

### Developer Notes

- Guardian decisions require policy bundle version and hash.
- Last-known-good fallback should mark execution as degraded.
- Do not allow mutation when policy cannot be resolved unless policy explicitly permits a fail-open mode. Enterprise default should be fail-closed.

## Scenario 11: Customer-Built Connector Uses Same Runtime Contract

### Customer Goal

A customer or partner adds a new connector without Everest redesigning every screen and action path.

### Initiation

```text
Admin registers connector manifest
  -> Connector Catalog validates manifest
  -> runtime endpoint or external process configured
  -> user selects connector in Portal
```

Current routes:

```text
GET  /v1/connectors/catalog
POST /v1/connectors/manifest/validate
POST /v1/connectors/runtime/invoke
```

### Backend Flow

```text
Connector Catalog
  -> load manifest
  -> validate capabilities and deployment modes
  -> show connector in Portal

Runtime Invocation
  -> build connector_runtime_v1 request
  -> include operation, object refs, Guardian decision, dry-run/apply context
  -> call HTTP runtime or external process
  -> receive normalized connector response
  -> record checks/results/details
  -> persist runtime audit
```

For enterprise target:

```text
Orchestrator
  -> routes connector operation through generic adapter
  -> receives normalized objects
  -> uses same DuckDB, Guardian, HITL, action, provenance flow
```

### Job State Sequence

The connector should reuse the same state model:

```text
queued
  -> running
  -> checkpointed
  -> completed | failed | paused | cancelled
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Connector Manifest | Declares capabilities and safety posture |
| Connector Runtime | Performs source-specific discovery/action |
| Orchestrator | Owns lifecycle and governance |
| Guardian | Gates mutation for every connector |
| Portal | Uses generic source/job/object/action views |

### Success End State

```text
connector appears in catalog
contract test passes
normalized objects are produced
Guardian gates actions
evidence/provenance model is unchanged
```

### Developer Notes

- Azure Blob should become one connector implementation, not the product architecture.
- Connector-specific details belong behind the adapter.
- UI should use dynamic capabilities, dropdowns, and suggestions from manifest/options.

## Scenario 12: Evidence Review, Audit Investigation, And Dashboard Closure

### Customer Goal

An auditor or security leader reviews what Everest found, what it decided, what it changed, what it blocked, and why.

### Initiation

```text
Portal: Evidence Center / Enterprise Dashboard
  -> reviewer selects execution_id, job_id, object, policy, or action
```

Current routes:

```text
GET /v1/evidence
GET /v1/evidence/package
GET /v1/audit/events
GET /v1/actions/history
GET /v1/execution/{execution_id}/summary
GET /v1/execution/{execution_id}/provenance
GET /v1/execution/{execution_id}/manifest/summary
GET /v1/execution/{execution_id}/manifest/details
GET /v1/actions/plan
GET /v1/guardian/decisions
```

### Backend Flow

```text
Reviewer Request
  -> API reads portal evidence/audit/action state
  -> Orchestrator manager returns execution status/summary/provenance
  -> DuckDB returns manifest summary/details
  -> Guardian decision views show decision and reasoning
  -> Evidence package composes exportable proof
```

### Evidence Chain

```text
Source save
  -> scan queued
  -> worker started
  -> pages scanned
  -> normalized records written
  -> execution started
  -> policy activated
  -> workflow resolved
  -> Guardian decision
  -> HITL approval if any
  -> action planned/blocked/completed
  -> batch summary
  -> evidence package
```

### Component Responsibilities

| Component | Responsibility |
|---|---|
| Evidence Store | User-facing evidence records |
| Audit Store | Actor/action/time trail |
| Provenance Store | Decision and execution proof |
| DuckDB Manifest Views | Per-object summary and action plan |
| Portal Dashboard | Rollups, next steps, exceptions |
| Export Handler | Controlled evidence package |

### Success End State

```text
reviewer can answer:
  who requested the work?
  which source and objects were involved?
  which policy version/hash was used?
  which rules were applied?
  what did Guardian decide?
  was HITL required?
  what action executed or blocked?
  what changed in the source?
  where is the action receipt?
```

### Developer Notes

- Evidence is not only success records. Blocks, failures, retries, and degraded policy fallback must be recorded.
- Dashboard should show both business outcomes and technical health.
- Evidence package should be understandable without requiring access to raw internal logs.

## Scenario Summary Matrix

| # | Scenario | Backend Unit | Main Terminal State |
|---|---|---|---|
| 1 | Source onboarding | Source record | `saved` |
| 2 | Async discovery | Scan job | `completed` or `failed` |
| 3 | Autonomous auto-tagging | Execution + action | `completed` |
| 4 | Sensitive object HITL | Execution + approval | `PENDING` approval |
| 5 | AI recommendation | Recommendation + approval/action | `recommended`, `pending_approval`, or `completed` |
| 6 | Delete denied | Action | `blocked` |
| 7 | Restricted operation | Action + approval | `planned`, `blocked`, or `completed` |
| 8 | Full export | Export action | `completed` or `blocked` |
| 9 | Recovery | Scan job | `completed`, `paused`, or `failed` |
| 10 | Policy update/airgap | Execution | `COMPLETED` or `FAILED` |
| 11 | Custom connector | Connector job/execution | `completed` or `failed` |
| 12 | Evidence review | Evidence package | `exported` / `viewed` |

## Required Developer Invariants

1. Portal does not mutate data directly.
2. AI does not mutate data directly.
3. Workbench is advanced and still governed.
4. All mutations require Guardian allow.
5. Restricted actions require HITL and confirmation.
6. Full export is not default zero-copy behavior and requires approval.
7. Every execution records policy version/hash.
8. Every action attempt records evidence and audit, even when blocked.
9. Job state must be visible: status, scope, checkpoint, retry, lease, error.
10. Connector output must be normalized before shared analysis.
11. DuckDB/local store is working memory and summary/proof store, not an uncontrolled raw data lake.
12. Future connectors must reuse the same Orchestrator, Guardian, HITL, provenance, and evidence model.

## Target Phase 1 Refactor Checklist

The next architecture-alignment implementation pass should add or harden:

1. A first-class `ActionRequestEnvelope` model.
2. A first-class `GuardianDecision` model with policy bundle hash.
3. A first-class `ActionReceipt` model.
4. Orchestrator-owned routing for scans, reads, AI recommendations, exports, and actions.
5. A generic connector job interface so Azure Blob is not special at orchestration level.
6. Explicit execution terminal states for `COMPLETED_WITH_HITL`, `BLOCKED`, and `DEGRADED`.
7. Enforcement tests proving no mutation can happen without Guardian allow.
8. Evidence tests proving blocks, failures, retries, HITL, and success all produce records.
9. UI fields backed by dynamic dropdown sources from saved state and connector capabilities.
10. Clear separation between user-facing simple labels and technical support payloads.

## Developer Mental Model

The simplest way to remember the design is:

```text
Discover
  -> Normalize
  -> Analyze
  -> Decide
  -> Act or Ask
  -> Prove
```

Detailed backend version:

```text
Intent
  -> Request Envelope
  -> Job / Execution
  -> Lease / Checkpoint
  -> Connector Adapter
  -> Zero-Copy Read
  -> Normalized Batch
  -> DuckDB Signals
  -> DSPM / AI
  -> Guardian
  -> Action Executor or HITL
  -> Receipt
  -> Provenance
  -> Evidence / Dashboard
```

This is the enterprise Everest backend architecture developers should implement against.
