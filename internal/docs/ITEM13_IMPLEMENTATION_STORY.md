# Item 13 Implementation Story

## Data Agent Local Resolution & Caching / Agent Core Runtime Slice

**Owner:** Arbind
**Module:** Item 13 — Data Agent Local Resolution & Caching
**Repo:** `data-agent-local-resolution`
**Purpose of this document:** Explain the implementation story in simple language so anyone reviewing the repo can understand what we built, why we built it, how the scope evolved, and how each file/folder connects to the MVS architecture.

---

# 1. Why this document exists

During the initial implementation, Item 13 was understood mainly as:

> Data Agent should resolve compiled policy artifacts locally and cache them.

So the first implementation focused on:

```text
Lease
  → local Git compiled policy lookup
  → SHA-256 hash verification
  → Ed25519 signature verification
  → policy activation
  → local cache / last-known-good fallback
```

After reviewing the complete MVS wiki pages, the Item 13 responsibility became broader.

Item 13 is not only a policy resolver. It is the **Agent Core runtime orchestration module** inside the Data Agent.

So this repo is now evolving from:

```text
Policy Resolver PoC
```

to:

```text
Agent Core Runtime Slice PoC
```

The earlier policy resolver work is still valid. It now becomes one internal part of the larger Agent Core runtime.

---

# 2. Updated understanding of Item 13

The updated scope of Item 13 is:

> Agent Core receives a prepared execution state and lease, resolves the correct compiled policy and preconfigured workflow locally, enforces the processing tier, orchestrates the batch pipeline, gates actions through Guardian, generates Batch Summary, and records provenance.

In simple words:

```text
Item 13 = local trusted runtime brain of the Data Agent
```

It does not own every capability internally. Instead, it orchestrates other components through clear contracts.

---

# 3. End-to-end MVS story

The full MVS flow is:

```text
1. Policy is authored in Maestro.
2. Policy goes through HITL approval.
3. Conductor compiles approved policy.
4. Compiled policy artifact is signed and stored in the compiled Git repo.
5. Workload Control creates a lease for a data store.
6. Item 9 prepares the execution state.
7. Item 13 receives the lease + execution state.
8. Item 13 resolves policy and workflow locally.
9. Connector produces normalized metadata.
10. Agent Core enforces processing tier.
11. Agent Core calls scoring/sampling, content intelligence, entity aggregation, Guardian, and connector action contracts.
12. Guardian must approve before any write-back action.
13. Connector performs approved action.
14. Agent Core generates Batch Summary.
15. Item 15 consumes Batch Summary and performs roll-up.
16. Item 12 displays Decision Provenance.
```

This means Item 13 sits in the runtime center of the Data Agent.

---

# 4. What we already started earlier

Earlier, we created this repo:

```text
data-agent-local-resolution/
```

The original structure focused on policy resolution:

```text
cmd/
  policy-agent/
    main.go

internal/
  api/
  config/
  policyresolver/

demo/
  compiled-policy-repo/
  leases/
  keys/

data/
  active-policy/
  last-known-good/
  policy-cache/
```

This earlier implementation included or planned:

```text
POST /v1/policy/resolve
GET  /v1/policy/active/{data_store_id}
POST /v1/policy/simulate/git-unavailable
POST /v1/policy/simulate/git-available
```

This part is still useful because Agent Core must first trust the policy before orchestrating anything else.

So we are not deleting this thinking.

We are repositioning it as:

```text
internal/policyresolver/
```

inside the broader Agent Core runtime.

---

# 5. Why we are changing the implementation direction

The full MVS documentation clarified that Item 13 is broader than a cache.

The Agent Core must also:

```text
- consume prepared execution state from Item 9
- resolve preconfigured workflow from local Git
- enforce processing tier
- control cost/depth of work
- coordinate downstream skills
- ensure Guardian-before-action
- emit Batch Summary
- emit provenance
- support graceful degradation
```

So the repo needs to grow.

The new implementation direction is:

```text
Policy resolver remains
+ Execution orchestration is added
+ Batch Summary model is added
+ Provenance model is added
+ Contracts are added
+ Demo inputs are added
```

---

# 6. Current target repo structure

The target repo structure is:

