# Item 13 — Agent Core Local Resolution, Caching & Runtime Orchestration
## Scope and Boundaries

**Owner:** Arbind  
**MVS Item:** 13 — Data Agent Local Resolution & Caching  
**Lifecycle:** Data Agent Lifecycle + Workflow  
**Primary Goal:** Build the Agent Core as the contract-first runtime orchestrator for the Data Agent.

---

## 1. Purpose

Item 13 is the central runtime orchestration module inside the Data Agent. It is responsible for receiving a prepared execution state from Item 9, resolving and caching the compiled policy and preconfigured workflow from local Git, enforcing the Processing Tier, orchestrating the MVS linear workflow, gating any write-back through Guardian approval, and generating the raw Batch Summary at the end of the batch.

This module is not just a local policy cache. It is the runtime control layer that coordinates the Data Agent execution path.

---

## 2. Core Responsibilities

### 2.1 Execution State Consumption

Item 13 consumes the prepared execution state from Item 9. The execution state must include at minimum:

- lease ID
- batch ID
- data store ID
- connector reference
- compiled policy reference
- workflow reference
- processing tier
- execution configuration
- indexing configuration
- runtime limits

Agent Core must not start processing unless execution state is valid.

### 2.2 Local Policy Resolution and Caching

Agent Core resolves compiled policy artifacts from the local compiled Git repository using lease reference fields:

- `compiled_policy_version`
- `compiled_git_tag` or `compiled_git_commit`
- `compiled_artifact_hash`

It should reload policy only when the version/hash changes. If unchanged, cached policy context should be reused.

### 2.3 Local Workflow Resolution and Caching

For the MVS, workflow is a simple preconfigured linear workflow stored in the compiled Git repo. Agent Core resolves and caches this workflow using the workflow reference supplied in execution state.

Default MVS workflow:

1. Resolve compiled policy
2. Resolve preconfigured workflow
3. Quick Scan / normalized metadata collection through Connector
4. High-speed metadata scoring and sampling
5. Content Intelligence based on sampling and tier
6. Entity Aggregation
7. Guardian Evaluation
8. Connector write-back only after Guardian approval
9. Batch Summary generation
10. Cleanup temporary state

### 2.4 Processing Tier Enforcement

Agent Core converts the Processing Tier into runtime behavior.

| Tier | Agent Core behavior |
|---|---|
| Tier 1 — Rolled-up / Dashboard | Minimal content intelligence, lean signals, aggregate-first behavior |
| Tier 2 — Standard / Signal | Default mode, selective content intelligence, high-signal ES fields |
| Tier 3 — Full Investigation | Deeper intelligence, richer signals, possible supporting indexes |

Agent Core decides:

- how aggressively to invoke Item 6 scoring and sampling
- how much Content Intelligence Item 14 should run
- what should stay in DuckDB
- what should be promoted to Elasticsearch
- what goes into the Batch Summary

### 2.5 Connector Coordination

Connectors own data source access, normalization, Quick Scan, and governance write-back actions. Agent Core only orchestrates when connectors are called.

Agent Core must not directly read native storage. It consumes normalized data produced by connectors.

### 2.6 Guardian-Before-Action Enforcement

No persistent governance action may be requested until Guardian approval is received.

This includes:

- tagging
- `dd_protection_flags`
- sensitivity labels
- quarantine
- archive
- delete
- remediation actions

Write-back is executed by connectors after Guardian approval.

### 2.7 Batch Summary Generation

At the end of each batch, Agent Core emits the raw Batch Summary. Item 15 owns roll-up, Elasticsearch lifecycle, compression, retention, and higher-level aggregation.

Batch Summary richness must depend on Processing Tier.

### 2.8 Resilience and Graceful Degradation

Agent Core must isolate failures where possible.

Expected behavior:

- one file failure should not fail the whole batch
- optional skill failure should degrade processing when safe
- systemic failures should stop the batch safely
- partial success should be reported
- all failures should produce provenance/audit records

---

## 3. In Scope for Tomorrow's PoC

The PoC will implement a contract-first vertical slice:

- ExecutionState contract validation
- local policy resolution and cache behavior
- local workflow resolution and cache behavior
- Processing Tier decision logic
- mock/contract-shaped skill calls
- Guardian-before-action enforcement
- raw Batch Summary generation
- provenance record generation
- graceful degradation simulation
- REST demo API
- sample input/output files

---

## 4. Out of Scope for Tomorrow's PoC

The following will not be fully implemented tomorrow:

- real Blob SDK integration
- real DuckDB full integration if time is short
- real Arrow Flight
- real Content Intelligence engine
- real Guardian service
- real Elasticsearch indexing
- real connector write-back
- full Workflow as Code engine
- production-grade cache eviction
- production-grade Git sync

These will be represented by clear contracts and mock implementations where needed.

---

## 5. Ownership Boundaries

| Area | Owner | Item 13 responsibility |
|---|---|---|
| Execution State | Item 9 | Consume and validate before processing |
| Compiled Policy Artifact | Item 2 | Resolve, verify/cache locally, expose policy context |
| Policy Hierarchy | Item 3 | Consume resolved compiled policy reference from lease/state |
| Quick Scan | Item 7 / Connector | Orchestrate, consume normalized metadata |
| Sampling / Scoring | Item 6 | Invoke and consume sampling decision |
| Content Intelligence | Item 14 | Control invocation depth and consume signals |
| Entity Aggregation | Item 10 | Invoke and consume aggregate output |
| Guardian | Guardian sidecar | Request approval before action |
| Write-back | Item 8 / Connector | Instruct connector only after Guardian approval |
| Batch Summary Roll-up | Item 15 | Emit raw Batch Summary only |
| Decision Provenance UI | Item 12 | Generate records; UI displays them |

---

## 6. Key Decisions

1. Agent Core is the runtime orchestrator, not a data source implementation.
2. Connectors are the data access and action layer.
3. Guardian approval is mandatory before persistent write-back.
4. Policy and workflow are resolved from local Git and cached by version.
5. Processing Tier is enforced by Agent Core at runtime.
6. DuckDB is the batch working-memory target; Elasticsearch receives only controlled outputs.
7. Batch Summary is generated by Agent Core; roll-up is owned by Item 15.
8. Tomorrow's build should prove contracts and orchestration, not production-level external integrations.

---

## 7. Success Criteria for Tomorrow

- A valid execution state can start an Agent Core run.
- Agent Core resolves policy and workflow from local files/Git structure.
- Agent Core applies tier behavior.
- Agent Core executes a mock MVS linear workflow through contract-shaped calls.
- Guardian-before-action rule is demonstrated.
- Batch Summary is emitted.
- Provenance records are emitted.
- Git/cache failure scenario is handled through degradation.
- The demo clearly shows what is real, what is mocked, and where other modules plug in.
