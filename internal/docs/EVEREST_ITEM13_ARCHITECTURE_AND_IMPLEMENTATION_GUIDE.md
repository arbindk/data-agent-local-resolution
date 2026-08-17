# Everest Item 13 Architecture & Implementation Guide

## 1. Purpose of this document

This document explains the Everest architecture and the Item 13 implementation in simple language.

It is written for developers, architects, QA, and team members who want to understand:

- What Everest is trying to build.
- What each major component does.
- Where Item 13 fits.
- What we implemented in the current repo.
- How the code is structured.
- How the demo flow works.
- How to build, run, and test the implementation.
- How other MVS items connect to Item 13.

The goal is that anyone reading this document should be able to understand the project without needing deep prior knowledge of the architecture.

---

## 2. Simple explanation of Everest

Everest is an agentic data governance architecture.

In simple words:

> Everest helps customers discover, understand, govern, classify, protect, and take action on data across many different data stores.

A customer may have data in many places:

- Azure Blob Storage
- Amazon S3
- SMB / Windows file shares
- NFS
- SharePoint
- OneDrive
- Google Drive
- Databases
- Data lakes
- FTP / SFTP
- Other future storage systems

Everest should support all of these without redesigning the whole system for every new source.

The key idea is:

> Bring intelligence and governance close to the data, instead of moving all data centrally first.

That is why the **Data Agent** and **Agent Core** are important.

---

## 3. Simple real-world example

Imagine a customer has an Azure Blob container:

```text
blob-finance-prod
```

It contains:

```text
finance/payroll/jan-payroll.xlsx
finance/contracts/acquisition-plan.pdf
finance/public/vendor-list.csv
```

The customer wants to know:

- Which files contain sensitive data?
- Which files have risky permissions?
- Which files should be tagged?
- Which files may need review?
- Which policy was used?
- Why did the system make a decision?
- Can this run locally without depending on central services?

Everest should process this data like this:

```text
1. Policy is authored and approved.
2. Policy is compiled into a signed artifact.
3. A Data Agent receives work for blob-finance-prod.
4. Agent Core resolves the right policy locally.
5. Connector/Quick Scan normalizes metadata.
6. Agent Core decides how deep processing should go.
7. Content Intelligence and Entity Aggregation may run.
8. Guardian approves or blocks governance actions.
9. Connector performs approved write-back actions.
10. Agent Core emits Batch Summary and Provenance.
11. Dashboards and reports show the results.
```

This is the story our Item 13 implementation supports.

---

## 4. Core architecture in simple terms

Everest has three major lifecycle areas:

```text
Policy as Code Lifecycle
Workflow as Code Lifecycle
Data Agent Lifecycle
```

### 4.1 Policy as Code Lifecycle

This is about creating and approving policies.

Example policy:

```text
If a file contains National ID or bank account information,
classify it as Confidential and require local-only processing.
```

Flow:

```text
Policy Author creates policy
   ↓
Human approval happens
   ↓
Policy is stored in human-readable Git
   ↓
Conductor compiles it
   ↓
Compiled policy is signed
   ↓
Compiled artifact is stored in compiled Git
   ↓
Data Agent later resolves it locally
```

Why this matters:

- No unapproved policy should run.
- Every policy should have version history.
- Every runtime policy should be signed.
- The Data Agent should verify the artifact before using it.

---

### 4.2 Workflow as Code Lifecycle

This is about deciding the execution steps.

For the MVS, we are not building a fully dynamic workflow engine.

Instead, we use a simple preconfigured linear workflow:

```text
1. Resolve compiled policy
2. Resolve workflow
3. Quick Scan / normalized metadata
4. Sampling / scoring
5. Content Intelligence
6. Entity Aggregation
7. Guardian Evaluation
8. Action Execution
9. Batch Summary
10. Cleanup
```

In the current implementation, workflow resolution is still simplified. We verify that `workflow_version` exists and treat it as resolved. Later, this should load a real workflow JSON from local Git.

---

### 4.3 Data Agent Lifecycle