```text
data-agent-local-resolution/
│
├── cmd/
│   └── policy-agent/
│       └── main.go
│
├── internal/
│   ├── api/
│   │   └── server.go
│   │
│   ├── config/
│   │   └── config.go
│   │
│   ├── policyresolver/
│   │   ├── models.go
│   │   ├── resolver.go
│   │   ├── validator.go
│   │   ├── hash.go
│   │   ├── signature.go
│   │   ├── git_resolver.go
│   │   ├── parser.go
│   │   └── cache.go
│   │
│   ├── execution/
│   │   ├── models.go
│   │   ├── manager.go
│   │   └── orchestrator.go
│   │
│   ├── batchsummary/
│   │   └── models.go
│   │
│   └── provenance/
│       └── models.go
│
├── contracts/
│   ├── execution_state.schema.json
│   ├── policy_context.schema.json
│   ├── workflow.schema.json
│   ├── normalized_batch.schema.json
│   ├── sampling_request.schema.json
│   ├── content_intelligence_request.schema.json
│   ├── guardian_decision.schema.json
│   ├── connector_action.schema.json
│   └── batch_summary.schema.json
│
├── demo/
│   ├── execution/
│   │   └── execution-state-standard.json
│   │
│   ├── normalized/
│   │   └── blob-batch-001.json
│   │
│   ├── compiled-policy-repo/
│   │   ├── policies/
│   │   └── workflows/
│   │
│   ├── leases/
│   └── keys/
│
├── data/
│   ├── active-policy/
│   ├── last-known-good/
│   ├── policy-cache/
│   ├── executions/
│   ├── summaries/
│   └── provenance/
│
├── docs/
│   ├── ITEM13_SCOPE_AND_BOUNDARIES.md
│   ├── ITEM13_CONTRACTS.md
│   ├── ITEM13_DEMO_PLAN.md
│   ├── ITEM13_IMPLEMENTATION_STORY.md
│   └── NEXT_IMPLEMENTATION_STEPS.md
│
├── scripts/
├── go.mod
└── README.md
```

---

# 7. Folder-by-folder explanation

## 7.1 `cmd/policy-agent`

This is the application entry point.

The service starts from:

```text
cmd/policy-agent/main.go
```

For now, the binary name is still `policy-agent.exe`.

Later, we may rename it to `agent-core.exe` or `data-agent-core.exe`, but for the hackathon we can keep the current name to avoid unnecessary churn.

---

## 7.2 `internal/api`

This owns the REST API routes.

Current or planned APIs:

```text
GET  /healthz

POST /v1/policy/resolve
GET  /v1/policy/active/{data_store_id}

POST /v1/execution/start
GET  /v1/execution/{execution_id}
GET  /v1/execution/{execution_id}/summary
GET  /v1/execution/{execution_id}/provenance
GET  /v1/cache/status
```

The `/policy` APIs prove local resolution and caching.

The `/execution` APIs prove Agent Core orchestration.

---

## 7.3 `internal/config`

This owns runtime configuration.

Examples:

```text
compiled repo path
policy cache path
active policy path
last-known-good path
public key path
data output paths
```

This allows the same binary to run on different machines without code changes.

---

## 7.4 `internal/policyresolver`

This is the local trust and cache layer.

It owns:

```text
- validate lease policy fields
- find compiled policy artifact in local Git
- verify SHA-256 hash
- verify Ed25519 signature
- parse compiled policy artifact
- activate policy
- cache active policy
- fallback to last-known-good policy
```

This answers:

> How does Agent Core know that the policy is authentic and safe to use?

The answer is:

> It uses `internal/policyresolver`.

---

## 7.5 `internal/execution`

This is the new Agent Core runtime layer.

It owns:

```text
- execution request model
- execution status
- execution manager
- orchestration flow
- processing tier behavior
- skill invocation sequence
```

This answers:

> How does Agent Core run one batch?

The answer is:

> It receives `ExecutionStartRequest`, resolves policy/workflow, loads normalized metadata, applies tier logic, coordinates skill contracts, records provenance, and emits summary.

---

## 7.6 `internal/batchsummary`

This defines the raw Batch Summary generated by Agent Core.

Item 13 creates the raw summary.

Item 15 consumes it for roll-up.

This answers:

> Why does Item 13 have Batch Summary model if Item 15 owns roll-up?

The answer is:

> Item 13 owns batch execution, so it knows what happened in the batch. Item 15 owns long-term aggregation, Elasticsearch storage, ILM, retention, and hierarchy roll-up.

---

## 7.7 `internal/provenance`

This defines decision and execution provenance.

It records:

```text
- which policy was active
- which workflow was active
- which rules were applied
- whether Guardian approved
- whether local execution was confirmed
- whether human review was required
- what action was requested
```

This supports Item 12.

---

## 7.8 `contracts`

This folder contains machine-readable contract schemas.

This supports the manager’s “contracts first” direction.

The purpose is to make integration clear before heavy coding starts.

Other teams can look at these schemas and align their outputs/inputs.

---

## 7.9 `demo`

This folder contains sample inputs for demo.

