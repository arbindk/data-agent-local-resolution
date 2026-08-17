# Everest Enterprise Quick Reference

Version: 3.0

## Product Purpose

Everest Enterprise governs customer storage operations. It discovers data, identifies sensitive content, recommends action, routes restricted operations through approval, applies approved changes, and records evidence.

## Primary User Workflows

1. Add or select Azure Blob source.
2. Discover containers and select scope.
3. Run metadata or content scan.
4. Review dashboard risk posture.
5. Review DSPM findings.
6. Ask AI for recommendations.
7. Preview action in Workbench or Action Center.
8. Submit HITL approval if required.
9. Apply approved action.
10. Review evidence and audit records.

## Core Screens

| Screen | Purpose |
| --- | --- |
| Enterprise Dashboard | Operational risk posture, next actions, evidence health. |
| Source Inventory | Source setup, saved sources, source-specific account access. |
| Architecture Flow | Product architecture and operational flow. |
| Azure Blob Workbench | Governed object inspection and action preview/apply. |
| Scans & Jobs | Metadata scans and async checkpointed scan jobs. |
| Evidence Center | Audit-grade proof and export readiness. |
| Guardian Agent | Decision status and safety controls. |
| Action Center | Remediation actions and HITL approvals. |
| Policy Governance | Policy decision and trust posture. |
| AI Tools | AI provider configuration and runtime validation. |
| Diagnostics | Authorized support and runtime health. |

## Required Enterprise Behaviors

- No raw credential display in normal UI.
- Dynamic dropdowns wherever saved values exist.
- Preview-first actions for all mutations.
- HITL for restricted/destructive/cross-border/high-risk operations.
- Source-specific credential isolation.
- Evidence record for every governed workflow.
- Dashboard suggestions based on current risk and job state.
- Raw payload visibility restricted to diagnostics/support mode.

## Sensitive-Data Scenario Families

- PII exposure
- PHI misplaced data
- PCI/payment patterns
- API keys/secrets
- legal hold and retention
- ransomware indicators
- public container exposure
- cross-border copy/move
- archive/rehydrate risk
- multi-source isolation
- air-gapped operation
- read-only secret provider operation

## Developer Orientation

| Area | Files |
| --- | --- |
| API routes and server wiring | `internal/api/` |
| Azure Blob connector config/actions | `internal/connectors/azureblob/` |
| Async scan job contract | `internal/controlplane/azureblob_scan_jobs.go` |
| Secret provider abstraction | `internal/secretstore/` |
| Portal UI | `web/index.html` |
| Portal state and evidence records | `internal/portalstate/` |
| DSPM/content intelligence | `internal/dspm/` |
| AI provider contracts | `internal/ai/` |

## Validation Baseline

Before handoff, validate:

- full Go test suite
- portal JavaScript parse
- Azure Blob source save/test/discover
- Workbench preview/apply on controlled container
- DSPM findings on sensitive files
- AI recommendations
- HITL approval path
- evidence and audit records
- async scan job queue/resume/pause/cancel with worker lease, heartbeat, retry, page scanning, and normalized records
- enterprise posture endpoint metrics and recommendations
- source-specific credential behavior
- authorization audit/enforce mode with enterprise identity headers
- day/night UI readability
