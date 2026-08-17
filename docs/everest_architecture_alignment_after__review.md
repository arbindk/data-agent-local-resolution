# Everest Architecture Alignment

## Status

Accepted direction for the next architecture iteration.

This document incorporates feedback on the Everest Azure Blob implementation and the target Data Agent Cell architecture.

## Overall Decision

Keep the current Azure Blob vertical slice because it provides valuable product capability:

- Workbench operations
- DSPM/content intelligence
- AI-assisted recommendations
- HITL task creation
- Evidence Center
- Connector catalog
- Deployment mode support

Refactor the control flow so those capabilities align with the target Everest architecture:

```text
Portal / Intent
  -> Data Agent Cell Orchestrator
  -> Discovery & Analysis
  -> Guardian Decision
  -> Approved Action Execution
  -> Provenance
```

The main architectural correction is to make the Data Agent Orchestrator the mandatory runtime entry point and Guardian the mandatory non-bypassable gate before mutation.

## Confirmed Architecture Decisions

| Decision | Outcome |
|---|---|
| Orchestrator mandatory runtime entry point | Accepted. All scans, reads, AI recommendations, and actions must flow through Agent Core Orchestrator. |
| Workbench positioning | Accepted. Workbench remains valuable but becomes advanced/expert investigation, not the primary MVS1 journey. |
| Zero-copy definition | Accepted. Practical zero-copy means no persistent full duplicate copies by default. Ephemeral feature extraction is allowed. |
| Durable orchestrator state | Accepted. Leases, checkpoints, and local state are allowed under "no queues". |
| Guardian decision versioning | Accepted. Guardian decisions must include policy bundle version/hash. |
| AI restrictions | Accepted. AI may recommend, explain, classify, prefill HITL, and support feature extraction. AI must not directly mutate data. |
| Generic connector contract | Accepted. Azure Blob must move behind a connector-agnostic contract using shared normalized structures. |

## Updated Target Product Behavior

Everest should run in autonomous mode by default, within policy guardrails.

The system should:

1. Take action automatically when policy explicitly allows it.
2. Create HITL tasks when Guardian returns HITL or obligations that require human review.
3. Reduce HITL volume over time as policy confidence and approved automation coverage improve.
4. Preserve governance by keeping Guardian as the non-bypassable mutation gate.

## MVS1 Autonomous Auto-Tagging Flow

The primary MVS1 flow is autonomous governed tagging.

```text
1. Portal submits intent or scheduled policy triggers scan.
2. Data Agent Orchestrator receives the request envelope.
3. Orchestrator uses generic Connector to discover sources, containers, objects, and metadata.
4. Orchestrator performs selective content feature extraction.
5. DuckDB is used as transient working memory for batch aggregation and scoring.
6. Orchestrator calls Guardian for recommendation and decision.
7. Guardian returns allow, deny, modify, obligations, or HITL.
8. If allowed, Orchestrator executes approved tagging through connector adapter.
9. If HITL is required, Orchestrator creates user tasks.
10. Guardian records decision provenance.
11. Orchestrator submits aggregated summary results and normalized signals to Elasticsearch.
```

## Guardian Timing

Guardian is called after discovery, metadata inference, selective feature extraction, and scoring have produced enough context for decisioning.

Guardian is called before any mutation.

This means read/discovery/analysis can occur under orchestrator control, but mutation cannot occur until Guardian returns an executable decision.

## Guardian Ownership

Guardian owns decision provenance.

Each Guardian decision must record:

- Policy bundle version
- Policy bundle hash
- Rules applied
- Reasoning
- Decision outcome
- Obligations
- HITL requirement if any
- Actor/request context
- Timestamp
- Data boundary
- Action type
- Connector/source reference

## Orchestrator Ownership

The Data Agent Orchestrator owns:

- Request envelope intake
- Context building
- Connector routing
- Discovery coordination
- Feature extraction coordination
- Batch state
- Leases
- Checkpoints
- Retries
- Guardian call orchestration
- Approved action dispatch
- HITL task creation
- Batch completion
- Aggregated summary submission to Elasticsearch

## DuckDB Usage

DuckDB should be treated as transient working memory for MVS1 batch processing.

Allowed:

- Normalized metadata
- Derived signals
- Temporary feature vectors
- Batch scoring data
- Aggregation tables
- Checkpoint-supporting summaries

Not allowed by default:

- Long-term full raw content retention
- Persistent duplicate copies of customer files
- Unbounded sensitive content storage

DuckDB state should reset per batch according to MVS design unless explicit retention is approved.

## Elasticsearch Usage

At batch completion, the Orchestrator submits:

- Aggregated summary results
- Normalized signals
- Policy/action outcomes
- Decision references
- Evidence/provenance references

Full findings, including non-normalized data, may be indexed depending on indexing mode and governance configuration.

## Zero-Copy Definition

Accepted definition:

Everest avoids persistent full duplicate copies of customer data by default.

Allowed access patterns:

- Reference
- Handle
- Stream
- Range read
- Cursor
- Bounded preview
- Ephemeral feature extraction

Explicit full export or download requires:

- Guardian approval
- Audit record
- Provenance linkage
- User/action attribution

## Generic Connector Direction

Azure Blob remains the first complete connector implementation, but the product architecture must not be Azure Blob-specific.