It allows us to test Agent Core without waiting for every external module to be ready.

Example:

```text
demo/execution/execution-state-standard.json
demo/normalized/blob-batch-001.json
demo/compiled-policy-repo/policies/...
demo/compiled-policy-repo/workflows/...
```

This is not “fake architecture.”

It is a contract-driven demo.

---

## 7.10 `data`

This folder contains runtime-generated outputs.

Examples:

```text
active-policy/
last-known-good/
policy-cache/
executions/
summaries/
provenance/
```

During demo, we can show real generated JSON files here.

---

# 8. Important key names and what they mean

## 8.1 `execution_id`

Unique ID for one Agent Core execution run.

Example:

```text
exec-001
```

This lets us fetch status, summary, and provenance later.

---

## 8.2 `lease_id`

Unique ID for one lease from Workload Control.

A lease means:

> This Data Agent has been assigned this unit of work.

---

## 8.3 `job_id`

Higher-level job ID.

A job may contain multiple leases or batches.

---

## 8.4 `data_store_id`

Identifies the data store being processed.

Example:

```text
prod-blob-finance
```

---

## 8.5 `compiled_policy_version`

The exact compiled policy version that must be used.

Example:

```text
cpv-2026.05.29.001
```

Agent Core must not silently use another policy version.

---

## 8.6 `compiled_git_tag`

Git tag pointing to the compiled policy artifact.

Example:

```text
compiled/cpv-2026.05.29.001
```

This allows deterministic local Git resolution.

---

## 8.7 `compiled_artifact_hash`

SHA-256 hash expected for the compiled policy artifact.

This protects against tampering.

---

## 8.8 `workflow_version`

The workflow version Agent Core should execute.

For MVS, this is a simple linear workflow.

Example:

```text
workflow-mvs-linear-v1
```

---

## 8.9 `processing_tier`

Controls cost/depth.

Expected values:

```text
tier_1
tier_2
tier_3
```

Meaning:

```text
tier_1 = rolled-up/dashboard
tier_2 = standard/signal
tier_3 = full investigation
```

---

## 8.10 `processing_mode`

Human-readable runtime mode.

Expected values:

```text
summary
standard
full_investigation
```

---

## 8.11 `entities_of_interest`

Entities from the active compiled policy.

Example:

```text
PII.Email
PII.NationalID
Financial.BankAccount
```

Content Intelligence and Entity Aggregation should only focus on these entities.

---

## 8.12 `dd_protection_flags`

A compact bitmask for governance signals.

This is a high-signal native tag used by Agent Core, Guardian, Connectors, and customer DLP tools.

---

## 8.13 `decision_id`

Unique ID for a governance decision.

Used by Decision Provenance.

---

# 9. What we changed from the first implementation

## Earlier implementation

```text
Policy resolution only
```

Focused APIs:

```text
POST /v1/policy/resolve
GET  /v1/policy/active/{data_store_id}
```

Focused code:

```text
internal/policyresolver
```

---

## Updated implementation

```text
Agent Core runtime slice
```

Additional APIs:

```text
POST /v1/execution/start
GET  /v1/execution/{execution_id}
GET  /v1/execution/{execution_id}/summary
GET  /v1/execution/{execution_id}/provenance
GET  /v1/cache/status
```

Additional code:

```text
internal/execution
internal/batchsummary
internal/provenance
contracts
demo
docs
```

---

# 10. What we are not building in Item 13

Item 13 does not own everything.

## Not owned by Item 13

```text
- Policy authoring UI
- HITL approval UI
- Conductor compilation service
- Policy hierarchy resolution logic
- Real Blob connector implementation
- Real Content Intelligence engine
- Real Guardian engine
- Real Elasticsearch roll-up logic
- Real Decision Provenance UI panel
```

## What Item 13 owns instead

```text
- consume execution state
- resolve policy and workflow locally
- cache policy/workflow
- enforce processing tier
- orchestrate workflow steps
- call skills through contracts
- enforce Guardian-before-action
- generate raw batch summary
- generate provenance records
- handle degradation and partial success
```

---

# 11. How to explain mocks in the demo

Some components may be mocked for tomorrow:

```text
Guardian
Content Intelligence
Connector write-back
Elasticsearch
DuckDB
```

This is acceptable because the goal is contract-first integration.

The demo is proving:

```text
Agent Core can orchestrate the complete flow correctly.
```

It is not claiming that every downstream module is production-ready.

Correct explanation:

> We are mocking unavailable external modules behind real contracts. This allows us to validate the Agent Core orchestration and integration boundaries without waiting for every module to complete.

---

# 12. Final one-line mental model

