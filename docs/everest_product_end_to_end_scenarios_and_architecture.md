# Everest Product End-To-End Scenarios And Architecture

## Document Purpose

This document explains Everest as a product from both business and technical perspectives.

For the developer-focused backend job lifecycle version of these scenarios, see:

- `docs/everest_backend_job_lifecycle_12_scenarios.md`

It is written for:

- Product managers
- Engineering leads
- Architects
- QA engineers
- Customer success teams
- Security and compliance reviewers
- New developers joining the project
- Non-technical stakeholders who need to understand the product flow

The document explains:

- What Everest is
- What problem it solves
- What each component does
- How the components communicate
- How autonomous actions are taken safely
- How Guardian enforcement works
- How zero-copy data access works
- How AI is involved without bypassing governance
- How evidence and provenance are created
- How end-to-end customer scenarios work

## Product Summary

Everest is an enterprise Data Security Posture Management and data operations platform.

For MVS1, Everest focuses on autonomous, governed data discovery and auto-tagging across customer data stores, starting with Azure Blob Storage.

Everest is designed to:

- Discover data sources and objects.
- Understand metadata, tags, content signals, and risk.
- Recommend or apply approved metadata and protection tags.
- Use AI for classification, explanation, recommendations, and HITL task preparation.
- Enforce policy through Guardian before any mutation.
- Avoid persistent full duplicate copies of customer data by default.
- Record decision provenance for every action.
- Support future connectors through a generic connector contract.

The most important product principle is:

```text
Everest can act autonomously only inside policy guardrails.
If policy does not explicitly allow an action, Everest creates a HITL task or blocks the action.
```

## Core Product Flow

At the highest level, Everest works like this:

```text
User / Schedule / Policy Intent
  -> Everest Portal
  -> Data Agent Cell Orchestrator
  -> Generic Connector Discovery
  -> Zero-Copy Access And Feature Extraction
  -> DuckDB Transient Working Memory
  -> Guardian Decision
  -> Approved Autonomous Action or HITL Task
  -> Decision Provenance
  -> Evidence And Elasticsearch Summary
```

Non-technical explanation:

```text
Everest finds data, understands risk, asks policy whether it can act,
then either acts safely or asks a human to approve.
Every decision is recorded.
```

Technical explanation:

```text
The Portal submits intent. The Orchestrator creates a request envelope,
coordinates discovery and analysis, calls Guardian for a decision, dispatches
only approved actions through connector adapters, writes provenance, and
submits normalized summary signals for search and reporting.
```

## Product Principles

### 1. Orchestrator-First

All scans, reads, AI recommendations, HITL requests, and actions must go through the Data Agent Cell Orchestrator.

The Portal does not directly mutate data.

AI does not directly mutate data.

Workbench does not directly bypass governance.

### 2. Guardian-Enforced

Guardian is the mandatory policy enforcement point before mutation.

No action such as tagging, masking, quarantine, move, archive, delete, or write-back can execute unless Guardian returns an executable decision.

Guardian can return:

- Allow
- Deny
- Modify
- Allow with obligations
- HITL required

### 3. Autonomous By Default, Within Guardrails

Everest should reduce manual work.

If policy explicitly allows a safe action, Everest can execute automatically.

If policy requires review, Everest creates a HITL task.

The long-term product goal is to reduce HITL volume as policies, confidence, and approvals mature.

### 4. Zero-Copy By Default

Everest avoids persistent full duplicate copies of customer data by default.

Allowed default patterns:

- Metadata read
- Object reference
- Handle/session
- Stream
- Range read
- Cursor
- Bounded preview
- Ephemeral feature extraction

Full export or download requires:

- Guardian approval
- Audit record
- Provenance linkage
- Clear user attribution

### 5. Generic Connector Architecture

Azure Blob is the first complete connector, but Everest must not be Azure Blob-specific.

The same architecture should support:

- Azure Blob Storage
- S3 or object stores
- SMB/NFS file shares
- Databases
- SaaS repositories
- APIs
- Streams
- Customer-built connectors

### 6. Decision Provenance

Every decision must be explainable and auditable.

Everest records:

- Who requested the action
- What data was involved
- Which policy bundle was used
- Which rules were evaluated
- What Guardian decided
- What obligations were returned
- What action was executed or blocked
- What result occurred
- When it happened
- Which system/component performed it

## Main Personas

