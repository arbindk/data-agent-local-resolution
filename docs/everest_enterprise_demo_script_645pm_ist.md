# Everest Azure Blob Enterprise Demo Script - 6:45 PM IST

## Purpose

This script is for presenting Everest as an enterprise Azure Blob data operations product. The flow should show how a customer can connect Azure Blob Storage, inspect containers and objects, run governed scans, detect sensitive content, receive AI-assisted recommendations, route risky actions through human approval, and produce auditable evidence.

Do not expose raw secrets, unrestricted raw payloads, or implementation-only terminology during the customer walkthrough.

## Demo Outcomes

By the end of the session, the customer should understand that Everest can:

- Manage saved Azure Blob sources without asking business users for a raw connection string.
- Discover containers and objects through guided dropdowns and suggestions.
- Scan real enterprise content for sensitive data, risky permissions, policy gaps, and operational posture.
- Support lightweight workbench actions and long-running async scan jobs.
- Use AI to summarize findings, recommend next actions, and prepare governed actions.
- Keep destructive or high-risk actions behind preview, policy, and HITL approval.
- Produce audit-ready evidence for security, compliance, and operations teams.
- Run across cloud-connected, private-network, and air-gapped deployment modes.

## Pre-Session Checklist

- Start the Everest API and portal.
- Confirm the portal opens at the expected environment URL.
- Confirm the Azure Blob source is saved in the portal.
- Confirm the saved source uses simplified fields, not visible raw connection-string entry.
- Confirm at least one test Azure Blob container is available.
- Confirm the test storage account has representative content:
  - Normal business documents.
  - PII-like content.
  - PHI-like content if appropriate for the customer.
  - Secret/token-like text.
  - Large files or mixed extensions.
  - Public or broad-access examples if available.
- Keep preview/dry-run enabled for any action that can modify data.
- Confirm Action Center is available for approvals.
- Confirm Evidence Center has at least one generated or available evidence record.
- Confirm theme toggle works in both day and night mode.

## Primary Demo Flow

### 1. Enterprise Dashboard

Open the Enterprise Dashboard.

Expected result:

- The dashboard shows enterprise posture, source count, object count, sensitive findings, high-risk evidence, pending approvals, and AI recommendations.
- The dashboard does not look like a sample-only page.
- Risk indicators are easy to explain to security, compliance, and storage operations stakeholders.

Positioning:

- Everest gives a single control plane for Azure Blob operations, data risk, governed remediation, and audit evidence.
- The dashboard is the executive and operator starting point.

### 2. Source Inventory

Open Source Inventory and review saved Azure Blob sources.

Expected result:

- Existing Azure Blob sources are listed.
- Users can select saved sources from dropdowns.
- Source configuration is represented through business-friendly fields.
- Raw secrets are not visible.
- Source status shows whether the source is ready, missing access, or needs attention.

Positioning:

- A business or operations user should not need to understand a raw connection string.
- Everest stores source access securely and uses it only through governed backend operations.

### 3. Azure Blob Workbench

Open Azure Blob Workbench and choose the saved Azure Blob source.

Expected result:

- The page shows saved Azure Blob sources.
- Add, modify, and remove actions are available for saved sources.
- The workbench status shows source access using readable labels such as saved source access.
- Text does not overlap in the Auth, Container, or Status fields.
- Only relevant controls are visible for the selected operation.

Run Discover Containers.

Expected result:

- Containers are discovered from Azure Blob.
- Container choices appear as dropdown or multi-select options.
- Selected containers are visible as selected values or chips before the next action is executed.

Run List Blobs for one or more containers.

Expected result:

- Objects are listed with name, container, size, tier, content type, modified time, and risk context where available.
- Object IDs and container names can be selected from existing values instead of typed manually.
- Raw object content is not shown unless explicitly enabled for a controlled troubleshooting view.

Positioning:

- Everest guides the user toward valid actions and removes unnecessary manual input.
- Preview-first behavior allows safe validation before any object-level change.

### 4. Sensitive Content And DSPM Scan

Open Scans & Jobs and start or review an Azure Blob scan.

Use representative content categories:

- PII file.
- PHI file.
- Credentials or token-like content.
- Unclassified business document.
- Large file or mixed-format object.
- Object with broad access or policy concern.

Expected result:

- Scan jobs show a clear lifecycle: queued, running, paused, completed, failed, or canceled.
- The execution lifecycle uses meaningful labels and not placeholder wording.
- Results distinguish normal files from sensitive or risky files.
- Keyword pattern matching, entity detection, and model-assisted classification are represented in findings where applicable.
- Tiered scan behavior is clear: fast metadata/content checks first, deeper checks for higher-risk files.

Positioning:

- Everest supports DSPM-style discovery and classification for Azure Blob.
- Customers can start with lightweight checks and escalate to deeper inspection only where risk justifies it.

### 5. AI-Assisted Recommendations

Open AI Tools or the Operations Assistant panel.

Expected result:

- AI providers can be configured for cloud-connected, private-network, or air-gapped environments.
- The assistant summarizes current evidence and recommends governed next actions.
- AI output is grounded in discovered evidence, policy, and selected objects.
- AI does not directly apply risky changes without policy approval.

Show examples:

- Recommend tags for sensitive objects.
- Complete missing metadata.
- Prepare archive recommendation.
- Prepare quarantine recommendation.
- Delegate assessment for human review.

Positioning:

- AI helps operators understand and act faster.
- Everest keeps AI bounded by policy, evidence, and human approval where needed.

### 6. Human-In-The-Loop Action Center

Open Action Center.

Expected result:

- High-risk or destructive actions appear as pending approval.
- Approvers can review action type, selected source, selected objects, policy reason, risk, and expected impact.
- Approved actions can proceed through governed execution.
- Rejected actions remain auditable.

Customer scenarios to highlight:

- Quarantine sensitive object.
- Change access tier for stale content.
- Apply metadata or tags.
- Hold or archive regulated content.
- Block delete or risky mutation without approval.

Positioning:

- Everest separates recommendation from execution.
- Security and operations teams can enforce policy without slowing every safe action.

### 7. Evidence Center

Open Evidence Center and generate or review an evidence package.

Expected result:

- Evidence includes sources, scans, findings, actions, approvals, audit events, and AI recommendation context.
- Evidence excludes raw secrets and uncontrolled raw object content.
- The package can support security review, compliance reporting, and operational handoff.

Positioning:

- Everest is not only an action portal; it is also an audit and accountability system.

### 8. Connector Catalog And Future Connectors

Open Connector Catalog.

Expected result:

- Azure Blob Storage appears as an available enterprise connector.
- The catalog shows connector status, runtime, deployment support, capabilities, and action model.
- Custom connectors are represented as pluggable future extensions through connector contracts.

Positioning:

- Azure Blob is the first enterprise-grade path, but the architecture supports any future connector.
- Customer-owned connectors can be plugged into Everest through defined contracts.

### 9. Deployment Modes

Open or reference Deployment Modes documentation if needed.

Expected result:

- Cloud-connected mode is supported.
- Private-network mode is supported.
- Air-gapped mode is supported.
- AI provider options are configurable per environment.
- Secret provider options are configurable per environment.

Positioning:

- Everest can match customer network and compliance constraints instead of forcing one deployment model.

## Real-Time Customer Scenarios To Cover

Use at least ten scenarios during the walkthrough or Q&A:

1. Discover all containers for a saved Azure Blob source.
2. Select multiple containers and confirm selected scope before scanning.
3. List blobs and review object metadata without exposing raw content.
4. Detect PII-like content and show recommended tags.
5. Detect PHI-like content and route remediation through HITL.
6. Detect token-like content and prepare quarantine recommendation.
7. Identify stale content and prepare archive or tiering action.
8. Review broad-access or policy-sensitive objects.
9. Pause and resume a long-running scan job.
10. Recover scan progress from checkpoint after interruption.
11. Submit a high-risk action to Action Center for approval.
12. Generate an evidence package for audit or compliance review.
13. Explain how the same model works in cloud-connected, private-network, and air-gapped deployments.
14. Explain how a future custom connector can be added through connector contracts.

## If Azure Connectivity Fails During The Session

Use this fallback path without presenting it as a product limitation:

- Show saved Azure Blob source configuration and explain source-specific access.
- Show Workbench preview mode and available governed actions.
- Show existing scan job records and normalized results.
- Show Enterprise Dashboard posture, recommendations, and pending actions.
- Show Evidence Center package structure.
- Show Connector Catalog and deployment modes.

Positioning:

- Everest is designed to preserve scan state, audit state, and evidence even when cloud connectivity is interrupted.
- For production operations, connectivity, permissions, and policy posture are visible and actionable.

## Questions To Be Ready For

Can business users add Azure Blob without a raw connection string?

- Yes. The portal should collect simplified Azure fields and the backend constructs and stores the required access material securely.

Can raw secrets be viewed?

- No, not in the standard user path. Raw secret visibility should remain disabled unless an authorized troubleshooting mode is explicitly enabled.

Can users select existing containers, files, policies, and jobs?

- Yes. The portal should prefer dropdowns, multi-selects, and suggestions from discovered or saved state.

Can AI directly delete or change customer data?

- No. AI can recommend and prepare actions. Risky or destructive actions should remain preview-first and approval-gated.

Can this work without internet?

- Yes. Everest supports private-network and air-gapped deployment models with customer-controlled AI/runtime options.

Can the customer add their own connector?

- Yes. Everest uses connector contracts so future customer-owned connectors can be plugged in without redesigning the portal.

## Close

End the session by returning to the Enterprise Dashboard.

Expected closing view:

- Updated posture metrics.
- Scan or action activity visible.
- Pending approvals visible if any high-risk action was staged.
- Evidence path available for export or review.

Recommended close:

- Everest provides a governed Azure Blob operations control plane: discovery, DSPM scanning, AI-assisted recommendations, human approval, safe execution, and audit evidence across enterprise deployment models.