This is the runtime execution flow.

The Data Agent runs close to the data source.

Inside the Data Agent, the **Agent Core** is the orchestrator.

Simple flow:

```text
Lease received
   ↓
Execution state received
   ↓
Policy resolved locally
   ↓
Workflow resolved locally
   ↓
Batch processed
   ↓
Guardian decision checked before action
   ↓
Batch Summary generated
   ↓
Provenance generated
```

---

## 5. What is a Data Agent?

A **Data Agent** is the local execution unit.

It runs near the customer data source.

It may run:

- On a server
- In Kubernetes
- As a local process
- In a container
- In a customer-controlled network
- In air-gapped or restricted environments

The Data Agent should be able to continue operating even if central services are temporarily unavailable.

The Data Agent contains:

```text
Agent Core
Connectors
Intelligence capabilities
Guardian
Local cache / local state
Local working storage
```

---

## 6. What is Agent Core?

Agent Core is the central orchestrator inside the Data Agent.

Simple definition:

> Agent Core is the local runtime brain of the Data Agent.

It does not directly implement every capability.

Instead, it coordinates the capabilities.

Agent Core owns:

```text
- receiving execution state
- receiving lease
- resolving policy
- resolving workflow
- enforcing processing tier
- deciding how deep to process
- coordinating other modules
- ensuring Guardian approval before action
- creating Batch Summary
- creating Provenance
- handling degradation and fallback
```

Agent Core does not own:

```text
- policy authoring UI
- HITL approval
- policy compilation
- real Blob SDK implementation
- real Content Intelligence engine
- real Guardian engine
- real Elasticsearch roll-up
- final Decision Provenance UI
```

---

## 7. What is Item 13?

Item 13 is:

```text
Data Agent Local Resolution & Caching
```

But after reading the full MVS architecture, Item 13 is broader than just a cache.

Item 13 means:

> Agent Core resolves and caches policy/workflow locally, consumes execution state, enforces processing tier, orchestrates batch processing, gates actions through Guardian, and emits Batch Summary plus Provenance.

The one-line story:

> Item 13 receives a lease and execution state, resolves trusted policy and workflow locally, controls how deeply the batch is processed, gates actions through Guardian, and emits batch summary plus provenance.

---

## 8. Key terms and meanings

### 8.1 Policy

A policy defines governance rules.

Example:

```text
If file contains PII.Email or Financial.BankAccount,
classify it as Confidential.
```

### 8.2 Human-readable policy

A policy written in a format humans can understand, usually YAML or JSON.

Used before compilation.

### 8.3 Compiled policy

A runtime-ready policy artifact produced by Conductor.

It contains:

```text
- source policy reference
- compiled policy version
- entities_of_interest
- decision rules
- remediation rules
- sovereignty rules
- signature metadata
```

### 8.4 Compiled Git Repo

A Git repository that stores signed compiled policy artifacts and workflow artifacts.

The Data Agent resolves artifacts locally from this repo.

### 8.5 Lease

A lease is a work assignment.

It tells the Data Agent:

```text
Process this data store using this policy version and this workflow version.
```

Important lease fields:

```text
lease_id
job_id
data_store_id
compiled_policy_version
compiled_git_tag
compiled_git_commit
compiled_artifact_hash
workflow_version
workflow_git_tag
```

### 8.6 Execution State

Execution state is prepared by Item 9.

It tells Agent Core how to behave.

Example fields:

```text
data_store_id
processing_tier
processing_mode
policy_version
workflow_version
settings
operational_config
```

### 8.7 Processing Tier

Processing tier controls cost and depth.

```text
tier_1 = Rolled-up / Summary / Lean mode
tier_2 = Standard / Default mode
tier_3 = Full Investigation / Deep mode
```

### 8.8 Processing Mode

A more user-friendly mode value.

Examples:

```text
summary
standard
full_investigation
```

### 8.9 Normalized Metadata

Connectors convert native source data into a standard shape.

Agent Core does not want to know every Blob, SMB, NFS, or SharePoint detail.

