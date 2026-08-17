# EVEREST — Data Agent Cell Architecture v2

## Based On

This package is based on the updated **Data Agent Cell — Architecture** direction:

```text
Zero-Copy
Orchestrator-First
No Queues
Guardian Embedded
Policy Always Enforced
```

This document intentionally removes the earlier Zubin comparison and explains EVEREST on its own terms.

---

# 1. One-Line Architecture

```text
Client sends request
→ Data Agent Cell Orchestrator builds context
→ Connector reads source by reference/stream/range
→ Worker components analyze/classify/act
→ Guardian authorizes every mutation
→ Provenance Writer records decisions/results
→ Response Composer returns metadata/results
```

Short version:

```text
Request → Orchestrator → Connector / Zero-Copy Access → Analysis / Action → Guardian → Provenance → Results
```

---

# 2. Key Architecture Principles

## 2.1 Colocated with Data

The Data Agent Cell runs close to customer data sources to reduce latency and avoid unnecessary movement of data.

Examples of colocated sources:

```text
Files / shares / object stores
Databases
APIs / SaaS / streams
Structured and unstructured data
```

## 2.2 Zero-Copy by Default

EVEREST operates on customer data **in place**.

This means:

```text
No persistent full copy of customer data by default.
Connector opens handle/session/reference.
Zero-Copy Accessor reads metadata, ranges, chunks, or streams.
Only results, metadata, evidence, decisions, and telemetry are returned.
```

It does **not** mean the runtime never uses memory buffers. Streaming, range reads, SDK buffers, and temporary feature extraction can still happen.

## 2.3 Policy Always Enforced

All actions must be authorized by Guardian.

```text
No Guardian decision = no mutation.
No trusted policy = no unsafe execution.
No bypass path from UI or AI to action executor.
```

## 2.4 Orchestrator-First

The Orchestrator is the only runtime entry and exit point for components and external systems.

All communication is coordinated through Orchestrator.

```text
Client / Portal
  → Orchestrator
  → internal components
  → Orchestrator
  → response / provenance / telemetry
```

## 2.5 No Queues / No Bus

The target architecture avoids external queue/pub-sub dependency for the core path.

Instead:

```text
In-process function calls
Synchronous calls where practical
Local embedded state for checkpoints/retries
Backpressure and flow-control handled by Orchestrator
```

Durable local state is allowed. This is not queue-first architecture.

---

# 3. Main Components

## 3.1 Client / Portal

The client sends requests to the Data Agent Cell.

Examples:

```text
Discover inventory
Classify / tag / label
Profile / analyze
Access / read metadata
Monitor / audit
Assess / plan migration
Copy / move / replicate
Verify / reconcile
Cutover / rollback
```

The client does not directly call connector mutation APIs.

---

## 3.2 Data Agent Cell

The Data Agent Cell is a self-contained runtime unit.

It includes:

```text
Orchestrator
Connector Adapters
Zero-Copy Accessor
Action Executor
Metadata & Cache
Telemetry Collector
Provenance Writer
Embedded Guardian Agent
Local State
```

It intercepts DSPM and migration actions, builds context, decides with Guardian, and executes with zero-copy access.

---

## 3.3 Orchestrator

The Orchestrator is the single control plane inside the Data Agent Cell.

It owns:

```text
request intake
authentication / mTLS / OAuth validation
rate limiting
request validation
correlation ID
identity resolution
resource discovery
classification context
environment context
canonical request creation
policy input building
workflow and state
checkpointing
retries / backoff
routing and coordination
concurrency control
lock management
resource scheduling
backpressure handling
response composition
obligation application
build response
return to client
```

The Orchestrator is both:

```text
the only entry point
the only exit point
```

---

## 3.4 Connector Adapters

Connector Adapters provide protocol-specific access to source systems.

Examples:

```text
File / object
Database
API / SaaS
Other future sources
```

Responsibilities:

```text
open handle/session
list objects/records
read metadata
read ranges/chunks/streams
return references and normalized metadata
execute approved connector-specific actions
```

Connector Adapters should not own policy decisions.

---

## 3.5 Zero-Copy Accessor

The Zero-Copy Accessor provides views on data without creating persistent full copies.

Supported patterns:

```text
memory mapping
direct streaming
scatter/gather I/O
range/chunk access
cursor-based read
bounded preview
```

It provides data access to worker components while keeping customer data in place.

---

## 3.6 Action Executor

Action Executor executes approved operations.

Examples:

```text
discover / scan
classify / tag
copy / move
verify / reconcile
other approved actions
```

It must receive Guardian authorization before performing any mutation.

---

## 3.7 Metadata & Cache

Metadata & Cache stores local metadata and performance caches.

Examples:

```text
resource cache
policy cache
content index cache
result cache
```

This is used to speed up repeated operations and reduce unnecessary source reads.

---

## 3.8 Telemetry Collector

Telemetry Collector records operational signals.

Examples:

```text
metrics
logs
traces
health signals
performance counters
```

This helps answer:

```text
How long did the scan take?
How many objects were processed?
Where was time spent?
Was the system I/O-bound, CPU-bound, or throttled?
```

---

## 3.9 Provenance Writer

Provenance Writer records tamper-evident decision and action records.

It records:

```text
event data
decisions
obligations
results
request actor
policy version
timestamps
environment
integrity signature/hash
```

This is a permanent proof trail.

---

## 3.10 Guardian Agent

Guardian Agent is embedded inside the Data Agent Cell.

Responsibilities:

```text
originates recommended actions based on policy and context
evaluates policy using local policy engine
decides ALLOW / DENY with obligations
hands approved actions and obligations to Data Agent components
records recommendation, decision rationale, and outcome
```

Guardian can use:

```text
policy rules
authorization rules
Python / .NET / Go / whatever scoring engines
local models or heuristics
```

Guardian is not optional for mutation.

---

## 3.11 Local State

Local State is embedded in the Data Agent Cell.

It stores:

```text
local store for configuration
checkpoints
caches
keys
runtime state
temporary result buffers
```

Local State is not the permanent customer-facing system of record.

---

# 4. Where Results Are Stored

This is the missing part that needs to be made explicit.

The diagram shows local state, metadata/cache, telemetry and provenance, but for implementation and QA we also need to separate:

```text
temporary local execution state
permanent result/evidence storage
searchable indexed output
customer source data
```

## 4.1 Storage overview

| Data / result | Component writing it | Storage name | Permanent or transient | Purpose |
|---|---|---|---|---|
| Raw customer files / objects / records | Customer source system | Customer Data Source | Permanent customer-owned | Actual data remains here |
| Request context | Orchestrator | Local State / request context | Transient/local durable | Active execution context |
| Connector handles / sessions | Connector Adapter | Runtime memory | Transient | Access data in place |
| Metadata read from source | Metadata & Cache | Local Metadata Cache | Transient/local cache | Avoid repeated source reads |
| Normalized object records | Orchestrator / Metadata component | Local State / Result Cache | Transient until published | Standard object representation |
| Batch checkpoints | Orchestrator | Local State / Checkpoint Store | Local durable | Resume/retry large operations |
| Classification results | Action Executor / DSPM worker | Result Cache | Transient until published | Sensitivity/risk output |
| Action candidates | Guardian / Orchestrator | Local State / Action Drafts | Transient | Proposed actions before authorization |
| Guardian decisions | Guardian Agent | Provenance Writer | Permanent evidence/provenance | Decision proof |
| Action results | Action Executor | Provenance Writer + Result Cache | Permanent evidence after publish | Applied/skipped/blocked/failed actions |
| Telemetry | Telemetry Collector | Telemetry Store / Observability Backend | Permanent or time-retained | Metrics/logs/traces |
| Manifest summary | Provenance Writer / Response Composer | Evidence Store | Permanent | High-level execution proof |
| Manifest details | Provenance Writer | Evidence Store / Search Index | Permanent | Object/batch-level proof |
| Searchable metadata/results | Response Composer / Publisher | Elasticsearch / OpenSearch | Permanent/rebuildable index | Search/query/dashboard |
| Final response | Response Composer | Client response | Transient | Returned to caller |
| Audit package | Provenance Writer | Evidence Object Store | Permanent | Compliance and review |

## 4.2 Important rule

```text
Customer data remains in the customer source.
EVEREST stores metadata, results, decisions, evidence, and provenance.
EVEREST should not persist full raw customer data by default.
```

## 4.3 Recommended implementation storage names

