# Everest Architecture Alignment

This repository is aligned as a Go-based Everest Data Agent Cell slice. It is not a full rewrite of legacy Zubin services; it keeps the strongest local execution pieces and folds them behind explicit contracts so the Control Plane, Guardian, connectors, evidence, and Portal can evolve independently.

## Current Implementation Map

| Everest target | Current implementation | Notes |
| --- | --- | --- |
| Portal / intent entry | `internal/api` handlers and `web` assets | Portal-originated operations build `ActionRequestEnvelope` before connector dispatch. |
| Control Plane lease path | `internal/api/controlplane_handlers.go` and `internal/platform` | `/v1/controlplane/run-next-lease` is the preferred enterprise execution path. Lease scope is the execution source of truth. |
| Data Agent Cell Orchestrator | `internal/execution/orchestrator.go` | Validates envelope, resolves policy/workflow, loads normalized batch, calls Guardian, builds evidence/provenance. |
| Runtime ports | `internal/execution/ports.go` | Explicit `PolicyPort`, `WorkflowPort`, `WorkStorePort`, `GuardianPort`, and `SearchIndexPort` isolate runtime dependencies. |
| Guardian | `internal/execution/guardian_embedded.go` | Current embedded logic is behind `GuardianPort`; a standalone Guardian can replace it without reopening API handlers. |
| Connector host / connector port | `internal/api/azureblob_connector_port.go` | Azure Blob handler dispatch now goes through `azureBlobConnectorPort` where practical. |
| Azure Blob connector | `internal/connectors/azureblob` | Reference connector for metadata scan, metadata/index tag write-back, archive, retrieve, quarantine, migration, and delete. |
| Policy artifact trust | `internal/policyresolver` | Local Git resolution, SHA-256 verification, Ed25519 verification, active/LKG policy cache. |
| HITL, approvals, and apply RBAC | `internal/platform/approval_store.go`, `internal/api/*actions*`, `internal/api/authorization.go` | Apply-mode mutation requires an allowed enterprise role, Guardian allow, approved Control Plane approval, and connector confirmation before execution. |
| Evidence/provenance | `internal/localstore/duckdbstore`, `internal/provenance`, `internal/platform/store.go` | Manifest, action plan, Guardian decision, provenance, audit, and evidence records are persisted for review/export. |
| Searchable summary index | `internal/searchindex`, `internal/contracts/search_index.go`, `internal/api/search_handlers.go` | Local portal-state index is active by default; Elastic/OpenSearch bulk publishing is enabled through configuration and degrades to queued/local evidence on publish failure. |
| Data Passport MVP | `internal/contracts/data_passport.go`, `internal/api/data_passport_handlers.go`, `internal/portalstate/data_passports.go` | Object/execution passports can be materialized from manifest, evidence, action, policy, workflow, and locality metadata. |
| AI/DSPM advisory layer | `internal/ai`, `internal/dspm`, `internal/api/ai_advisor_handlers.go` | AI can recommend and submit HITL when configured, but does not autonomously apply by default. |

## Canonical Flow

Enterprise execution should use:

1. Platform creates a job and lease.
2. Data Agent calls `/v1/controlplane/run-next-lease`.
3. Agent scans only the lease scope.
4. Orchestrator validates the envelope and resolves trusted policy/workflow state.
5. Guardian evaluates before any connector mutation.
6. Action Executor applies only after Guardian allow, required HITL approval, confirmation, and write-back mode checks.
7. Evidence, receipts, provenance, and status are uploaded back to the Control Plane.

Direct connector endpoints such as `/v1/connectors/azureblob/scan` are retained for demo/dev and diagnostics. They now advertise `/v1/controlplane/run-next-lease` as the preferred enterprise route.

## Fail-Closed Coverage

The current test coverage now includes:

- Guardian unavailable blocks execution.
- Guardian deny produces blocked receipts and no applied action.
- HITL-required decisions produce blocked receipts until approval exists.
- Stale or not-ready policy stops before workflow, Guardian, or connector work.
- Missing approved HITL grant blocks Azure Blob action execution before connector construction.
- Unauthorized apply-mode roles are blocked before Control Plane approval lookup or connector construction.
- Search index publish failure degrades/queues after local manifest/evidence persistence and does not bypass Guardian or fail governed execution.

## Design Direction

Near-term implementation should keep the Azure Blob path as the reference Data Agent Cell and broaden through ports, not copy/paste handlers per connector. New connectors should implement the same connector host pattern and share the common envelope, Guardian, HITL, receipt, and evidence gates.

The embedded Guardian should become a replaceable local service or library with the same `GuardianPort` contract. The policy resolver should continue to fail closed when no verified policy or verified last-known-good policy is available.


## Architecture Pillars

- Zero-copy access: Azure Blob scans enumerate object metadata with connector-native listing APIs and do not download full object bodies by default.
- No source locking for reads: read scans do not acquire Azure Blob leases or source-object locks; normal in-process mutexes are still used for local app state.
- Policy as code: signed compiled JSON policy artifacts are resolved from the local policy repo, SHA-256 checked, Ed25519 verified, cached, and used by Guardian decisions. OPA/Rego/CEL is not currently embedded.
- Searchable summaries: normalized metadata, decisions, action receipts, evidence refs, provenance refs, and Data Passport refs are indexed. Raw customer content and secrets are not indexed by default.

## Search Index / Elasticsearch

Everest target architecture includes Elasticsearch/OpenSearch as the searchable summary/evidence tier. This repo now implements that as a port-backed runtime slice:

- Default mode: `EVEREST_SEARCH_INDEX_MODE=local` writes searchable documents to portal state.
- External mode: `EVEREST_SEARCH_INDEX_MODE=elastic` sends NDJSON `_bulk` requests to `EVEREST_SEARCH_INDEX_ENDPOINT` and mirrors locally first.
- Disabled mode: `EVEREST_SEARCH_INDEX_MODE=disabled` records index-disabled receipts without publishing.
- API: `GET /v1/search` queries local index documents first and falls back to legacy portal arrays when the index is empty.
- Failure behavior: Elastic/OpenSearch failure marks the execution degraded/queued after local manifest/evidence persistence; it never authorizes mutation or skips Guardian.
## Data Passport MVP

The Data Passport MVP is now concrete enough for demo and integration work:

- Schema: `contracts.DataPassport` with subject, execution, classification, risk, policy, workflow, Guardian, action, locality, evidence, provenance, and stable `passport_hash` fields.
- Lifecycle: `observed`, `classified`, `action_planned`, `pending_approval`, `action_applied`, and `blocked`.
- Storage owner: local Portal state for the current MVS, via `portalstate.UpsertDataPassport` and filtered list/get helpers.
- API: `GET /v1/data-passports`, `POST /v1/data-passports`, and `GET /v1/data-passports/{passport_id}`.

A Passport is materialized from local manifest/evidence/action records and does not contain raw object content or secrets.

## Connector Roadmap Order

The next connector order should be:

1. S3-compatible object storage, because it is closest to Azure Blob semantics and validates the connector port abstraction.
2. SMB/NFS for customer edge-site parity with the current Zubin UDE footprint.
3. SharePoint/OneDrive for Microsoft 365 enterprise value and different permission semantics.
4. PostgreSQL/table source for structured-data Data Passport validation.

## Gaps / Questions

- Only Azure Blob is implemented as a full connector; S3, SMB/NFS, SharePoint/OneDrive, database, and other enterprise connectors remain roadmap items.
- Local store package is named `duckdbstore` but currently uses SQLite; either rename the package or move to real DuckDB for large analytical workloads.
- Broader RBAC/identity integration still needs AD/LDAP/Entra-backed actors and role mapping; apply-mode mutations now enforce allowed roles locally.
- Standalone Guardian service boundaries, policy decision signing, and provenance signing strength still need finalization.
- AD/LDAP/Entra integration, SLM/glossary migration, role-tier policy bundles, push vs pull assignment, and connector timelines remain open architecture questions.