It wants normalized records like:

```text
file_id
path
size_bytes
content_type
permission_risk_score
dd_protection_flags
last_accessed
source_system
```

### 8.10 entities_of_interest

Entities that the active policy cares about.

Examples:

```text
PII.Email
PII.NationalID
Financial.BankAccount
```

Content Intelligence and Entity Aggregation should focus only on these entities.

### 8.11 Guardian

Guardian is the policy enforcement / decision approval component.

Important rule:

> Agent Core must not request write-back or governance actions until Guardian approves.

### 8.12 Write-back action

A write-back action changes the source or target system.

Examples:

```text
apply tag
apply sensitivity label
apply dd_protection_flags
quarantine
archive
delete
move
```

### 8.13 Batch Summary

A compact summary of what happened in one batch.

It includes:

```text
file count
risk summary
classification summary
actions planned/taken
policy exceptions
sovereignty summary
active policy version
workflow version
processing tier
```

Item 13 emits it. Item 15 rolls it up.

### 8.14 Provenance

Provenance explains why a decision happened.

It answers:

```text
Which policy was active?
Which workflow was active?
Which rules applied?
Did Guardian approve?
Was local execution confirmed?
Was HITL needed?
```

---

## 9. MVS item map and where Item 13 connects

### Item 1 — Governed Policy Promotion

Owns the approval flow for policies.

Produces:

```text
approved human-readable policy
```

Item 13 does not talk directly to Item 1, but Item 13 depends on the final approved compiled output.

---

### Item 2 — Conductor Compilation Service

Compiles approved policies into runtime artifacts.

Produces:

```text
compiled-policy.json
compiled-policy.sig
compiled_artifact_hash
compiled_git_tag
compiled_git_commit
entities_of_interest
```

Item 13 consumes these through the local compiled Git repo.

---

### Item 3 — Policy Hierarchy Resolution

Decides which policy should apply.

Hierarchy example:

```text
Specific Request
   overrides
Data Store Override
   overrides
Enterprise Default
```

Item 13 does not redo this decision.

Item 13 trusts the lease but verifies the artifact.

---

### Item 4 — Data Store Policy Override & History

Allows policy overrides per data store.

Item 13 sees the result through the lease/execution state.

---

### Item 5 — Policy Governance UI + Intelligent Tagging Wizard

User-facing policy and tagging experience.

It may eventually trigger flows that create policies, previews, or actions.

Item 13 may generate provenance that this UI can show.

---

### Item 6 — Statistical Sampling Strategies

Determines which files deserve deeper analysis.

Item 13 controls when/how to call Item 6 based on tier.

---

### Item 7 — Quick Scan Analytics

Collects fast metadata early.

In the architecture, Quick Scan lives close to the connector and writes metadata to DuckDB.

Item 13 consumes the normalized metadata.

In our demo, we simulate this using:

```text
demo/normalized/blob-batch-001.json
```

---

### Item 8 — Classification Execution

Performs classification/write-back behavior for source systems.

Important:

> Write-back actions must happen only after Guardian approval.

Item 13 enforces that ordering.

---

### Item 9 — Agent Core Execution State Initialization

Prepares the complete execution state.

Item 9 gives Item 13:

```text
compiled policy reference
processing tier
processing mode
workflow reference
settings
operational config
```

Our demo request includes this as:

```json
"execution_state": { ... }
```

---

### Item 10 — Entity Frequency Aggregation

Aggregates entity counts.

Item 13 sends or consumes aggregated signals for summary/provenance.

In our demo, entity counts are simulated from `entities_of_interest`.

---

### Item 11 — Compliance & Achievement Reporting

Builds dashboards/reports.

It consumes rolled-up data eventually produced from Item 15.

Item 13 indirectly feeds this through Batch Summary.

---

### Item 12 — Decision Provenance Side Panel

Displays provenance records.

Item 13 creates provenance records.

Item 12 displays them.

---

### Item 13 — Data Agent Local Resolution & Caching / Agent Core

Our module.

Owns:

```text
policy/workflow local resolution
cache
execution orchestration
tier enforcement
Guardian-before-action rule
Batch Summary
Provenance
```

---

### Item 14 — Content Intelligence Engine

Extracts policy-relevant entities and signals.

Item 13 controls when and how deeply Item 14 runs.

In our demo, this is simulated.

---

### Item 15 — Batch Summary & Data Store Roll-up

Consumes Batch Summary from Item 13.

Performs long-term roll-up into datastore/tenant dashboards.

---

## 10. Architecture diagram

```text
Policy Author / Maestro
        |
        v
HITL Approval
        |
        v
Human-readable Git Repo
        |
        v
Conductor
        |
        v
Compiled Git Repo
        |
        v
+--------------------------------------------------+
| Data Agent                                       |
|                                                  |
|  +--------------------------------------------+  |
|  | Agent Core / Item 13                       |  |
|  |--------------------------------------------|  |
|  | Receives Lease + Execution State           |  |
|  | Resolves Policy + Workflow Locally         |  |
|  | Enforces Processing Tier                   |  |
|  | Orchestrates Batch                         |  |
|  | Calls Skills / Contracts                   |  |
|  | Enforces Guardian Before Action            |  |
|  | Emits Batch Summary + Provenance           |  |
|  +----------------------+---------------------+  |
|                         |                        |
|      +------------------+------------------+     |
|      |                  |                  |     |
|  Connector          Content Intel       Guardian |
|  / Quick Scan       / Aggregation       Decision |
|      |                  |                  |     |
|      +------------------+------------------+     |
|                         |                        |
|                  Batch Summary                  |
|                         |                        |
+-------------------------+------------------------+
                          |
                          v
                 Item 15 Roll-up / Reports
```

---

## 11. Runtime sequence diagram

```text
Client / Demo
   |
   | POST /v1/execution/start
   v
API Server
   |
   v
Execution Orchestrator
   |
   v
Policy Resolver
   |
   | Resolve compiled-policy.json
   | Verify SHA-256
   | Verify Ed25519 signature
   | Activate policy
   v
Policy Context
   |
   v
Load Normalized Batch
   |
   v
Apply Processing Tier
   |
   v
Simulate Skill Calls
   |
   v
Simulate Guardian Approval
   |
   v
Generate Batch Summary
   |
   v
Generate Provenance
   |
   v
Store Execution Result
```

---

## 12. Current implementation scope

### Real in current implementation

```text
Go service
REST APIs
compiled policy artifact
local compiled Git repo folder
SHA-256 verification
Ed25519 signature verification
policy activation
last-known-good cache
normalized batch loading
processing tier behavior selection
batch summary generation
provenance generation
execution status APIs
```

### Simulated in current implementation

```text
real workflow JSON resolution
real Guardian service
real Content Intelligence
real Entity Aggregation
real Connector write-back
real DuckDB
real Elasticsearch
real UI
```

This is acceptable because this implementation is a contract-first demo-ready Agent Core runtime slice.

---

## 13. Repo structure

```text
data-agent-local-resolution/
│
├── cmd/
│   ├── policy-agent/
│   │   └── main.go
│   └── policy-tool/
│       └── main.go
│
├── internal/
│   ├── api/
│   │   └── server.go
│   ├── config/
│   │   └── config.go
│   ├── policyresolver/
│   │   ├── models.go
│   │   ├── resolver.go
│   │   ├── validator.go
│   │   ├── hash.go
│   │   ├── signature.go
│   │   ├── git_resolver.go
│   │   ├── parser.go
│   │   └── cache.go
│   ├── execution/
│   │   ├── models.go
│   │   ├── manager.go
│   │   └── orchestrator.go
│   ├── batchsummary/
│   │   └── models.go
│   └── provenance/
│       └── models.go
│
├── contracts/
├── demo/
│   ├── compiled-policy-repo/
│   ├── execution/
│   ├── keys/
│   ├── leases/
│   └── normalized/
│
├── data/
│   ├── active-policy/
│   ├── audit/
│   ├── executions/
│   ├── last-known-good/
│   ├── policy-cache/
│   ├── provenance/
│   └── summaries/
│
├── docs/
├── scripts/
├── bin/
└── go.mod
```