| Storage | Suggested usage |
|---|---|
| `local_state.db` or embedded local store | request state, checkpoints, runtime context |
| `policy_cache` | active policy and last known good |
| `metadata_cache` | local metadata and content index cache |
| `result_cache` | temporary result records before response/publish |
| `provenance_store` | permanent decision/action evidence |
| `telemetry_store` | metrics, logs, traces, health |
| `elasticsearch` / `opensearch` | searchable normalized results and evidence |
| `evidence_object_store` | exported evidence packages and manifests |

---

# 5. Operation Flows

## 5.1 Discover / Inventory

### Why customer runs it

Customer wants to know what exists before classification or remediation.

### Sequence

```text
Client sends discover request
→ Orchestrator validates request and builds context
→ Connector Adapter opens handle/session
→ Zero-Copy Accessor lists objects/records by reference
→ Metadata & Cache stores local metadata cache
→ Provenance Writer records inventory event
→ Response Composer returns inventory summary
```

### Data read

```text
names
paths
object IDs
sizes
timestamps
ownership/permission signals
storage-native tags/metadata
```

### Data not read by default

```text
full file content
full database rows
full document payloads
```

---

## 5.2 Quick Scan / Metadata Scan

### Why customer runs it

Customer wants fast visibility with minimal data access.

### Sequence

```text
Orchestrator receives quick_scan request
→ Connector lists objects in batches
→ Zero-Copy Accessor reads metadata only
→ Metadata & Cache stores local metadata/result cache
→ DSPM worker may run metadata-only rules
→ Provenance Writer records scan summary
→ Search index receives normalized metadata/results
→ Response Composer returns summary
```

### Backend handling

```text
I/O-bound
batch-based
metadata-only
low memory footprint
no full content persistence
```

### Result storage

```text
Local State / Result Cache: transient scan rows
Elasticsearch/OpenSearch: searchable normalized metadata
Evidence Store: manifest and summary
Provenance Store: scan decision/event history
```

---

## 5.3 Classify / Tag / Label

### Why customer runs it

Customer wants to know what data means and apply safe business/security labels.

### Sequence

```text
Orchestrator receives classify/tag request
→ Policy input builder creates decision context
→ Connector reads metadata and allowed features
→ DSPM/classification worker classifies objects
→ Guardian evaluates if tag/label write-back is requested
→ Action Executor applies only approved labels/tags
→ Provenance Writer records decision and result
```

### Classification inputs

```text
file/object path
file name
metadata
tags
permissions
bounded preview if allowed
business glossary
policy rules
optional NER/model features
```

### Classification outputs

```text
classification label
risk score
sensitivity level
matched rule list
recommended tag/label
obligations
```

---

## 5.4 Profile / Analyze

### Why customer runs it

Customer wants to understand data shape, risk distribution, duplicates, stale data, exposure, and migration readiness.

### Sequence

```text
Connector reads metadata/features
→ Metadata & Cache stores performance cache
→ Analysis workers aggregate results
→ Telemetry Collector records timing and volume
→ Provenance Writer records analysis context
→ Response Composer returns dashboards/rollups
```

### Example outputs

```text
object count
size distribution
file type distribution
owner distribution
risk by folder/source
stale data
over-shared data
classification coverage
migration readiness
```

---

## 5.5 Access / Read Metadata

### Why customer runs it

Customer wants to inspect metadata without downloading the actual file.

### Sequence

```text
Client requests metadata
→ Orchestrator validates access
→ Connector opens handle/reference
→ Zero-Copy Accessor reads metadata only
→ Response Composer returns metadata
→ Provenance Writer records access event
```

### Safety

```text
No full file download.
No raw content unless explicitly approved.
Metadata access is still audited.
```

---

## 5.6 Monitor / Audit

### Why customer runs it

Customer needs proof of what happened.

### Sequence

```text
Telemetry Collector records operational signals
→ Provenance Writer records decision/action proof
→ Response Composer returns audit view
→ Evidence Store holds long-term proof
```

### Stored results

```text
who
what
when
where
why
policy version
decision
action
outcome
duration
component version
environment
```

---

## 5.7 Migration Operations

Migration actions include:

```text
assess / plan
copy / move / replicate
verify / reconcile
cutover / rollback
```

