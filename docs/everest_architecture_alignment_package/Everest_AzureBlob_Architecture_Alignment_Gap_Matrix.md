# Everest Azure Blob — Architecture Alignment and Gap Matrix

## Purpose

This document separates two things clearly:

1. **Team target architecture** — the architecture shown in the shared diagram.
2. **Current Azure Blob implementation** — what is actually implemented in the current working build.

We should not mix them. The target architecture is the blueprint. The current Azure Blob build is the working reference slice.

---

## 1. Executive Summary

The team architecture describes a broader **Data Agent Cell** model with:

- Orchestrator-first execution
- Zero-copy data access
- No queues / no bus
- Guardian Agent enforcement
- Connector adapters
- Zero-copy accessors
- Action executor
- Metadata/classification/telemetry collector
- Provenance writer
- Embedded local state
- Policy/workflow runtime
- Concurrency and resiliency controls

The current Azure Blob implementation is a **working reference slice**, not the full target architecture.

Current implementation covers:

- Control Plane API
- Data Agent API
- Azure Blob configuration/test/discovery
- Azure Blob source onboarding
- Job and lease creation
- Job execution
- Dashboard summary
- Evidence summary
- Guardian Agent summary view
- Policy Governance scenarios
- Item 13 policy artifact trust
- Active policy / last-known-good
- Clean enterprise UI baseline

Main gaps before Azure Blob is complete:

- Object-level manifest details
- Action result endpoint
- Decision Provenance endpoint
- Agent Working Store drilldown
- Guardian Agent per-action decision records
- Write-back evidence verification
- Export package
- Batch/checkpoint/resiliency behavior
- Formal connector contracts for future stores

---

## 2. Architecture Positioning

### 2.1 Team Target Architecture

The shared diagram should be treated as the **target blueprint**.

Target flow:

```text
User / Control Plane
→ Data Agent Cell
→ Orchestrator
→ Connector Adapter
→ Zero-Copy Accessor
→ Metadata / Classification / Telemetry Collector
→ Agent Core / Workflow Runtime
→ Guardian Agent
→ Action Executor
→ Provenance Writer
→ Evidence / Audit
```

### 2.2 Current Azure Blob Build

The current build should be described as:

```text
A working Azure Blob reference implementation that covers the main governed execution loop, but not every part of the target Data Agent Cell architecture yet.
```

Current implemented pattern:

```text
Source Inventory
→ Onboarding
→ Control Plane job/lease
→ Data Agent execution
→ Item 13 policy artifact trust
→ Azure Blob scan
→ Summary evidence
→ Guardian Agent summary
→ Policy Governance
→ Agent Working Store summary
```

---

## 3. Target Architecture Areas from Team Diagram

### UI / Purpose Layer

Supports operations such as:

- Discover / inventory
- Classify / tag / label
- Profile / analyze
- Access / read metadata
- Monitor / audit
- Remediate / write back

### Orchestrator

Single internal control point inside the Data Agent Cell.

Responsibilities:

- Receive request
- Validate policy/workflow context
- Build execution plan
- Coordinate connector calls
- Coordinate Guardian Agent decisions
- Coordinate evidence/provenance output
- Return response

### Connector Adapter

Storage-specific layer.

Examples:

- Azure Blob adapter
- SMB adapter
- SharePoint adapter
- OneDrive adapter
- NFS adapter

The adapter hides storage-specific API details from the rest of the system.

### Zero-Copy Accessor

Reads data in place without copying full files to the Control Plane.

For Azure Blob:

- List blobs
- Read properties
- Read metadata/tags
- Optionally stream sampled content if policy allows
- Avoid sending full file content to Control Plane

### Action Executor

Executes approved actions.

Azure Blob v1 allowed real actions:

- Apply blob metadata
- Apply blob index tags

Blocked or dry-run only:

- Delete
- Archive
- Retrieve
- Quarantine
- Migration

### Metadata / Classification / Telemetry Collector

Collects:

- Object metadata
- Size
- Content type
- Last modified time
- Existing tags
- Classification results
- Scan telemetry
- Batch metrics

### Provenance Writer

Records:

- Request
- Policy
- Rules applied
- Reasoning
- Decision ID
- Guardian Agent decision
- HITL flag
- Final action result
- Evidence timestamp

### Guardian Agent

Final runtime safety gate before mutation.

It answers:

```text
Is this specific action safe to execute on this specific object?
```

Expected behavior:

- Allow safe actions
- Mark review-needed actions
- Block unsafe actions
- Fail closed if unavailable

### Local State

Embedded in the Data Agent Cell.

Stores:

- Policy cache
- Active policy
- Last-known-good policy
- Execution checkpoint
- Normalized objects
- Manifest draft
- Action candidates
- Provenance draft
- Temporary scan state

### Control Plane / External Systems

Control Plane owns:

- Job
- Lease
- Source registration
- Agent registration
- Policy assignment
- Evidence summary
- Audit
- Governance decision records

External systems:

- Azure Blob
- Git policy repository
- Identity/secret provider
- Optional model service
- Optional search/index service

---

## 4. Current Azure Blob Implementation Summary

| Area | Current Implementation |
|---|---|
| UI | Clean enterprise UI baseline |
| Control Plane | Onboarding, reset, backup, dashboard summary, governance context |
| Data Agent | Health, Azure Blob connector, run job, runtime status |
| Azure Blob Connector | Load config, test connection, discover containers, scan path |
| Job/Lease | Created through onboarding quickstart |
| Item 13 | Compiled policy version, Git tag/hash/signature, active policy, LKG |
| Dashboard | Summary-level enterprise view |
| Evidence Center | Summary-level evidence and audit |
| Guardian Agent | Summary-level safety posture |
| Policy Governance | India → UAE, Safe Tagging, policy artifact trust |
| Agent Working Store | Summary-level normalization story |
| Documentation | Customer demo, technical handoff, glossary, Q&A |

---

## 5. Architecture Alignment Matrix

| # | Architecture Area | Team Target Architecture | Current Azure Blob Implementation | Status | Gap / Action |
|---|---|---|---|---|---|
| 1 | UI Purpose | Enterprise operation surface | Clean UI with main pages | Partial | Needs QA polish and full scenarios |
| 2 | Orchestrator | Single internal control point | Run job flow exists, orchestration not formalized | Partial | Define orchestrator contract |
| 3 | Connector Adapter | Store-specific adapter pattern | Azure Blob only | Implemented for one connector | Formalize connector interface |
| 4 | Zero-Copy Access | Read source in place | Metadata-first Azure Blob access | Partial | Clarify sampled-content rules |
| 5 | Action Executor | Executes approved actions | Safe metadata/index tag path partially present | Partial | Complete action result evidence |
| 6 | Metadata Collector | Collect metadata/classification/telemetry | Summary metadata and scan result | Partial | Store normalized object rows |
| 7 | Classification | Rule/model-based classification | Basic classification concept | Partial | Define v1 rule-based scope |
| 8 | Telemetry | Runtime metrics and progress | Basic status | Partial | Add stage timing/batch metrics |
| 9 | Provenance Writer | Full decision chain | Summary-level provenance | Partial | Add provenance endpoint |
| 10 | Guardian Agent | Runtime enforcement | Summary view and policy gating | Partial | Persist per-action decisions |
| 11 | Local State | Embedded execution state | Policy cache/LKG + summary | Partial | Add workstore summary/details |
| 12 | Policy as Code | Git-backed, signed policy | Implemented for compiled policy | Mostly done | Add evidence tie-in and failure tests |
| 13 | Workflow as Code | Dynamic workflow artifact | Workflow tag concept only | Partial | Add workflow artifact trust later |
| 14 | No Queues / No Bus | Orchestrator-first, direct calls | Direct API/control flow | Aligned | Clarify sync/async model |
| 15 | Batch Processing | Large scale batches/checkpoints | Not fully exposed | Missing | Add batch/checkpoint design |
| 16 | Resiliency | Agent death/retry/resume | Partial | Missing | Lease TTL/checkpoint QA scenarios |
| 17 | Rate Limiting | Configurable limits | Not productized | Missing | Add connector limits config |
| 18 | Storage | Local + permanent stores | SQLite/local summary | Partial | Define Postgres/object store/search roles |
| 19 | Data Science / NER | Optional model layer | Not part of v1 | Design decision | Mark optional/future |
| 20 | Sensitive Data | No raw content by default | Metadata-first direction | Partial | Define redaction/content rules |
| 21 | Access Control | Role-based access | Role views only, no RBAC | Partial | Real RBAC next phase |
| 22 | Cross-Platform | Future connector protocol | Not formalized | Missing | Define connector SDK/interface |
| 23 | Sidecar | Optional helper component | Not implemented | Future | Keep optional |
| 24 | Export | Evidence package | Not implemented | Missing | Add JSON export or hide |
| 25 | QA Readiness | End-to-end scenarios | In progress | Partial | Build QA checklist after gaps closed |

---

## 6. Azure Blob Completion Scope

### Must complete for Azure Blob QA

| Priority | Item | Why |
|---|---|---|
| P0 | Stable UI with no broken links | QA must navigate without developer help |
| P0 | Source Inventory end-to-end | Valid source is the entry point |
| P0 | Onboarding source/job/lease | Control Plane must own governed work |
| P0 | Run Job execution | Core Data Agent loop |
| P0 | Item 13 Policy Artifact Trust | Key owned module |
| P0 | Workstore summary and normalized object rows | Foundation for evidence and future connectors |
| P0 | Manifest summary/details | Evidence completeness |
| P0 | Action results | Guardian/write-back proof |
| P0 | Decision Provenance | Differentiator and audit proof |
| P0 | Guardian Agent decision records | Safety enforcement proof |
| P1 | Write-back result evidence | Required for apply mode |
| P1 | Reset/backup/re-run | Repeatable QA |
| P1 | Export JSON evidence | Useful for QA/customer sharing |
| P1 | Batch/checkpoint design | Needed for scale story |
| P2 | Dynamic Workflow as Code | Important but can follow policy trust |
| P2 | RBAC/login | Enterprise hardening |
| P2 | PostgreSQL/object store deployment | Production hardening |