---

## 14. Code walkthrough

### 14.1 `cmd/policy-agent/main.go`

This is the service entry point.

Responsibilities:

```text
- load config
- create policy resolver
- create execution manager
- create orchestrator
- create API server
- start HTTP server
```

Flow:

```text
main()
  → config.Load()
  → policyresolver.NewResolver()
  → execution.NewManager()
  → execution.NewOrchestrator()
  → api.NewServer()
  → http.ListenAndServe()
```

---

### 14.2 `internal/api/server.go`

This file owns HTTP routes.

Important routes:

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

It receives JSON, converts it into Go structs, calls the correct internal service, and returns JSON response.

---

### 14.3 `internal/config/config.go`

Loads runtime paths.

Important defaults:

```text
COMPILED_REPO_PATH = demo/compiled-policy-repo
POLICY_CACHE_PATH = data/policy-cache
ACTIVE_POLICY_PATH = data/active-policy
LKG_POLICY_PATH = data/last-known-good
AUDIT_PATH = data/audit
POLICY_PUBLIC_KEY_PATH = demo/keys/ed25519-public.b64
```

---

### 14.4 `internal/policyresolver/models.go`

Defines policy-related models.

Important models:

```text
PolicyLease
ArtifactRef
ArtifactLocation
CompiledPolicyArtifact
PolicyContext
ResolveResult
```

---

### 14.5 `internal/policyresolver/validator.go`

Validates required lease fields.

Important function:

```go
ValidateLease(l PolicyLease) error
```

---

### 14.6 `internal/policyresolver/hash.go`

Computes and verifies SHA-256.

Important functions:

```go
ComputeSHA256(path string)
VerifySHA256(path string, expected string)
```

---

### 14.7 `internal/policyresolver/signature.go`

Verifies Ed25519 signature.

Important function:

```go
VerifyEd25519(publicKeyPath, artifactPath, signaturePath string)
```

---

### 14.8 `internal/policyresolver/git_resolver.go`

Finds the compiled artifact in local Git folder.

Important function:

```go
Resolve(ref ArtifactRef)
```

---

### 14.9 `internal/policyresolver/parser.go`

Parses `compiled-policy.json`.

Important function:

```go
ParseCompiledPolicy(path string)
```

---

### 14.10 `internal/policyresolver/cache.go`

Stores active and last-known-good policy contexts.

Important functions:

```go
Activate(ctx *PolicyContext)
GetActive(dataStoreID string)
GetLastKnownGood(dataStoreID string)
```

---

### 14.11 `internal/policyresolver/resolver.go`

Main policy resolution flow.

Important function:

```go
ResolveAndActivatePolicy(ctx, lease)
```

Flow:

```text
validate lease
resolve artifact
verify hash
verify signature
parse artifact
create policy context
activate/cache policy
return result
```

---

### 14.12 `internal/execution/models.go`

Defines Agent Core execution models.

Important models:

```text
ExecutionStartRequest
ExecutionLease
ExecutionState
NormalizedBatch
NormalizedFileRecord
ExecutionStatus
ExecutionResult
```

---

### 14.13 `internal/execution/manager.go`

Stores execution outputs in memory.

Important functions:

```go
SaveStatus
GetStatus
SaveSummary
GetSummary
SaveProvenance
GetProvenance
```

---

### 14.14 `internal/execution/orchestrator.go`

Main Agent Core demo orchestrator.

Important function:

```go
Start(ctx, req)
```

Flow:

```text
create RUNNING status
resolve policy
resolve workflow placeholder
load normalized batch
determine tier behavior
build batch summary
build provenance
mark COMPLETED
save results
```

---

### 14.15 `internal/batchsummary/models.go`

Defines Batch Summary structure.

Used as output for Item 15.

---

### 14.16 `internal/provenance/models.go`

Defines Provenance record structure.