| Persona | What They Need | Everest Value |
|---|---|---|
| Business Owner | Know what sensitive data exists and whether it is protected | Dashboard, recommendations, evidence |
| Data Steward | Fix missing metadata, tags, and ownership | Auto-tagging, HITL tasks, workbench |
| Security Officer | Ensure risky data is governed | Guardian decisions, policy enforcement |
| Compliance Reviewer | Prove what happened and why | Evidence Center, provenance records |
| Platform Admin | Configure connectors, AI, deployment modes | Source Inventory, Connector Catalog, AI Tools |
| Developer | Add connectors and runtime capabilities | Generic connector contract |
| QA Engineer | Validate flows and invariants | Scenario tests and enforcement tests |

## Component Overview

### 1. Everest Portal

The Portal is the user-facing control surface.

It should allow users to:

- Add or select data sources.
- View dashboard posture.
- Start autonomous scans.
- Review Guardian decisions.
- Review HITL tasks.
- Inspect evidence and provenance.
- Use advanced Workbench only when needed.

The Portal should not:

- Directly execute mutations.
- Expose raw secrets after save.
- Persist unrestricted raw customer content.
- Bypass Guardian.

Flow:

```text
User action in Portal
  -> Request intent
  -> Data Agent Cell Orchestrator
```

### 2. Data Agent Cell

The Data Agent Cell is a colocated runtime near customer data.

It owns the local execution flow.

It reduces data movement and supports private-network or air-gapped operation.

Main responsibilities:

- Receive request envelopes.
- Coordinate connector discovery.
- Perform bounded analysis.
- Use DuckDB as transient batch working memory.
- Call Guardian for decisions.
- Execute approved actions.
- Create HITL tasks when needed.
- Write telemetry and provenance.

### 3. Orchestrator

The Orchestrator is the single runtime control plane inside the Data Agent Cell.

It is the most important component in the architecture.

The Orchestrator owns:

- Request intake
- Context building
- Connector routing
- Workflow and state
- Leases
- Checkpoints
- Retries
- Guardian decision calls
- Action dispatch
- HITL task creation
- Response composition
- Batch completion summary

Flow:

```text
Portal / Intent
  -> Orchestrator
  -> Context Builder
  -> Connector Discovery
  -> Feature Extraction
  -> Guardian Decision
  -> Execute or HITL
  -> Provenance
```

### 4. Request Envelope

The request envelope standardizes every operation.

It prevents each screen or API from inventing its own execution model.

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

Example:

```text
Request Envelope
  actor: data-steward
  connector: azureblob
  operation: auto_tag
  scope: container=finance, prefix=payroll/
  policy_context: MVS1 autonomous tagging
```

### 5. Context Builder

The Context Builder prepares the information Guardian and downstream components need.

It resolves:

- Actor identity
- Tenant
- Role
- Source
- Connector capabilities
- Object references
- Existing metadata
- Existing tags
- Data region
- Policy context
- Runtime mode
- Correlation ID

Flow:

```text
Request Envelope
  -> Context Builder
  -> Canonical Action Context
```

### 6. Generic Connector Adapter

The connector adapter gives Everest a consistent way to talk to different data systems.

Azure Blob is the first implementation.

The connector adapter should expose:

- Discover sources
- Discover containers/buckets/shares/tables
- List objects
- Read metadata
- Read tags
- Get object reference
- Stream/range read
- Apply metadata/tags
- Return action receipt
- Report capabilities

Flow:

```text
Orchestrator
  -> Generic Connector Adapter
  -> Azure Blob / File Share / Database / SaaS / API
```

### 7. Zero-Copy Accessor

The Zero-Copy Accessor controls how Everest reads data.

It avoids persistent full duplicate copies.

It supports:

- Read by reference
- Stream
- Range read
- Cursor
- Bounded preview
- Ephemeral feature extraction

Example:

```text
Object reference
  -> range read 0-64 KB
  -> extract classification features
  -> discard raw content
  -> keep only normalized signals
```

### 8. Discovery And Analysis

Discovery and analysis determines what exists and what risk exists.

It uses:

- Metadata inference
- Existing tags
- File/object name
- Path/prefix
- Size/type/tier
- Content type
- Bounded content features
- Keyword patterns
- Entity extraction
- Risk scoring
- AI-assisted classification where allowed

Output:

- Classification
- Risk score
- Suggested metadata
- Suggested tags
- HITL recommendation
- Evidence signals

### 9. DuckDB Transient Working Memory

DuckDB is used as local working memory for a batch.

It should be treated as transient, not a long-term raw content store.

Allowed data:

- Normalized metadata
- Object references
- Batch features
- Feature vectors
- Classification scores
- Aggregation tables
- Checkpoint-supporting summary data

Not allowed by default:

- Persistent full raw files
- Unbounded sensitive content
- Long-term duplicate object copies

Flow:

```text
Connector metadata + bounded features
  -> DuckDB transient tables
  -> scoring and aggregation
  -> normalized signals
  -> reset or expire per batch policy
```

### 10. Guardian Agent

Guardian is the policy enforcement point.

Guardian decides whether Everest can take action.

Guardian evaluates:

- Actor
- Role
- Source
- Operation
- Data classification
- Risk score
- Policy bundle
- Data region
- Target region
- Action type
- Existing approvals
- Obligations

Guardian returns:

- Allow
- Deny
- Modify
- HITL required
- Obligations
- Reasoning
- Policy bundle version
- Policy bundle hash

Flow:

```text
Orchestrator
  -> Guardian decision request
  -> Guardian evaluates policy
  -> Guardian returns decision + obligations
  -> Orchestrator executes only if allowed
```

### 11. Action Executor

The Action Executor performs approved operations.

It never executes directly from Portal or AI.

It executes only after Guardian returns an executable decision.

Examples:

- Apply metadata
- Apply index tags
- Apply dd_protection_flags
- Mask/protect if supported
- Quarantine if approved
- Archive if approved
- Delete only with strict approval and confirmation

Flow:

```text
Guardian Allow / Modify
  -> Orchestrator
  -> Action Executor
  -> Connector Adapter
  -> Data Source
  -> Action Receipt
```

### 12. HITL Task Creator

HITL means Human-In-The-Loop.

If Guardian requires human review, Everest creates a task instead of directly executing.

HITL is required when:

- Policy confidence is not high enough.
- Data is high risk.
- Action is destructive.
- Action crosses boundary or region.
- Obligation requires human approval.
- AI recommendation is not allowed to auto-apply.

Flow:

```text
Guardian returns HITL
  -> Orchestrator creates task
  -> User reviews task
  -> User approves/rejects
  -> Orchestrator asks Guardian again if needed
  -> Approved action executes or remains blocked
```

### 13. AI Recommendation Layer

AI helps users and the Orchestrator understand data and actions.

AI can:

- Summarize evidence
- Classify content
- Extract features
- Recommend metadata
- Recommend tags
- Explain policy decisions
- Prefill HITL tasks
- Generate action plans

AI cannot:

- Directly mutate customer data.
- Bypass Guardian.
- Override policy.
- Hide provenance.
- Execute delete/archive/quarantine/move by itself.

Flow:

```text
Normalized signals + evidence context
  -> AI recommendation
  -> Orchestrator
  -> Guardian decision
  -> Approved action or HITL task
```

### 14. Provenance Writer

Provenance explains what happened and why.

Guardian owns decision provenance.

The Orchestrator and Action Executor contribute action receipts and runtime facts.

Provenance records include:

- Request ID
- Decision ID
- Actor
- Tenant
- Source
- Connector
- Object reference
- Policy bundle version
- Policy bundle hash
- Rules applied
- Reasoning
- Decision
- Obligations
- HITL status
- Action receipt
- Timestamp
- Signature or hash

Flow:

```text
Guardian Decision + Action Receipt
  -> Provenance Record
  -> Evidence Center
  -> Elasticsearch / Audit Export
```

### 15. Elasticsearch Summary

Elasticsearch is used for summary/search/reporting.

At batch completion, the Orchestrator submits:

- Aggregated summary results
- Normalized signals
- Risk distribution
- Classification counts
- Policy/action outcomes
- Decision references
- Evidence/provenance references

Full findings may be indexed depending on governance configuration and indexing mode.

Flow:

```text
Batch complete
  -> Orchestrator aggregates results
  -> Summary and normalized signals
  -> Elasticsearch
  -> Dashboard / reports / search
```

## End-To-End Scenario 1: New Customer Connects Azure Blob

### Business Story

A customer wants Everest to discover and classify data in Azure Blob Storage without exposing raw secrets to business users.

### User Flow

```text
User opens Source Setup
  -> selects Azure Blob connector
  -> enters account name and approved access details
  -> saves source
  -> discovers containers
  -> selects scope
```

### System Flow

```text
Portal
  -> Orchestrator request envelope
  -> Connector capability check
  -> Secret/access reference stored securely
  -> Connector lists containers
  -> Orchestrator returns discovered scopes
  -> Portal shows dropdowns/suggestions
```

### What User Sees

