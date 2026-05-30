# Item 13 — Demo Plan
## Contract-First Agent Core Runtime Slice

**Owner:** Arbind  
**Goal:** Demonstrate that Agent Core can consume execution state, resolve local policy/workflow, enforce Processing Tier, orchestrate the MVS workflow through contracts, gate actions through Guardian, and emit Batch Summary + Provenance.

---

## 1. Demo Objective

The demo should show a working, contract-first vertical slice of Item 13:

```text
ExecutionState + Lease
   ↓
Agent Core validates state
   ↓
Resolve compiled policy from local Git/cache
   ↓
Resolve preconfigured workflow from local Git/cache
   ↓
Load normalized metadata batch
   ↓
Apply Processing Tier behavior
   ↓
Call mock/contract-shaped Sampling
   ↓
Call mock/contract-shaped Content Intelligence
   ↓
Call mock/contract-shaped Entity Aggregation
   ↓
Call mock Guardian
   ↓
Only after Guardian approval, create Connector Action plan
   ↓
Generate Batch Summary
   ↓
Generate Provenance records
```

---

## 2. What Must Be Real in the Demo

- ExecutionState validation
- compiled policy file resolution from local repo path
- workflow JSON resolution from local repo path
- version-aware policy/workflow cache behavior
- Processing Tier decision logic
- Batch Summary generation
- Provenance record generation
- Guardian-before-action enforcement
- graceful degradation behavior
- REST API endpoints
- sample input/output contracts

---

## 3. What Can Be Mocked Behind Contracts

- Blob connector SDK
- DuckDB internals if time is short
- Statistical Sampling implementation
- Content Intelligence engine
- Entity Aggregation implementation
- Guardian service
- Connector write-back
- Elasticsearch indexing

Mocking is acceptable because this demo is contract-first. The payloads must match agreed contracts so real modules can replace mocks later.

---

## 4. Required Demo Files

```text
contracts/
  execution_state.schema.json
  policy_context.schema.json
  workflow.schema.json
  normalized_batch.schema.json
  sampling_request.schema.json
  content_intelligence_request.schema.json
  guardian_decision.schema.json
  connector_action.schema.json
  batch_summary.schema.json
  provenance_record.schema.json

demo/
  execution-state/
    standard-tier.json
    rolled-up-tier.json
    full-investigation-tier.json
  compiled-policy-repo/
    policies/
      cpv-2026.05.29.001/
        compiled-policy.json
        compiled-policy.sig
    workflows/
      mvs-linear-workflow-v1.json
  normalized/
    blob_batch_001.json
```

---

## 5. Required REST APIs

Minimum:

```text
POST /v1/execution/start
GET  /v1/execution/{execution_id}
GET  /v1/execution/{execution_id}/summary
GET  /v1/execution/{execution_id}/provenance
GET  /v1/cache/status
```

Simulation endpoints:

```text
POST /v1/simulate/git-unavailable
POST /v1/simulate/guardian-failure
POST /v1/simulate/content-intel-failure
POST /v1/simulate/reset
```

---

## 6. Demo Scenario 1 — Standard Tier Happy Path

### Goal
Show the normal MVS runtime path in Standard / Signal mode.

### Steps

1. Start Agent Core service.
2. Submit `standard-tier.json` execution state.
3. Agent Core validates state.
4. Agent Core resolves policy and workflow from local Git structure.
5. Agent Core loads normalized metadata batch.
6. Agent Core applies Tier 2 behavior.
7. Mock Sampling selects high-risk files.
8. Mock Content Intelligence processes sampled files.
9. Mock Entity Aggregation returns entity counts.
10. Mock Guardian approves action.
11. Agent Core creates Connector Action request only after Guardian approval.
12. Agent Core emits Batch Summary.
13. Agent Core emits Provenance records.

### Expected Output

- status: `completed`
- policy loaded from `local_git`
- workflow loaded from `local_git`
- processing tier: `standard_signal`
- Guardian decision present
- Connector action plan present
- Batch Summary present
- Provenance present

---

## 7. Demo Scenario 2 — Rolled-up Mode Still Emits Exceptions

### Goal
Show that Tier 1 remains lean but still captures clear policy exceptions.

### Steps

1. Submit `rolled-up-tier.json`.
2. Agent Core applies Tier 1 behavior.
3. Content Intelligence is skipped or minimized.
4. High-risk metadata exception is still emitted as lean signal.
5. Batch Summary is generated.

### Expected Output

- processing tier: `rolled_up_dashboard`
- content intelligence depth: `minimal_or_skipped`
- signals index output contains policy exception
- Batch Summary is lean

---

## 8. Demo Scenario 3 — Guardian Failure Prevents Write-back

### Goal
Show zero-trust action gating.

### Steps

1. Enable Guardian failure simulation.
2. Submit Standard Tier execution.
3. Agent Core reaches Guardian Evaluation step.
4. Guardian fails/unavailable.
5. Agent Core must not create connector write-back request.
6. Batch completes as degraded or partial success.
7. Provenance records failure reason.

### Expected Output

- status: `degraded`
- connector action request: not created
- reason: `guardian_unavailable`
- partial batch success: true
- provenance recorded: true

---

## 9. Demo Scenario 4 — Git Unavailable Uses Cached Policy/Workflow

### Goal
Show local autonomy and graceful degradation.

### Prerequisite
Run Scenario 1 once so policy/workflow are cached.

### Steps

1. Enable Git unavailable simulation.
2. Submit Standard Tier execution again.
3. Agent Core cannot resolve from Git.
4. Agent Core reuses cached policy/workflow.
5. Execution continues in degraded mode.

### Expected Output

- status: `completed_with_degradation` or `degraded_ready`
- policy activation source: `cache`
- workflow source: `cache`
- fallback used: true
- local execution confirmed: true

---

## 10. Demo Scenario 5 — Full Investigation Mode

### Goal
Show that Tier 3 increases depth.

### Steps

1. Submit `full-investigation-tier.json`.
2. Agent Core applies Tier 3 behavior.
3. Sampling becomes more aggressive.
4. Content Intelligence depth increases.
5. Batch Summary includes richer fields.
6. Indexing plan includes supporting indexes.

### Expected Output

- processing tier: `full_investigation`
- content intelligence depth: `deep`
- indexing mode: `rich_supporting_indexes`
- Batch Summary richness: `high`

---

## 11. Demo Script — Talking Points

Start with:

> This demo focuses on Item 13 as the Agent Core runtime orchestrator. The goal is not to fully implement every downstream skill, but to prove the contracts and execution control points that allow all modules to integrate cleanly.

Then show:

1. ExecutionState input from Item 9.
2. Policy and workflow resolved locally.
3. Processing Tier controls runtime depth.
4. Skills are called through clean contracts.
5. Guardian approval gates action.
6. Batch Summary and Provenance are emitted.
7. Degradation works without breaking the batch.

End with:

> This gives us the core integration skeleton. Each owner can replace the mocked contract endpoints with real implementations without changing Agent Core orchestration behavior.

---

## 12. Success Checklist

- [ ] Scope document reviewed.
- [ ] Contracts document created.
- [ ] Demo plan created.
- [ ] ExecutionState sample created.
- [ ] Policy and workflow sample created.
- [ ] Agent Core API starts.
- [ ] Standard Tier demo works.
- [ ] Rolled-up Tier demo works.
- [ ] Guardian failure demo works.
- [ ] Git/cache fallback demo works.
- [ ] Batch Summary output visible.
- [ ] Provenance output visible.
