# Everest Final Architecture Storyboard

## Accepted Direction

This storyboard reflects the architecture direction accepted after Cuong's review:

```text
Portal / Intent
  -> Data Agent Cell Orchestrator
  -> Discovery & Analysis
  -> Guardian Decision
  -> Approved Action Execution or HITL Task
  -> Decision Provenance
  -> Elasticsearch Summary
```

## Core Architecture Invariants

1. Orchestrator is the mandatory runtime entry point.
2. Guardian is the non-bypassable gate before mutation.
3. Autonomous mode is default when policy explicitly allows action.
4. HITL tasks are created only when Guardian requires review or obligations cannot be satisfied automatically.
5. AI can recommend, classify, explain, extract features, and prefill HITL; AI cannot directly mutate data.
6. Zero-copy means no persistent full duplicate copies by default.
7. DuckDB is transient batch working memory for normalized signals, scoring, and aggregation.
8. Guardian owns decision provenance with policy bundle version/hash.
9. Azure Blob is the first connector implementation behind a generic connector contract.
10. Elasticsearch receives aggregated summary results and normalized signals at batch completion.

## Dense PNG Contents

The dense architecture summary should show:

- Portal / Intent layer
- Data Agent Cell Orchestrator
- Request envelope
- Discovery and metadata inference
- Generic connector adapter
- Zero-copy accessor
- Selective content feature extraction
- DuckDB transient working memory
- Guardian Agent decision point
- Approved action executor
- HITL task creator
- Decision provenance writer
- Elasticsearch summary/index output
- Policy authoring, compilation/signing, and distribution
- Azure Blob plus future connector targets
- Non-bypassable mutation boundary

## Animated GIF Frames

### Frame 1: Intent Enters Orchestrator

The user, schedule, or autonomous policy trigger submits intent. The Portal does not execute directly; it creates a request envelope for the Orchestrator.

### Frame 2: Orchestrator Builds Context

The Orchestrator normalizes actor, tenant, source, connector, operation, scope, object references, and policy context.

### Frame 3: Generic Connector Discovery

The Orchestrator uses the generic connector contract to discover sources, containers, objects, metadata, and capabilities.

### Frame 4: Zero-Copy Feature Extraction

The Zero-Copy Accessor uses handle, stream, range, cursor, bounded preview, or ephemeral extraction. No persistent full duplicate copy is created by default.

### Frame 5: DuckDB Working Memory

DuckDB holds transient batch-normalized metadata, features, scoring inputs, and aggregation state. It resets or expires according to batch policy.

### Frame 6: Guardian Decision

Guardian evaluates policy, risk, obligations, and HITL requirements. It returns allow, deny, modify, obligations, or HITL.

### Frame 7: Autonomous Action Or HITL

If Guardian allows the operation, the Orchestrator executes the approved tagging/action through the connector adapter. If Guardian requires HITL, the Orchestrator creates a user task.

### Frame 8: Provenance Recorded

Guardian records decision provenance with policy version/hash, rules applied, reasoning, obligations, outcome, actor, and timestamp.

### Frame 9: Summary To Elasticsearch

At batch completion, Orchestrator submits aggregated summaries and normalized signals to Elasticsearch. Full findings depend on indexing mode and governance configuration.

### Frame 10: Evidence View

Evidence Center presents the decision, policy hash, action receipt, HITL status if applicable, and audit trail.