- Saved Azure Blob source.
- Containers as dropdown choices.
- Access status.
- No raw secret displayed after save.
- Recommended next step: run autonomous scan.

### What System Achieves

- Source is registered.
- Connector capability is known.
- Access is stored as a secret/reference.
- Containers are discoverable for future scans.

## End-To-End Scenario 2: Autonomous MVS1 Auto-Tagging

### Business Story

Everest should automatically tag safe objects when policy allows it, without requiring a human for every object.

### User Flow

```text
User selects source and scope
  -> starts autonomous MVS1 scan
  -> reviews dashboard
  -> sees auto-applied tags and HITL tasks
```

### System Flow

```text
Intent
  -> Orchestrator creates request envelope
  -> Connector discovers objects
  -> Zero-Copy Accessor reads metadata and bounded features
  -> DuckDB stores transient batch signals
  -> Discovery & Analysis scores objects
  -> Orchestrator calls Guardian
  -> Guardian allows safe auto-tagging
  -> Action Executor applies dd_protection_flags
  -> Provenance record created
  -> Summary sent to Elasticsearch
```

### What User Sees

- Objects scanned.
- Tags applied automatically where allowed.
- HITL tasks only for exceptions.
- Evidence and decision records.

### What System Achieves

- Autonomous productivity.
- Policy enforcement.
- No uncontrolled data copy.
- Decision provenance.

## End-To-End Scenario 3: Sensitive Payroll File Requires HITL

### Business Story

Everest finds a file that appears sensitive, such as a payroll or salary file. Policy does not allow automatic action without review.

### User Flow

```text
User starts scan
  -> dashboard shows sensitive finding
  -> Action Center shows HITL task
  -> reviewer approves or rejects
```

### System Flow

```text
Connector lists object
  -> Zero-Copy Accessor extracts bounded features
  -> DSPM marks object as sensitive
  -> Orchestrator sends context to Guardian
  -> Guardian returns HITL required
  -> Orchestrator creates HITL task
  -> Guardian records decision provenance
```

### What User Sees

- Sensitive finding.
- HITL task with reason.
- Required reviewer.
- Policy reason.
- Evidence trail.

### What System Achieves

- Risky action is not silently applied.
- Human review is requested.
- Governance remains auditable.

## End-To-End Scenario 4: Safe Metadata Completion

### Business Story

Many files are missing owner, classification, or retention metadata. Everest can fill safe metadata automatically when policy allows.

### User Flow

```text
User runs scan
  -> Everest detects missing metadata
  -> Everest recommends metadata
  -> Guardian allows auto-apply
  -> metadata is applied
```

### System Flow

```text
Object metadata discovered
  -> missing fields identified
  -> AI/DSPM recommends metadata
  -> Orchestrator asks Guardian
  -> Guardian allows safe metadata update
  -> Action Executor applies metadata through connector
  -> Action receipt created
```

### What User Sees

- Metadata completion count.
- Objects updated.
- Evidence record.
- Policy version/hash.

### What System Achieves

- Data quality improves.
- Manual cleanup decreases.
- Every update is governed.

## End-To-End Scenario 5: AI Recommends Tags But Cannot Execute

### Business Story

AI can help determine likely classification and protection tags, but AI must not directly mutate customer data.

### User Flow

```text
User opens recommendation
  -> AI suggests tags
  -> user sees explanation
  -> action goes to Guardian
  -> allowed action executes or HITL task is created
```

### System Flow

```text
Normalized evidence
  -> AI recommendation
  -> Orchestrator receives recommendation
  -> Guardian evaluates policy
  -> Action Executor runs only if Guardian allows
```

### What User Sees

- AI recommendation.
- Confidence/rationale.
- HITL status if required.
- No direct AI execution button that bypasses policy.

### What System Achieves

- AI improves efficiency.
- Guardian remains mandatory.
- Decision is explainable.

## End-To-End Scenario 6: Delete Attempt Is Blocked

### Business Story

A user or automated policy tries to delete an object. Delete is destructive and should require strict policy and confirmation.

### User Flow

```text
Delete requested
  -> Guardian blocks or requires HITL
  -> user sees reason
  -> evidence records blocked attempt
```

### System Flow

```text
Delete intent
  -> Orchestrator request envelope
  -> Guardian evaluates destructive-action policy
  -> Guardian returns deny or HITL
  -> Orchestrator does not call executor
  -> Provenance records blocked attempt
```

### What User Sees

- Delete not executed.
- Clear reason.
- Required approval if applicable.
- Audit record.