### Required rule

Migration is always Guardian-controlled.

### Sequence

```text
Client requests migration action
→ Orchestrator builds migration context
→ Connector Adapter reads source references
→ Analysis workers assess risk/readiness
→ Guardian validates action against policy and context
→ Action Executor performs approved action
→ Provenance Writer records action receipt and obligations
```

### Result storage

```text
Migration plan: Result Cache / Evidence Store
Action receipt: Provenance Store
Verification result: Evidence Store / Search Index
Telemetry: Telemetry Store
```

---

# 6. Zero-Copy — How It Works

## 6.1 Read path

```text
Data Source
→ Connector Adapter opens reference/session
→ Zero-Copy Accessor reads metadata/range/stream
→ Worker consumes data view
→ Result metadata is generated
→ Data itself remains in source
```

## 6.2 Allowed access patterns

```text
open handle/session
get reference
memory map / stream
direct I/O
scatter/gather I/O
range/chunk read
cursor read
bounded preview
```

## 6.3 What is copied

```text
metadata
classification result
decision result
telemetry
provenance
small bounded features if policy allows
```

## 6.4 What is not copied by default

```text
full data source
full files
full database tables
bulk raw content into Control Plane
raw sensitive payloads into AI
```

---

# 7. No Lock — How It Works

## 7.1 Read path

For most DSPM actions, no source lock is needed because workers read disjoint ranges or object references.

```text
Worker 1 reads range A
Worker 2 reads range B
Worker N reads range N
Shared data remains one copy
```

## 7.2 Write / mutation path

Writes are controlled.

Possible models:

```text
copy-on-write preferred
append-only when possible
exclusive write lock rare
ETag/version check for object stores
idempotency key for actions
orchestrator lock for critical mutations
```

## 7.3 Lock ownership

| Lock / control | Owner | Purpose |
|---|---|---|
| Request lifecycle | Orchestrator | Prevent duplicate execution |
| Local checkpoint lock | Orchestrator | Prevent corrupt checkpoints |
| Action idempotency key | Action Executor | Prevent duplicate writes |
| Storage-native ETag/version | Connector Adapter | Prevent stale writes |
| Guardian decision ID | Guardian | Ensure action uses approved decision |

---

# 8. Concurrency and Scaling

## 8.1 Concurrency model

The Orchestrator controls concurrency.

It manages:

```text
worker routing
resource scheduling
backpressure
lock management
retries/backoff
checkpointing
```

## 8.2 Read path scaling

```text
Batch source objects
Assign disjoint ranges to workers
Read metadata/features in parallel
Store temporary results locally
Publish aggregated results
```

## 8.3 Write path scaling

Writes are slower and more controlled.

```text
Guardian decision required
Obligations enforced
Idempotency key required
Storage-native version check
Limited write concurrency
Action receipt required
```

## 8.4 Example for large source

```text
Files/objects: 2,000,000
Batch size: 1,000
Batches: 2,000
Workers: 16
In-flight batches: configurable
Checkpoint interval: every batch
```

The system should not:

```text
load all records into memory
send one API call per file to Control Plane
copy all files centrally
block the whole source
```

---

# 9. Guardian and Data Agent Contract

The Guardian and Data Agent Cell contract ensures no unauthorized action executes.

## 9.1 Contract flow

```text
Orchestrator sends Action Request to Guardian
Guardian returns ALLOW / DENY / obligations
Orchestrator routes approved action to DAC components
DAC executes approved action only
DAC returns results and provenance
```

## 9.2 No-bypass rule

```text
No action is executed without a valid ALLOW decision from Guardian.
```

## 9.3 Guardian records

```text
recommendation
decision rationale
decision outcome
policy version
obligations
action type
environment
timestamp
integrity signature
```

---

# 10. AI Configuration and Safety

## 10.1 Where AI is configured

AI configuration belongs to the platform/control configuration layer.

It includes:

```text
model provider
local/private/cloud mode
endpoint reference
secret reference
allowed data classes
allowed operations
prompt/context limits
redaction rules
approval requirements
```

## 10.2 How AI works

AI can:

```text
summarize findings
classify or assist classification
explain policy result
recommend metadata/tag values
prepare HITL task text
recommend next action
```

AI cannot:

```text
directly execute writes
delete data
move data
change permissions
export sensitive content
bypass Guardian
```

## 10.3 AI input safety

AI should receive:

```text
redacted metadata
classification signals
policy context
evidence references
bounded snippets only if approved
```

AI should not receive by default:

```text
full raw customer files
unredacted sensitive content
secrets
credentials
```

---

# 11. Adding a New Connector

## 11.1 Connector onboarding

To add a connector:

```text
1. Add connector type to connector catalog.
2. Define source registration fields.
3. Define credential requirements.
4. Implement Connector Adapter.
5. Implement Zero-Copy Access methods.
6. Implement normalization.
7. Define supported actions.
8. Define error model.
9. Define telemetry.
10. Add QA scenarios.
```

## 11.2 Common connector contract

Every connector must support:

```text
ValidateConnection()
Discover()
ReadMetadata()
ReadFeatures()
NormalizeObject()
GetCapabilities()
ApplyApprovedAction()
VerifyAction()
ReturnEvidence()
```

## 11.3 Cross-connector communication

Connectors should not directly talk to each other.

Cross-connector action flow:

```text
Source Connector
→ Orchestrator
→ Guardian
→ Target Connector
→ Provenance Writer
```

Example:

```text
Azure Blob to SharePoint movement
```

Sequence:

```text
Azure Blob Connector reads source references.
Orchestrator builds context.
Guardian validates action, region, risk, and policy.
Target Connector performs approved action.
Provenance records source and target.
```

---

# 12. Adding a New Data Agent Cell

## 12.1 Registration flow

```text
1. Install Data Agent Cell near customer data.
2. Agent bootstraps identity.
3. Agent registers with platform.
4. Agent advertises capabilities.
5. Admin maps sources to agent.
6. Agent starts heartbeat.
7. Agent receives requests/leases.
```

## 12.2 Agent capabilities

```text
supported connectors
network zone
deployment mode
available CPU/memory
max workers
local store capacity
model availability
version
health
```

## 12.3 If agent dies

```text
heartbeat stops
Orchestrator/Control Plane detects timeout
local checkpoint is used for recovery
operation resumes or retries
mutation remains safe because Guardian decision is required
provenance records failure and recovery
```

---

# 13. Customer Data Safety

EVEREST protects customer data by combining:

```text
colocation with data
zero-copy default
least-privilege credentials
mTLS/OAuth request intake
RBAC
policy verification
Guardian authorization
local state and checkpoints
bounded preview
redaction before AI
provenance
telemetry
audit evidence
fail-closed mutation
```

## 13.1 Safe defaults

```text
metadata-only first
dry-run where action risk exists
no raw secret returned
no full data copy by default
no AI direct execution
no mutation without Guardian
all decisions recorded
```

---

# 14. Operation Summary

| Operation | Main components | Result storage |
|---|---|---|
| Discover / Inventory | Orchestrator, Connector, Zero-Copy Accessor, Metadata Cache | Local cache, Evidence Store, Search Index |
| Quick Scan | Orchestrator, Connector, Metadata & Cache, Telemetry, Provenance | Result Cache, Elasticsearch/OpenSearch, Evidence Store |
| Classify / Tag / Label | Orchestrator, DSPM worker, Guardian, Action Executor | Classification results, Provenance Store, Action Receipt |
| Profile / Analyze | Orchestrator, Metadata & Cache, Telemetry Collector | Rollups, Telemetry Store, Search Index |
| Access / Read Metadata | Orchestrator, Connector, Zero-Copy Accessor | Access provenance, metadata response |
| Monitor / Audit | Telemetry Collector, Provenance Writer | Telemetry Store, Evidence Store |
| Copy / Move / Replicate | Orchestrator, Guardian, Action Executor, Connector | Action Receipt, Provenance Store |
| Verify / Reconcile | Orchestrator, Connector, Provenance Writer | Verification evidence |
| Cutover / Rollback | Orchestrator, Guardian, Action Executor | Action receipt, rollback evidence |

---

# 15. Final Mental Model

Use this in team discussions:

```text
Data Agent Cell is the runtime.
Orchestrator controls every step.
Connector reads data in place.
Workers analyze and prepare results.
Guardian approves or denies mutations.
Provenance records everything.
Only metadata/results/evidence move out by default.
```