Used as output for Item 12.

---

## 15. Demo flow with Azure Blob example

Input data store:

```text
blob-finance-prod
```

Demo files:

```text
jan-payroll.xlsx        risk score 92
vendor-list.csv         risk score 38
acquisition-plan.pdf    risk score 78
```

Execution request:

```text
demo/execution/execution-standard.json
```

When we call:

```text
POST /v1/execution/start
```

Agent Core does:

```text
1. Reads lease.
2. Reads execution state.
3. Resolves cpv-2026.05.29.001 from local Git.
4. Verifies hash.
5. Verifies signature.
6. Activates policy.
7. Resolves workflow placeholder.
8. Loads blob-batch-001.json.
9. Applies tier_2 standard behavior.
10. Calculates risk:
    - one critical
    - one high
    - one low
11. Creates policy exception for file-001.
12. Simulates Guardian approval.
13. Plans write-back action.
14. Generates Batch Summary.
15. Generates Provenance.
16. Returns completed result.
```

---

## 16. Build and run

### 16.1 Run tests

```powershell
go test ./...
```

Expected output:

```text
? everest.local/data-agent-policy-resolver/cmd/policy-agent [no test files]
? everest.local/data-agent-policy-resolver/cmd/policy-tool [no test files]
? everest.local/data-agent-policy-resolver/internal/api [no test files]
? everest.local/data-agent-policy-resolver/internal/config [no test files]
? everest.local/data-agent-policy-resolver/internal/policyresolver [no test files]
? everest.local/data-agent-policy-resolver/internal/execution [no test files]
? everest.local/data-agent-policy-resolver/internal/batchsummary [no test files]
? everest.local/data-agent-policy-resolver/internal/provenance [no test files]
```

`[no test files]` means Go compiled the package successfully but there are no unit tests yet.

---

### 16.2 Build service

```powershell
go build -o .\bin\policy-agent.exe .\cmd\policy-agent
```

No output means success.

---

### 16.3 Start service

```powershell
.\bin\policy-agent.exe
```

Expected:

```text
Data Agent / Agent Core demo service starting on :8080
```

---

### 16.4 Health check

```powershell
curl.exe http://localhost:8080/healthz
```

Expected:

```json
{
  "status": "ok",
  "service": "data-agent-local-resolution"
}
```

---

### 16.5 Start execution

```powershell
curl.exe -X POST http://localhost:8080/v1/execution/start `
  -H "Content-Type: application/json" `
  --data-binary "@demo/execution/execution-standard.json"
```

Expected result should include:

```json
{
  "status": {
    "execution_id": "exec-001",
    "status": "COMPLETED",
    "policy_resolved": true,
    "workflow_resolved": true,
    "guardian_approved": true,
    "records_processed": 3
  }
}
```

---

### 16.6 Get execution status

```powershell
curl.exe http://localhost:8080/v1/execution/exec-001
```

Expected:

```json
{
  "execution_id": "exec-001",
  "status": "COMPLETED",
  "records_processed": 3,
  "policy_resolved": true,
  "workflow_resolved": true
}
```

---

### 16.7 Get summary

```powershell
curl.exe http://localhost:8080/v1/execution/exec-001/summary
```

Expected summary includes:

```json
{
  "batch_id": "batch-001",
  "file_count": 3,
  "risk_summary": {
    "critical": 1,
    "high": 1,
    "medium": 0,
    "low": 1
  }
}
```

---

### 16.8 Get provenance

```powershell
curl.exe http://localhost:8080/v1/execution/exec-001/provenance
```

Expected records include:

```text
policy_activation
guardian_before_action
```

---

### 16.9 Cache status

```powershell
curl.exe http://localhost:8080/v1/cache/status
```

Expected:

```json
{
  "status": "available",
  "last_known_good_policy": "enabled"
}
```

---

## 17. Common troubleshooting

### Port 8080 already in use

Check:

```powershell
netstat -ano | findstr :8080
```

Kill:

```powershell
taskkill /PID <PID> /F
```

---