### What System Achieves

- No accidental destructive mutation.
- Blocked attempts are still auditable.

## End-To-End Scenario 7: Full Content Export

### Business Story

A user wants to download or export full object content for investigation.

This is not default zero-copy behavior and must be governed.

### User Flow

```text
User requests export
  -> Guardian approval required
  -> if allowed, export proceeds
  -> evidence records export
```

### System Flow

```text
Export request
  -> Orchestrator builds context
  -> Guardian evaluates export policy
  -> Guardian allow/HITL/deny
  -> if allowed, connector streams content
  -> audit and provenance record export
```

### What User Sees

- Approval requirement.
- Export reason.
- Export status.
- Evidence record.

### What System Achieves

- Zero-copy default remains intact.
- Explicit export is controlled and auditable.

## End-To-End Scenario 8: Air-Gapped Environment

### Business Story

Some customers cannot connect to internet-based services.

Everest must still operate with local policy, local AI, local connectors, and local evidence.

### User Flow

```text
Admin selects air-gapped deployment mode
  -> local connectors and AI providers are configured
  -> scans and decisions run locally
  -> evidence export is generated offline
```

### System Flow

```text
Portal
  -> Data Agent Cell
  -> local connector
  -> local Guardian policy bundle
  -> local AI provider if configured
  -> local DuckDB working memory
  -> local evidence/provenance
```

### What User Sees

- Air-gapped deployment mode.
- Local provider status.
- Evidence available for export.
- No required public internet dependency.

### What System Achieves

- Sovereign deployment support.
- Policy enforcement remains local.
- Data remains within customer boundary.

## End-To-End Scenario 9: New Connector Added

### Business Story

The customer wants Everest to support a future data source beyond Azure Blob.

### User Flow

```text
Admin registers connector
  -> Connector Catalog validates manifest
  -> connector exposes capabilities
  -> Orchestrator uses same flow
```

### System Flow

```text
Connector manifest
  -> Connector Catalog
  -> capability model
  -> Orchestrator routing
  -> Connector Adapter
  -> Generic object references
  -> Guardian decision
  -> Provenance
```

### What User Sees

- New connector available.
- Supported actions.
- Deployment mode.
- Runtime status.

### What System Achieves

- Multi-store architecture.
- No need to redesign Portal for each connector.
- Common governance and evidence model.

## End-To-End Scenario 10: Policy Update

### Business Story

Security updates policy rules. Data agents must enforce the new version consistently.

### User Flow

```text
Policy team updates policy
  -> policy is compiled and signed
  -> Data Agent pulls bundle
  -> future decisions use new version
```

### System Flow

```text
Policy authoring
  -> compile Cedar/Python policy bundle
  -> sign/hash bundle
  -> distribute bundle
  -> Data Agent verifies bundle
  -> Guardian uses bundle for decisions
  -> provenance records version/hash
```

### What User Sees

- Policy version.
- Decision reason.
- Policy hash in evidence.

### What System Achieves

- Versioned governance.
- Repeatable decisions.
- Auditability.

## End-To-End Scenario 11: Batch Failure And Recovery

### Business Story

A scan is interrupted due to network failure or source throttling. Everest should recover without losing control state.

### User Flow

```text
Scan starts
  -> failure occurs
  -> dashboard shows failed/paused state
  -> user resumes or system retries
```

### System Flow

```text
Orchestrator state machine
  -> lease/checkpoint created
  -> connector processes cursor/ranges
  -> failure recorded
  -> retry/backoff applied
  -> resume from checkpoint
  -> provenance records partial/completed state
```

### What User Sees

- Clear status.
- Retry/resume option.
- Evidence of failure reason.
- No duplicated uncontrolled actions.

### What System Achieves

- Durable local state.
- No external queue dependency.
- Controlled recovery.

## End-To-End Scenario 12: Evidence Review

### Business Story

Compliance wants to know what happened, why it happened, and who approved it.

### User Flow

```text
Reviewer opens Evidence Center
  -> selects execution or object
  -> reviews decision provenance
  -> exports evidence package
```

### System Flow

```text
Request + Guardian decision + action receipt
  -> Provenance Writer
  -> Evidence Center
  -> Elasticsearch summary
  -> export package
```

### What User Sees

- Actor.
- Source/object.
- Policy version/hash.
- Decision.
- Reasoning.
- Obligations.
- Action receipt.
- HITL approval if any.
- Timestamp.

### What System Achieves

- Audit-ready proof.
- Compliance traceability.
- Investigation support.