The connector contract should normalize:

- Source identity
- Object identity
- Metadata
- Tags
- Access methods
- Capabilities
- Supported actions
- Action requirements
- Error model
- Evidence output

Future connectors should reuse the same flow for:

- File shares
- Databases
- Object stores
- SaaS repositories
- APIs
- Streams

## Current Implementation Gap After Review

| Current Implementation | Required Direction |
|---|---|
| Azure Blob Workbench drives many operations directly | Workbench should call Orchestrator request envelope |
| Guardian/HITL exists but is not proven as mandatory before all mutations | Guardian must be non-bypassable before mutation |
| Evidence Center records useful evidence | Guardian must own decision provenance with policy bundle version/hash |
| Azure Blob-specific UI is strong | Refactor toward generic connector contract |
| Async scan state exists | Treat as orchestrator-managed durable state, not queue-first design |
| AI recommendations exist | AI must be bounded to recommendation/explanation/classification/HITL prefill |
| DuckDB/local store exists | Shape it as transient batch working memory and normalized signal store |

## Phase 1 Refactor Scope

Phase 1 should be led from Agent Core.

### 1. Request Envelope

Introduce a single envelope for all runtime operations:

```text
ActionRequestEnvelope
```

Required fields:

- request_id
- tenant_id
- actor
- role
- source_id
- connector_id
- operation
- intent
- scope
- object_refs
- policy_context
- data_boundary
- correlation_id
- requested_at

### 2. Orchestrator Decision Model

Introduce:

```text
OrchestratorDecision
```

Required fields:

- request_id
- decision_id
- guardian_decision_id
- status
- allowed_operation
- obligations
- hitl_required
- execution_plan
- reason

### 3. Guardian Decision Model

Introduce:

```text
GuardianDecision
```

Required fields:

- decision_id
- policy_bundle_version
- policy_bundle_hash
- decision
- obligations
- rules_applied
- reasoning
- risk_score
- hitl_required
- provenance_record_id

### 4. Action Receipt

Introduce:

```text
ActionReceipt
```

Required fields:

- action_id
- request_id
- decision_id
- connector_id
- source_id
- object_refs
- action_type
- status
- started_at
- completed_at
- result_summary
- error
- receipt_hash

### 5. Provenance Record

Introduce:

```text
DecisionProvenanceRecord
```

Required fields:

- provenance_record_id
- decision_id
- request_id
- actor
- policy_bundle_version
- policy_bundle_hash
- rules_applied
- reasoning
- outcome
- obligations
- action_receipt_id
- timestamp
- signature_or_hash

## Required Enforcement Tests

Phase 1 is not complete without invariant tests.

Required tests:

1. Workbench mutation cannot execute without Orchestrator.
2. AI recommendation cannot execute mutation directly.
3. Orchestrator cannot execute mutation without Guardian decision.
4. Guardian deny blocks action executor.
5. Guardian HITL creates task and blocks direct mutation.
6. Guardian allow with obligations applies obligations before execution.
7. Every mutation attempt creates provenance, even blocked attempts.
8. Every Guardian decision includes policy bundle version/hash.
9. Raw secrets are never returned by source config APIs.
10. Full content export/download requires Guardian approval and audit.
11. Connector adapter receives normalized object references, not portal-specific fields.
12. DuckDB batch state resets or expires according to batch policy.

## UI Implications

The primary customer journey should become:

```text
Dashboard
  -> Source Setup
  -> Autonomous Scan / Auto-Tagging
  -> Guardian Decisions
  -> HITL Tasks if required
  -> Evidence / Provenance
```

Workbench should move to:

```text
Advanced Investigation
```

Workbench should still support:

- Source inspection
- Object-level investigation
- Controlled preview
- Connector troubleshooting
- Expert operations

But Workbench should not appear as the primary MVS1 path.

## Final Architecture Visual Requirements

The final team presentation should include both:

1. Dense PNG architecture summary
2. Animated GIF sequence

### Dense PNG Must Show

- Portal / Intent layer
- Data Agent Cell Orchestrator
- Generic Connector Adapter
- Discovery and metadata inference
- Selective content feature extraction
- DuckDB transient working memory
- Guardian Agent policy decision
- HITL task creation
- Approved action executor
- Decision provenance
- Elasticsearch summary/indexing output
- Zero-copy access boundary
- Azure Blob as one connector implementation
- Future connectors as first-class extension points

### GIF Sequence Should Show

1. User or schedule submits intent.
2. Orchestrator receives request envelope.
3. Generic connector discovers source/object metadata.
4. Zero-copy accessor performs bounded reads/feature extraction.
5. DuckDB aggregates transient batch signals.
6. Guardian evaluates policy and returns decision.
7. Orchestrator auto-applies allowed tags or creates HITL task.
8. Guardian records decision provenance.
9. Orchestrator submits batch summary and normalized signals to Elasticsearch.
10. Evidence view shows decision, policy hash, action receipt, and outcome.

## Recommended Next Step

Create the final architecture visual package from this accepted direction:

- `everest_final_architecture_summary.png`
- `everest_autonomous_flow_sequence.gif`
- Optional supporting markdown:
  - `everest_final_architecture_storyboard.md`

After the visual is accepted, begin Phase 1 refactor around request envelopes, Guardian decisions, action receipts, and enforcement tests.

