# Everest Enterprise Screen Guide

Version: 3.0  
Scope: Official screen-level guide for Everest Enterprise Edition.

## 1. Enterprise Dashboard

Purpose:

The dashboard provides executive and operator posture for governed Azure Blob operations.

Required information:

- total sources, containers, objects, and scanned records
- sensitive-data count by severity
- exposure and residency risks
- active scan jobs and checkpoints
- pending approvals
- recommended next actions
- evidence health and export readiness
- secret provider posture without values
- enterprise posture analytics from `/v1/analytics/enterprise-posture`

Expected behavior:

- Dashboard updates after source discovery, scan, Workbench action, AI recommendation, approval, and evidence refresh.
- Dashboard displays sensitive findings, high-risk evidence, async records, top risks, and recommended next actions.
- Dashboard must prioritize user action: scan, approve, remediate, export evidence, or review risk.
- Dashboard must not expose raw secrets or unrestricted file content.

## 2. Source Inventory

Purpose:

Source Inventory manages enterprise data sources and account access using user-friendly fields.

Fields:

- storage account
- cloud suffix
- container or selected containers
- prefix
- data store ID
- action mode
- write-back mode
- credential status

Rules:

- Users must not be asked to type a raw connection string in the normal flow.
- Backend creates and stores the account access string.
- Saved sources must appear in source dropdowns across Workbench, scans, jobs, and action workflows.
- Source-specific credential isolation must be preserved.

## 3. Architecture Flow

Purpose:

Architecture Flow explains enterprise product behavior without requiring implementation history.

Views:

- complete architecture
- component communication
- positive operational flow
- negative/restricted flow

Expected behavior:

- The flow must explain portal, agent API, secret provider, connector, working store, DSPM, AI, Guardian/HITL, action execution, and evidence.
- The flow must support cloud-connected, private-network, hybrid, and air-gapped deployment discussions.

## 4. Azure Blob Workbench

Purpose:

Workbench lets authorized users inspect and operate on Azure Blob objects through governed, preview-first controls.

Operations:

- discover containers
- list blobs
- view object
- create/update object
- download object
- set metadata
- set index tags
- copy
- move-prep
- tier
- rehydrate
- delete

Rules:

- Saved source dropdown drives backend source access.
- Object/container/file ID fields should prefer dropdowns and suggestions.
- Preview is the default for mutating operations.
- Restricted/destructive operations require confirmation and approval.
- Raw support payloads remain hidden unless diagnostics visibility is enabled.

## 5. Operations Assistant

Purpose:

Operations Assistant guides a user through a controlled validation or remediation workflow.

Expected steps:

- prepare controlled run
- run preview
- apply approved safe steps in controlled container
- generate evidence
- preview cleanup
- cleanup controlled object
- reset assistant state

Rules:

- The assistant must not imply throwaway or local-only behavior.
- It must use selected saved source and source-specific credential access.
- It must return to preview-first posture after apply.
- It must record evidence and audit events.

## 6. Scans And Jobs

Purpose:

Scans and Jobs manages governed scan execution and async checkpointed jobs.

Expected behavior:

- Metadata scan writes normalized records.
- Async scan job can be queued, resumed, paused, cancelled, completed, or failed.
- Checkpoints show container, prefix, page marker, pages scanned, and objects scanned.
- Job state persists to configured checkpoint path.
- Resume starts the Azure Blob paging worker for the selected source.
- Each scanned page updates portal state, checkpoint state, and evidence.
- Worker heartbeat, lease ownership, retry count, and throttle count are visible for operational monitoring.
- Runtime card shows secret provider posture without secret values.
- Runtime card shows authorization mode.
- Portal API calls include enterprise actor, role, and tenant headers.

## 7. DSPM And Content Intelligence

Purpose:

DSPM identifies sensitive data, risky metadata, exposed objects, and governance violations.

Expected behavior:

- Keyword and pattern matching support customer-defined checks.
- Entity mapping covers PII, PHI, PCI, secrets, legal, and domain-specific entities.
- Model profiles can include built-in rules, local BERT/domain NER, or customer-approved AI services.
- Findings produce recommended metadata, index tags, actions, and evidence.

## 8. AI Tools

Purpose:

AI Tools configure customer-approved AI providers and runtime behavior.

Expected behavior:

- Provider catalog supports cloud-connected, private-network, and air-gapped options.
- AI recommendations must be evidence-grounded.
- Autonomous apply is blocked unless explicitly governed.
- HITL submission is available for restricted actions.
- Prompt/raw payload visibility must follow diagnostics policy.

## 9. Action Center

Purpose:

Action Center manages remediation, approval, and execution posture.

Expected behavior:

- Action canvas shows operation, source, object, policy, rationale, approval state, and execution state.
- Restricted actions route to HITL.
- Approved safe actions can execute against the selected source.
- Blocked actions must show clear policy reason.

## 10. Evidence Center

Purpose:

Evidence Center provides audit-grade proof for enterprise operations.

Expected evidence:

- source ID and data store
- actor and role
- object/container/prefix
- scan/job ID
- policy decision
- AI recommendation if used
- HITL approval if required
- action mode
- operation status
- timestamp
- action receipt or block reason

## 11. Diagnostics

Purpose:

Diagnostics supports authorized troubleshooting without exposing data by default.

Rules:

- Raw payloads are hidden unless explicitly enabled.
- Notifications appear in top-right notification center.
- Users can clear notifications.
- Diagnostics must identify provider posture, runtime health, and contract checks without revealing secrets.