### Hash mismatch

Recompute hash:

```powershell
$path = "demo\compiled-policy-repo\policies\cpv-2026.05.29.001\compiled-policy.json"
$hash = Get-FileHash $path -Algorithm SHA256
"sha256:" + $hash.Hash.ToLower()
```

Update the hash in:

```text
demo/execution/execution-standard.json
demo/leases/valid-lease.json
```

---

### Signature verification failed

Artifact may have changed after signing.

Resign:

```powershell
.\bin\policy-tool.exe sign `
  -private demo\keys\ed25519-private.b64 `
  -artifact demo\compiled-policy-repo\policies\cpv-2026.05.29.001\compiled-policy.json `
  -out demo\compiled-policy-repo\policies\cpv-2026.05.29.001\compiled-policy.sig
```

Then verify:

```powershell
.\bin\policy-tool.exe verify `
  -public demo\keys\ed25519-public.b64 `
  -artifact demo\compiled-policy-repo\policies\cpv-2026.05.29.001\compiled-policy.json `
  -sig demo\compiled-policy-repo\policies\cpv-2026.05.29.001\compiled-policy.sig
```

---

## 18. What to say in the demo

Use this explanation:

> This demo shows Item 13 as the Agent Core runtime orchestrator. It receives a lease and execution state, resolves a signed compiled policy from local Git, verifies hash and signature, loads normalized metadata from a Blob-like batch, enforces Tier 2 standard behavior, simulates Guardian-before-action, generates a raw Batch Summary for Item 15, and creates Provenance records for Item 12.

---

## 19. What questions the team may ask

### Q1. Is Item 13 only a cache?

No.

Item 13 includes local policy/workflow resolution, caching, tier enforcement, orchestration, batch summary, and provenance.

### Q2. Why do we still have policy resolver?

Because Agent Core cannot safely run without verifying the active compiled policy.

### Q3. Why are Guardian and Content Intelligence mocked?

Because this is a contract-first Agent Core runtime slice. Real modules can replace the mocks later without changing the orchestration boundary.

### Q4. Why does Agent Core create Batch Summary?

Because Agent Core runs the batch and knows what happened. Item 15 owns roll-up and long-term storage.

### Q5. Why does Agent Core create Provenance?

Because it knows which policy/workflow/tier/rules were active during runtime. Item 12 displays provenance.

### Q6. Why do we use normalized batch input?

Because Agent Core should be protocol-agnostic. Connectors hide Blob/SMB/NFS/S3-specific details.

### Q7. What is the strongest value of this demo?

It proves the integration model:

```text
execution state + lease + local policy + normalized metadata + tier behavior + guardian gate + summary + provenance
```

---

## 20. Current gaps and next steps

Current gaps:

```text
workflow JSON is not fully resolved from Git
Guardian is mocked
Content Intelligence is mocked
Entity Aggregation is mocked
DuckDB is not integrated
Elasticsearch is not integrated
execution results are stored in memory only
no UI yet
unit tests are minimal/not added yet
```

Recommended next steps:

```text
1. Add real workflow JSON resolution from local Git.
2. Persist execution status/summary/provenance to disk.
3. Add unit tests.
4. Add simple browser UI.
5. Add real Guardian endpoint integration.
6. Replace demo normalized JSON with Quick Scan output.
7. Add DuckDB working memory.
8. Add Elasticsearch indexing plan/output.
9. Add gRPC contracts.
```

---

## 21. Final summary

The current implementation is a demo-ready Agent Core runtime slice for Item 13.

It demonstrates:

```text
- local compiled policy resolution
- hash verification
- signature verification
- policy activation
- active/LKG cache
- execution state consumption
- normalized metadata consumption
- processing tier behavior
- Guardian-before-action simulation
- Batch Summary generation
- Provenance generation
- REST APIs for demo
```

The most important takeaway:

> Item 13 is the local runtime brain of the Data Agent. It ensures that the Data Agent runs with a trusted policy, follows the correct workflow, controls processing cost, coordinates downstream components, and produces auditable output.
