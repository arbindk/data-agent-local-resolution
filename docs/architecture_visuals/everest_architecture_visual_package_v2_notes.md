# Everest Architecture Visual Package V2 Notes

## Files

- `everest_data_agent_cell_architecture_v2.png`
- `everest_data_agent_cell_sequence_v2.gif`
- `scripts/generate_everest_architecture_visuals_v2.py`

## Purpose

The v2 architecture package is designed to match the style and density of the team-provided architecture references.

It explains the accepted Cuong-reviewed direction:

```text
Portal / Intent
  -> Data Agent Cell Orchestrator
  -> Discovery & Analysis
  -> Guardian Decision
  -> Approved Action Execution or HITL Task
  -> Decision Provenance
  -> Elasticsearch Summary
```

## What The Dense PNG Covers

The dense PNG includes:

- Purpose and scope
- Why no external queue/bus is needed on the enforcement path
- Data Agent Cell internal Orchestrator architecture
- Request intake
- Context builder
- Policy decision client
- Workflow and state
- Routing and coordination
- Response composer
- Function calling layer
- Connector adapters
- Zero-copy accessor
- Discovery and analysis
- DuckDB transient working memory
- Telemetry collector
- Provenance writer
- Guardian Agent decision point
- Upstream systems
- Zero-copy data access explanation
- Zero-copy concurrency model
- Limitations and trade-offs
- Guardian and Data Agent enforcement contract
- End-to-end action lifecycle
- Provenance record contents
- MVS1 autonomous auto-tagging behavior
- Legend, communication lines, and key takeaway

## What The GIF Covers

The GIF has 10 frames:

1. Intent enters through Portal and request envelope.
2. Orchestrator normalizes request context.
3. Generic connector discovers source/object metadata.
4. Zero-copy accessor performs bounded feature extraction.
5. DuckDB stores transient batch signals and scores.
6. Orchestrator requests Guardian decision.
7. Approved action executes or HITL task is created.
8. Guardian records decision provenance.
9. Orchestrator submits batch summary to Elasticsearch.
10. Portal/Evidence shows outcome, provenance, and next task.

## Presentation Recommendation

Use the dense PNG first to explain the full system.

Then use the GIF to walk through the action sequence.

Key message:

Everest MVS1 is autonomous by default only inside policy guardrails. The Orchestrator is the mandatory runtime entry point, Guardian is the mandatory mutation gate, DuckDB is transient working memory, and every decision/action produces provenance.