---

## 7. Decisions Needed from Team

### Guardian Agent

Questions:

- Embedded inside Data Agent process?
- Separate sidecar?
- Fail closed when unavailable?
- Own final runtime decision or only validate policy decision?

Recommendation:

```text
For Azure Blob v1, Guardian Agent is embedded/logical inside Data Agent execution and fails closed for mutations.
```

### Zero-Copy

Questions:

- Does zero-copy mean no full file movement?
- Is metadata-only scan acceptable for v1?
- Are sampled-content reads allowed?

Recommendation:

```text
Zero-copy for v1 means no full dataset transfer to Control Plane. Data Agent reads metadata/content in place and uploads only evidence/findings.
```

### Storage

Questions:

- SQLite/DuckDB for local agent state?
- PostgreSQL for production Control Plane?
- OpenSearch/Elastic needed in v1?

Recommendation:

```text
SQLite/DuckDB for local working store, PostgreSQL for Control Plane system of record, OpenSearch optional for future search-heavy evidence queries.
```

### Data Science / NER

Questions:

- Is NER required for Azure Blob v1?
- Which models are approved?
- Can raw content leave customer boundary?

Recommendation:

```text
NER should not block Azure Blob v1. Use rule-based metadata/content classification first. Make model layer pluggable and policy-controlled.
```

### Workflow

Questions:

- Is current six-step flow hard-coded for v1?
- Should Workflow as Code be implemented before QA?

Recommendation:

```text
Keep six-step lifecycle for Azure Blob QA. Add Workflow as Code after policy artifact trust is stable.
```

### Scale

Questions:

- Required test size?
- Expected max files per test?
- Is batch/checkpoint required for QA?

Recommendation:

```text
Define QA tiers: small, medium, large. Implement batching/checkpoint before large-scale QA.
```

---

## 8. Proposed Component Rule Book

### Control Plane

Owns:

- Agent registration
- Data source registration
- Jobs
- Leases
- Policy assignment
- Governance decision
- Audit
- Evidence summary

Does not:

- Scan customer data directly
- Execute connector operations
- Hold full file content by default

### Data Agent

Owns:

- Local execution
- Connector invocation
- Local working store
- Item 13 resolution
- Guardian Agent invocation
- Evidence upload

Does not:

- Execute without lease
- Execute mutation without Guardian Agent
- Upload full sensitive content by default

### Connector Adapter

Owns:

- Store-specific API calls
- Metadata retrieval
- Optional content sampling
- Safe write-back execution

Does not:

- Make policy decisions
- Directly update Control Plane DB

### Item 13

Owns:

- Policy artifact resolution
- Git tag/commit
- Checksum verification
- Signature verification
- Active policy
- Last-known-good fallback

Does not:

- Scan storage
- Execute remediation

### Guardian Agent

Owns:

- Runtime action safety decision
- Allow/review/block
- Fail-closed behavior for mutation

Does not:

- Replace policy governance
- Scan data source directly

### Agent Working Store

Owns:

- Local normalized objects
- Batch checkpoints
- Manifest draft
- Action candidate draft
- Local provenance draft

Does not:

- Act as permanent system of record

### Evidence Publisher

Owns:

- Uploading summary evidence
- Uploading manifest/action/provenance records
- Audit-ready proof package

Does not:

- Upload full file content by default

---

## 9. Implementation Roadmap Before QA

### Step 1 — Finish local working store contract

Add:

```text
GET /v1/workstore/summary
GET /v1/workstore/normalized-objects
```

### Step 2 — Finish evidence contract

Add:

```text
GET /v1/manifests/summary
GET /v1/manifests/details
GET /v1/actions/results
GET /v1/provenance
```

### Step 3 — Finish Guardian Agent output

Add:

```text
GuardianDecision records
ActionCandidate records
ActionResult records
```

### Step 4 — Finish write-back proof

Add:

```text
applied/skipped/blocked/failed status
old/new metadata values where safe
evidence event
audit event
```

### Step 5 — Finish QA scenarios

Create QA test cases for:

- Source config
- Connection test
- Discovery
- Onboarding
- Reset
- Run job
- Dry-run
- Apply mode
- Guardian block
- India → UAE block
- Policy artifact trust
- Git unavailable
- LKG fallback
- Evidence review
- Agent death / retry
- Empty container
- Large dataset

---

## 10. Recommended Next Action

Start with this first implementation task:

```text
Implement Agent Working Store summary and normalized object endpoints.
```

Why first:

```text
Normalized objects are the bridge between connector scan, Agent Core, Guardian Agent, Evidence Center, and future connectors.
```

Once that is implemented, the next two tasks become easier:

```text
Manifest details
Decision Provenance
```