## Detailed MVS1 Auto-Tagging Flow

This is the primary product path.

```text
1. User or schedule starts MVS1 auto-tagging.
2. Portal submits intent to Orchestrator.
3. Orchestrator creates ActionRequestEnvelope.
4. Context Builder resolves actor, source, connector, policy context.
5. Generic Connector discovers containers/objects.
6. Zero-Copy Accessor reads metadata and bounded content features.
7. DuckDB stores transient normalized signals.
8. Discovery & Analysis scores objects.
9. AI may assist with classification or tag recommendations.
10. Orchestrator sends decision request to Guardian.
11. Guardian evaluates policy bundle.
12. Guardian returns allow, deny, modify, obligations, or HITL.
13. If allow, Action Executor applies approved tags.
14. If HITL, task is created for reviewer.
15. Guardian records decision provenance.
16. Action Executor writes action receipt.
17. Orchestrator aggregates batch summary.
18. Summary and normalized signals go to Elasticsearch.
19. Portal dashboard and Evidence Center update.
```

## Technical Flow Diagram

```text
Portal / Intent
  |
  v
ActionRequestEnvelope
  |
  v
Data Agent Cell Orchestrator
  |
  +--> Context Builder
  |
  +--> Generic Connector Adapter
  |       |
  |       v
  |     Data Source
  |
  +--> Zero-Copy Accessor
  |       |
  |       v
  |     Bounded Feature Extraction
  |
  +--> DuckDB Transient Working Memory
  |
  +--> DSPM / AI Recommendation
  |
  v
Guardian Decision
  |
  +--> Deny -> Provenance -> Portal
  |
  +--> HITL -> Task -> Provenance -> Portal
  |
  +--> Allow / Modify
          |
          v
      Action Executor
          |
          v
      Connector Adapter
          |
          v
      Apply Approved Metadata / Tags
          |
          v
      Action Receipt
          |
          v
      Provenance + Elasticsearch Summary
```

## Non-Technical Explanation Of The Same Flow

```text
Everest receives a request.
Everest checks what data exists.
Everest looks at enough information to understand risk.
Everest asks Guardian what is allowed.
If allowed, Everest acts.
If not allowed, Everest asks a human or blocks.
Everest records everything.
```

## What Makes Everest Enterprise-Ready

Everest becomes enterprise-ready when these statements are true:

- All operations go through Orchestrator.
- All mutations require Guardian.
- AI cannot bypass policy.
- Workbench cannot bypass policy.
- Secrets are hidden after save.
- Raw full content is not persisted by default.
- Every action attempt is recorded.
- Every decision has policy version/hash.
- Every connector uses the same contract.
- Every customer deployment mode is supported.

## Required QA Coverage

QA should validate:

1. Source save without secret exposure.
2. Container discovery.
3. Object listing.
4. Metadata-only scan.
5. Bounded feature extraction.
6. DuckDB transient state.
7. AI recommendation without direct mutation.
8. Guardian allow auto-tags.
9. Guardian deny blocks mutation.
10. Guardian HITL creates task.
11. Action receipt is created.
12. Provenance includes policy version/hash.
13. Evidence export works.
14. Full content export requires approval.
15. Connector contract works for Azure Blob and future connector mock.
16. Batch failure resumes from checkpoint.
17. Air-gapped mode works without external dependency.

## Glossary

| Term | Meaning |
|---|---|
| Data Agent Cell | Local runtime near customer data |
| Orchestrator | Mandatory control plane inside Data Agent Cell |
| Guardian | Policy enforcement and decision provenance component |
| HITL | Human-In-The-Loop review task |
| Zero-copy | No persistent full duplicate copy by default |
| Connector Adapter | Common interface to data stores |
| DuckDB Working Memory | Transient local batch analysis store |
| Provenance | Tamper-evident record of decision and action |
| Action Receipt | Result record from an approved executor operation |
| Request Envelope | Standard request model for all operations |
| MVS1 | First product slice focused on autonomous DSPM auto-tagging |

## Final Product Message

Everest is an autonomous, governed data operations platform.

It discovers and understands enterprise data, recommends or applies approved protection actions, routes exceptions to humans, and proves every decision.

For MVS1, Everest starts with autonomous DSPM auto-tagging:

```text
Discover -> Analyze -> Decide -> Act or HITL -> Prove
```

The architecture is designed so customers can trust autonomy:

```text
Autonomous where policy allows.
Human review where policy requires.
Provenance for every decision.
```