Remember this sentence:

> Item 13 receives a lease and execution state, resolves trusted policy and workflow locally, controls how deeply the batch is processed, gates actions through Guardian, and emits batch summary plus provenance.

Everything in this repo supports that story.

---

# 13. Current implementation status

## Done / Started

```text
- Go module initialized
- Basic service created
- Local compiled policy artifact created
- Ed25519 key/sign/verify utility created
- SHA-256 hash verification created
- Policy resolver started
- Initial API started
- Contract docs created
- Contract/demo package created
```

## Changing now

```text
- Expanding from policy resolver to Agent Core runtime slice
- Adding execution models
- Adding batch summary models
- Adding provenance models
- Adding orchestration APIs
```

## Next implementation step

Next file to implement:

```text
internal/execution/orchestrator.go
```

This file will connect:

```text
ExecutionStartRequest
  → policyresolver.ResolveAndActivatePolicy()
  → workflow resolution
  → normalized batch loading
  → tier enforcement
  → mock skill calls
  → Guardian-before-action
  → BatchSummary
  → ProvenanceRecord
```

---

# 14. How to answer team questions

## Q: Did we restart from scratch?

No.

> We reused the valid policy resolver foundation and expanded it into the broader Agent Core runtime scope after the full MVS documents clarified Item 13.

## Q: Why did the scope change?

> The complete wiki showed that Item 13 owns not just local policy cache, but Agent Core orchestration, tier enforcement, workflow resolution, batch summary generation, and graceful degradation.

## Q: Is the earlier implementation wasted?

No.

> The earlier policy resolver is still required. Agent Core cannot execute safely without resolving and verifying the active compiled policy.

## Q: Why are contracts important?

> Contracts allow other module owners to build in parallel without integration chaos.

## Q: Why are some components mocked?

> Because the goal is to prove Agent Core orchestration and contracts. Real modules can replace mocks later without changing the core flow.

## Q: What is the final demo?

> Start an execution, resolve policy/workflow locally, process a normalized batch, enforce tier, simulate downstream calls, require Guardian approval before action, generate summary, and show provenance.



# Update 2 — Consolidated Policy Resolver Package

During implementation, the repo had an import cycle because `internal/policyresolver/resolver.go` imported helper subpackages like `artifact`, while those helper packages imported the parent `policyresolver` package.

To keep the hackathon build simple and stable, we consolidated the policy resolver helper files directly under `internal/policyresolver`.

The active policy resolver package now contains:

- `models.go`
- `resolver.go`
- `validator.go`
- `hash.go`
- `signature.go`
- `git_resolver.go`
- `parser.go`
- `cache.go`

The old subfolder `.go` files were renamed to `.bak` so Go does not compile them.

This keeps the policy resolver as a clean internal trust layer inside Agent Core:
lease → local Git artifact → hash verification → signature verification → parse policy → activate/cache → last-
known-good fallback.


Why these folders are needed?

| Folder                  | Meaning                                 |
| ----------------------- | --------------------------------------- |
| `internal/execution`    | Agent Core runtime execution flow       |
| `internal/batchsummary` | Raw Batch Summary emitted by Agent Core |
| `internal/provenance`   | Decision/execution provenance records   |
| `data/executions`       | Runtime execution status output         |
| `data/summaries`        | Batch summary output                    |
| `data/provenance`       | Provenance output                       |

I separated the Agent Core responsibilities into runtime execution, batch summary, and provenance because they integrate with different MVS items. Execution is Item 13, Batch Summary feeds Item 15, and Provenance feeds Item 12.

What this file means

This file is the core runtime vocabulary for Item 13.

ExecutionStartRequest

This is the request body for the future API:

POST /v1/execution/start

It contains:

Key	Meaning
execution_id	Unique ID for this Agent Core execution run
lease	Work assignment from Workload Control
execution_state	Prepared runtime state from Item 9
batch_input_path	Path to normalized batch JSON file

Simple explanation:

This is how we start one Agent Core batch execution.

ExecutionLease

This represents the lease coming from Workload Control.

Important fields:

Key	Meaning
lease_id	Unique lease/work assignment
job_id	Parent job
data_store_id	Data store being processed
compiled_policy_version	Exact compiled policy version to use
compiled_git_tag	Git tag for local policy resolution
compiled_git_commit	Git commit fallback reference
compiled_artifact_hash	SHA-256 expected hash
workflow_version	Workflow version to run
workflow_git_tag	Git tag for workflow definition

Team explanation:

The lease tells Agent Core what policy and workflow it must execute for this data store.

ExecutionState

This comes from Item 9.

Important fields:

Key	Meaning
processing_tier	Tier 1, Tier 2, or Tier 3
processing_mode	summary, standard, or full_investigation
settings	Runtime switches
operational_config	Extensible future config

Team explanation:

Item 9 prepares the state. Item 13 consumes it and enforces it during runtime.

NormalizedFileRecord

This is what Agent Core consumes from Connector / Quick Scan.

It hides protocol-specific details.

Blob, NFS, SMB, S3, and SharePoint should eventually all normalize to this shape.

Team explanation:

Agent Core should not know Blob SDK, SMB ACL internals, or NFS traversal details. It consumes normalized metadata.





batchsummary/models.go

What this file means

This defines the raw batch summary that Agent Core emits.

Item 13 owns this because it runs the batch.

Item 15 consumes it because Item 15 owns roll-up.

Team explanation:

Agent Core creates the raw batch-level summary because it knows what happened in that batch. Item 15 then rolls it up into datastore, tenant, or higher-level aggregates.



gofmt -w internal\provenance\models.go

What this file means

This record answers:

Why did Agent Core take or plan an action?
Which policy was active?
Which workflow was active?
Did Guardian approve?
Was HITL required?
Was execution local?

Team explanation:

Provenance makes the decision defensible and auditable.


gofmt -w internal\execution\manager.go
What this file means

This is a simple in-memory store for the demo.

It stores by execution_id:

status
summary
provenance

Later, this can become SQLite or a durable local store.

Team explanation:

For the demo, in-memory is enough to prove orchestration. The contract remains stable, so storage can later be replaced with SQLite or DuckDB-backed persistence.


# Update 3 — Added Agent Core Runtime Models

After consolidating the policy resolver, we added the first Agent Core runtime model files:

- `internal/execution/models.go`
- `internal/execution/manager.go`
- `internal/batchsummary/models.go`
- `internal/provenance/models.go`

These files define the runtime language of Item 13.

`execution/models.go` defines the execution request, lease, execution state, normalized batch, and execution status. This connects Item 9, Workload Control, Connector/Quick Scan, and Agent Core.

`batchsummary/models.go` defines the raw Batch Summary that Agent Core emits after each batch. Item 15 consumes this for roll-up and long-term storage.

`provenance/models.go` defines the provenance records needed to explain which policy/workflow was active, why a decision happened, whether Guardian approved, and whether local execution was confirmed. This supports Item 12.

`execution/manager.go` provides a simple in-memory store for demo execution status, summary, and provenance records.

This step moves the repo further from a policy-resolution-only PoC toward the broader Agent Core runtime slice.



What you can say in the daily demo after this step

I fixed the Go package structure and removed the import cycle. Then I added the first Agent Core runtime models for Item 13: execution request, lease, execution state, normalized batch record, execution status, batch summary, and provenance. This keeps the implementation contract-first and prepares the repo for the actual orchestrator flow.




# Update 4 — Added Agent Core Demo Orchestration Flow

We added `internal/execution/orchestrator.go`, which connects the main Item 13 runtime flow.

The demo flow now supports:

1. `POST /v1/execution/start`
2. Policy resolution through the existing policy resolver
3. Workflow resolution placeholder for MVS linear workflow
4. Normalized batch loading from demo JSON
5. Processing tier behavior selection
6. Guardian-before-action simulation
7. Raw Batch Summary generation
8. Provenance generation
9. Execution status lookup
10. Summary and provenance lookup APIs

This makes the repo demo-ready as an Agent Core runtime slice, while still keeping external components like Guardian, Content Intelligence, DuckDB, Elasticsearch, and real connectors behind contract-shaped mocks for now.





# Update 6 — Added Real Workflow JSON Resolution and Removed UI Hardcoding

We improved the demo flow by adding real workflow JSON resolution.

Earlier, Agent Core only checked whether `workflow_version` was non-empty. Now it loads the workflow definition from:

`demo/compiled-policy-repo/workflows/workflow-mvs-linear-v1.json`

The workflow definition contains the MVS linear steps:
- resolve compiled policy
- resolve workflow
- load normalized batch
- apply processing tier
- sampling and scoring
- content intelligence
- entity aggregation
- Guardian evaluation
- connector write-back
- batch summary
- provenance

We added `internal/execution/workflow.go` to resolve and validate the workflow definition.

We also removed hardcoded execution request details from the browser UI. The UI now calls:

`GET /v1/demo/default-request`

This endpoint loads:

`demo/execution/execution-standard.json`

and sends that request to:

`POST /v1/execution/start`

This means policy version, hash, workflow version, and batch input path are no longer hardcoded in the UI. If the demo request changes, we only update the JSON file.