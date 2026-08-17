# Item 13 Next Implementation Steps

1. Copy the `contracts/` folder into the repo root.
2. Copy the `demo/` folder into the repo root, preserving existing policy resolver demo files if needed.
3. Add `/v1/execution/start`, `/v1/execution/{id}`, `/v1/execution/{id}/summary`, `/v1/execution/{id}/provenance`, and `/v1/cache/status`.
4. Keep the existing local policy resolver as `policycache` inside Agent Core.
5. Add workflow resolver for `workflow-mvs-linear-v1.json`.
6. Add orchestrator that runs the MVS linear workflow using mock clients behind real contracts.
7. Generate `batch_summary.json`, `provenance.json`, and `signals_index.json` under `data/executions/<execution_id>/`.

Demo order:
- Standard Tier execution
- Rolled-up Tier execution with high-risk exception signal
- Guardian failure/no write-back
- Git unavailable/cache reuse
