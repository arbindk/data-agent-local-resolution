# Everest Data Agent Core: UI-Only Testing And Client Demo Script

## Purpose

This document provides browser-only testing steps for the current Everest Portal implementation.

It is intended for client demonstrations, QA walkthroughs, and manager review where the tester should not run API calls or command-line validation during the demonstration.

The test flow assumes the Everest services are already running and the tester can open the Portal in a browser.

```text
Portal URL: http://localhost:8080
```

No PowerShell, curl, Postman, or direct API command is required for the steps below.

## Demo Positioning

The Portal demonstrates your Data Agent Core ownership through the user journey:

```text
Source setup
  -> Connector discovery
  -> Governed onboarding
  -> Scan/job execution
  -> DSPM and AI recommendations
  -> Guardian decision review
  -> HITL approval
  -> Action preview or apply
  -> Evidence and audit proof
```

The important message for the client:

```text
The user works through the Portal.
The backend creates jobs, runs the Data Agent Core, enforces Guardian/HITL gates, and records evidence.
The user does not need to understand internal API calls or connection-string details.
```

## Test Data Modes

### Mode 1: Connected Azure Blob Environment

Use this when the test environment can connect to Azure Blob.

Required from tester or admin:

- Storage account name
- Account access key or approved imported secret
- One or more container names, or permission to discover containers
- Optional prefix/scope

Expected result:

```text
The Portal can test access, discover containers, list blobs, run scans, and preview governed actions.
```

### Mode 2: No-Azure UI Walkthrough

Use this when Azure access is not available during the client session.

Expected result:

```text
The Portal still demonstrates architecture, source setup screens, connector/AI catalogs, policy governance, Action Center gates, evidence states, and UI behavior.
Live Azure-specific operations may show an access/configuration message until a source is connected.
```

## UI Test Case 1: Portal Shell, Theme, And Notifications

### Steps

1. Open `http://localhost:8080` in the browser.
2. Confirm the left navigation is visible.
3. Confirm the top-right status pills are visible:
   - Agent
   - Control Plane
   - Mode
4. Click the theme toggle in the top-right corner.
5. Click it again to switch back.
6. Click `Alerts`.
7. Click `Clear all`.

### Expected Result

- The Portal loads without a blank screen.
- Left navigation shows:
  - Enterprise Dashboard
  - Architecture Flow
  - Source Inventory
  - Azure Blob Workbench
  - Connector Catalog
  - AI Tools
  - Onboarding
  - Scans & Jobs
  - Evidence Center
  - Guardian Agent
  - Action Center
  - Policy Governance
  - Agent Working Store
  - Diagnostics
- Theme changes between day and night.
- Notifications open from the top-right area.
- `Clear all` removes notification entries.
- Notification popups should not appear as unreadable black boxes in day mode.

### What This Proves

```text
The Portal shell is stable, supports day/night mode, and exposes a user-friendly notification model.
```

## UI Test Case 2: Enterprise Dashboard First View

### Steps

1. Open `Enterprise Dashboard`.
2. Click `Refresh Dashboard`.
3. Review `Current operating state`.
4. Review KPI cards:
   - Total objects
   - Total data
   - High risk
   - Governance
5. Review role tabs:
   - Estate Overview
   - CIO
   - CISO
   - Compliance
   - Data Steward
   - AI Readiness
   - Execution
   - Audit & Export

### Expected Result

- Dashboard refresh completes.
- The page shows a recommendation or next step.
- If no source/job exists yet, the dashboard guides the user to validate a source.
- If previous evidence exists, dashboard KPIs show object, risk, action, and governance counts.
- Role tabs switch without layout issues.

### What This Proves

```text
The product starts from an enterprise operations view, not a developer-only tool.
```

## UI Test Case 3: Add Azure Blob Using Friendly Fields

### Steps

1. Open `Source Inventory`.
2. Keep `Connector` as `Azure Blob Storage`.
3. Select the appropriate `Deployment mode`.
4. Enter:
   - Account name
   - Account access key
   - Cloud suffix
   - Containers, if already known
   - Prefix / scope, if needed
   - Data store ID
   - Source region
5. Confirm `Advanced access input` is not enabled by default.
6. Click `Save Source`.
7. Review `Source status`.

### Expected Result

- User is not forced to enter a raw connection string.
- Account name and account access key are collected as simplified fields.
- Raw secret should not remain visibly exposed after save.
- `Access status` should show configured/saved behavior.
- A notification confirms the source was saved.
- Source status should move away from pending when save succeeds.

### What This Proves

```text
Everest hides technical connection-string complexity from business users.
The backend can create/store required access material from simplified UI inputs.
```

## UI Test Case 4: Advanced Secret Import Visibility

### Steps

1. Stay on `Source Inventory`.
2. Locate `Advanced access input`.
3. Enable the checkbox.
4. Confirm `Support import secret` appears.
5. Disable the checkbox.

### Expected Result

- Advanced secret import is hidden by default.
- It appears only when explicitly enabled.
- Normal users are guided to friendly account fields instead.

### What This Proves

```text
The UI keeps technical/raw access fields out of the normal user path.
```

## UI Test Case 5: Test Connection And Discover Containers

### Steps

1. Open `Source Inventory`.
2. Select or load a saved source using `Load Saved Config` if needed.
3. Click `Test Connection`.
4. Click `Discover Containers`.
5. Review `Discovered containers`.

### Expected Result In Connected Azure Mode

- `Test Connection` succeeds.
- `Source status` shows connection OK.
- `Discovered containers` displays a table of containers.
- Each container row has a checkbox.
- The status cards indicate source readiness.

### Expected Result Without Azure Access

- The UI shows a clear configuration/access error.
- The page should not freeze or expose raw secrets.
- The next step should remain understandable: configure a valid source or use a known container manually.

### What This Proves

```text
Container choices come from backend discovery and can be used as dropdown/suggestion inputs.
```

## UI Test Case 6: Select Multiple Containers

### Steps

1. After container discovery, select two or more container checkboxes.
2. Click `Use Selected Containers`.
3. Review the selected-container summary.
4. Open `Azure Blob Workbench`.
5. Review the selected-container summary there as well.

### Expected Result

- A visible summary appears:

```text
Selected containers
N selected. Workbench object actions use the active container: <container-name>.
```

- Selected containers appear as chips.
- `Source Inventory`, `Onboarding`, and `Azure Blob Workbench` receive the selected container values.
- The active Workbench container is the first selected container.

### What This Proves

```text
The current implementation supports multi-container selection and carries that scope forward to governed work.
```

## UI Test Case 7: Saved Azure Blob Sources In Workbench

### Steps

1. Open `Azure Blob Workbench`.
2. Review `Saved Azure Blob sources`.
3. Click `Use` on a saved source.
4. Click `Modify` on a saved source.
5. Return to `Azure Blob Workbench`.
6. Click `Add Azure Blob`.

### Expected Result

- Saved sources are listed with display name, storage account, container/scope, and deployment mode.
- `Use` selects the source and applies it to Workbench fields.
- `Modify` opens Source Inventory with the saved values populated.
- `Add Azure Blob` opens Source Inventory with a clean form.
- `Remove` is available for saved source cleanup.

### What This Proves

```text
The Workbench is not asking users to retype technical values repeatedly.
Saved sources drive dynamic source selection.
```

## UI Test Case 8: Workbench Container And Blob Operations

### Steps

1. Open `Azure Blob Workbench`.
2. Select a saved source.
3. Set `Operation` to `list containers`.
4. Click `Discover Containers`.
5. Select one container and click `Use`.
6. Set `Operation` to `list blobs`.
7. Click `List Blobs`.
8. If blobs appear, click `Use` on one blob.
9. Set `Operation` to `view blob`.
10. Click `Run Operation`.

### Expected Result In Connected Azure Mode

- Workbench status shows:
  - Status
  - Auth
  - Container
- Auth should show a friendly value such as `account access saved`, not `saved_connection_string`.
- Container listing appears in a table.
- Blob listing appears in a table.
- Blob details show:
  - Blob
  - Size
  - Type
  - Tier
  - Metadata
  - Index tags
  - Preview when available

### Expected Result Without Azure Access

- The UI shows a clear access/configuration message.
- Raw support data remains hidden unless explicitly enabled.

### What This Proves

```text
Workbench supports advanced inspection while still using saved source context and friendly auth labels.
```

## UI Test Case 9: Workbench Field Visibility

### Steps

1. Open `Azure Blob Workbench`.
2. Change `Operation` through:
   - list containers
   - list blobs
   - view blob
   - create blob
   - set metadata
   - set tags
   - copy blob
   - move blob
   - set tier
   - rehydrate blob
   - delete blob
3. Watch which fields appear and disappear.

### Expected Result

- Blob field appears only when object-specific operation needs it.
- Prefix appears for list blobs.
- Limit appears for list operations.
- Target fields appear for copy/move.
- Tier fields appear for set tier/rehydrate.
- Confirmation appears for delete/move/tier/rehydrate style operations.
- Content, file, metadata, tags, and overwrite controls appear only when relevant.

### What This Proves

```text
The UI reduces unnecessary input fields and shows elements only when required.
```

## UI Test Case 10: Support Data Visibility

### Steps

1. Open `Azure Blob Workbench`.
2. Run any Workbench operation.
3. Confirm normal result cards/tables are visible.
4. Click `Support Data`.
5. Click it again to hide support data.
6. Repeat in day mode.

### Expected Result

- Raw payload/support data is hidden by default.
- Support data appears only after explicit user action.
- In day mode, the payload area is readable.
- Hiding support data returns the user to normal business-facing output.

### What This Proves

```text
Raw technical data is available for troubleshooting but not exposed by default.
```

## UI Test Case 11: Onboarding Creates Governed Work

### Steps

1. Open `Onboarding`.
2. Confirm source fields are prefilled from Source Inventory when available:
   - Data store ID
   - Storage account
   - Containers
   - Prefix / scope
3. Select policy.
4. Keep `Action mode` as `Preview` for safety.
5. Keep `Write-back enabled` as `false` for the safe client demo path.
6. Click `Onboard Azure Blob Source`.
7. Review `Jobs`.
8. Review `Evidence & audit summary`.

### Expected Result

- `Governed work created` appears.
- A job and lease are shown.
- Scopes show the selected container(s).
- `Open Scans & Jobs` is available.
- Jobs table shows the created work.

### What This Proves

```text
The Portal creates governed work instead of directly executing random operations.
```

## UI Test Case 12: Run Governed Job From Scans & Jobs

### Steps

1. Open `Scans & Jobs`.
2. Click `Refresh Execution Status`.
3. Click `Run Job`.
4. Review `Execution status`.
5. Review the `Execution lifecycle` row.
6. Click `Review Evidence`.

### Expected Result In Connected Azure Mode

- Execution completes.
- Result card shows:
  - Execution
  - Status
  - Objects scanned
  - Scopes
  - Evidence
  - Write-back
- Runtime status shows:
  - Runtime
  - Latest execution
  - Data Agent
  - Secret provider
  - Secret write mode
  - Authorization

### Expected Result Without Azure Access

- If the job depends on live Azure credentials, it may show an access/configuration failure.
- The UI should still show the failure clearly.
- No uncontrolled action should occur.

### What This Proves

```text
The user starts execution from the UI; backend Data Agent Core handles the job lifecycle.
```

## UI Test Case 13: Async Azure Blob Scan Job

### Steps

1. Open `Scans & Jobs`.
2. Click `Refresh Execution Status`.
3. Click `Queue Async Scan`.
4. Review the async scan job table.
5. Click `Refresh Async Jobs`.
6. Click `Resume` if the job is queued or paused.
7. Click `Pause`.
8. Click `Resume` again.
9. Click `Records` after records are available.

### Expected Result In Connected Azure Mode

- Job table shows:
  - Job
  - Status
  - Source
  - Checkpoint
  - Records
  - Worker
  - Retries
  - Action buttons
- Checkpoint includes container, prefix, scope index, page count.
- Retry and throttle counters are visible.
- Records opens normalized scan output when available.

### Expected Result Without Azure Access

- Queue may be blocked with a credential/source message.
- UI should show the reason and remain usable.

### What This Proves

```text
The current implementation includes checkpoint-aware async scan lifecycle controls through the UI.
```

## UI Test Case 14: Evidence Center Review

### Steps

1. Open `Evidence Center`.
2. Click `Refresh Evidence Center`.
3. Review `Evidence status`.
4. Review `Decision Provenance Detail`.
5. Review `Highest-risk objects`.
6. Review `Audit timeline`.
7. Click export buttons:
   - Export Evidence Package
   - Executive Export
   - Manifest CSV
   - Guardian Export
   - Compliance Export
   - AI Readiness Export

### Expected Result

- Evidence status shows object/action/audit counts.
- Provenance detail shows:
  - Execution ID
  - Evidence ID
  - Decision ID
  - Object
  - Policy Version
  - Rules Applied
  - Reasoning
  - Guardian Agent Decision
  - HITL Required
  - Final Action
- Highest-risk objects table appears when risk/action records exist.
- Audit timeline shows event, status, actor, and time.
- Export buttons download or open evidence artifacts.

### What This Proves

```text
Everest can explain what happened, why it happened, and who/what was involved.
```

## UI Test Case 15: Guardian Agent Review

### Steps

1. Open `Guardian Agent`.
2. Click `Refresh Guardian Agent`.
3. Review `Action posture`.
4. Review `Remediation queue`.
5. Click `Evaluate Policy Governance`.

### Expected Result

- Action posture shows:
  - Candidates
  - Allowed
  - Needs review
  - Blocked / preview
- If an execution exists, remediation queue shows object, risk, action, decision, and status.
- If no execution exists, the page explains Guardian controls:
  - Safe metadata/index tag write-back
  - Restricted operations
  - Guardian decisions

### What This Proves

```text
Guardian is represented as the safety gate before write-back or restricted actions.
```

## UI Test Case 16: Policy Governance

### Steps

1. Open `Policy Governance`.
2. Click `Load Policies`.
3. Click `India -> UAE Scenario`.
4. Click `Safe Tagging Scenario`.
5. Click `Refresh Policy Artifact Trust`.
6. Review `Policy Artifact Trust`.
7. Review `Policy library`.

### Expected Result

- Policies load into the policy library.
- Regional/governance scenarios return clear allow/block/review style outcomes.
- Policy artifact trust view shows policy readiness and trust information.

### What This Proves

```text
Policy is visible to the user as an enterprise governance control, not hidden backend logic.
```

## UI Test Case 17: Action Center Preview, Approval, And Block

### Steps

1. Open `Action Center`.
2. Click `Refresh Action Center`.
3. In `Azure Blob action matrix`, choose a restricted action such as `archive`, `quarantine`, `migration`, or `delete`.
4. Click `Select`.
5. Fill `File ID` or `Blob path` if not already populated.
6. Click `Preview Action`.
7. Review `Action Result`.
8. Click `Request Approval`.
9. In `Approval queue`, click `Approve` or `Reject`.
10. Click `Refresh Action Center`.

### Expected Result

- Preview action returns `planned`.
- Preview result says Azure Blob was not modified.
- Restricted actions show approval requirements.
- Approval queue shows `PENDING` requests.
- Approve changes status to `APPROVED`.
- Reject changes status to `REJECTED`.
- HITL posture counters update.

### What This Proves

```text
Risky actions are planned first, then routed through HITL instead of silently applying.
```

## UI Test Case 18: Execute Action Block Without Required Gates

### Steps

1. Open `Action Center`.
2. Choose `delete` or another restricted operation.
3. Fill `File ID` or `Blob path`.
4. Set `Guardian decision` to `deny` or `pending`.
5. Leave confirmation blank or enter the wrong confirmation.
6. Click `Execute Action`.

### Expected Result

- The action is blocked or approval is required.
- The UI explains why execution cannot continue.
- Azure Blob should not be modified.
- Evidence/action history records the blocked attempt.

### What This Proves

```text
The current implementation blocks unsafe execution when Guardian/HITL/confirmation gates are not satisfied.
```

## UI Test Case 19: AI Copilot Recommendations

### Steps

1. Run or select a governed execution first.
2. Open `Action Center`.
3. Review `AI Copilot`.
4. Select an AI provider.
5. Keep `AI mode` as `recommend only`.
6. Click `Ask AI`.
7. Review recommendations.
8. Click `Use` on one recommendation.
9. Change `AI mode` to `submit pending HITL approvals`.
10. Click `AI Submit HITL Queue`.

### Expected Result

- AI recommendations appear in a table.
- Columns include provider, action, object, risk, confidence, mode, next.
- `Use` loads the recommendation into the action form.
- If HITL submission is enabled, pending approvals are created.
- AI does not directly execute mutations.

### What This Proves

```text
AI participates as an assistant for recommendation and HITL preparation, not as an ungoverned action executor.
```

## UI Test Case 20: AI Tools Provider Configuration

### Steps

1. Open `AI Tools`.
2. Click `Refresh Providers`.
3. Select a provider.
4. Review `Provider filters`.
5. Review `AI provider catalog`.
6. Select:
   - Network mode
   - Auth scheme
   - Model / profile
   - Automation
   - Redaction
   - Prompt audit
   - Evidence grounding
   - HITL submission
7. Click `Save AI Config`.
8. Click `Test Provider`.
9. If sample runtime is available, click `Use Sample Runtime`, then `Invoke AI Runtime`.
10. Click `Run Contract Test`.
11. Click `Refresh History`.

### Expected Result

- Provider catalog loads built-in and customer-owned providers.
- Selected provider details show capabilities and safety posture.
- Config save succeeds.
- Provider test succeeds for configured/built-in providers.
- Runtime invocation returns structured results when runtime is available.
- Contract test shows checks with pass/fail details.
- Runtime audit history records invocation.

### What This Proves

```text
Customers can configure different AI tools for connected, private-network, or air-gapped environments.
```

## UI Test Case 21: Connector Catalog And Runtime Contract

### Steps

1. Open `Connector Catalog`.
2. Click `Refresh Catalog`.
3. Change `Customer environment` filter.
4. Change `Runtime` filter.
5. Review `Available and extensible connectors`.
6. Select a connector.
7. Review `Selected connector details`.
8. In `Runtime contract`, select an operation:
   - health
   - discover_sources
   - scan_metadata
   - preview_action
   - execute_action
9. If sample runtime is available, click `Use Sample Runtime`.
10. Click `Invoke Runtime`.
11. Click `Run Contract Test`.
12. Click `Refresh History`.

### Expected Result

- Connector catalog lists built-in, planned, and customer manifest connectors.
- Environment support matrix updates based on filters.
- Runtime result appears with:
  - status
  - connector ID
  - operation
  - runtime kind
  - contract version
  - checks
- Runtime audit history records connector invocations.
- Apply-style runtime calls should remain blocked unless Guardian/HITL/confirmation conditions are met.

### What This Proves

```text
The product is being built for future connectors, not only Azure Blob.
```

## UI Test Case 22: Agent Working Store

### Steps

1. Open `Agent Working Store`.
2. Click `Refresh Working Store`.
3. Review table/card output.
4. Click `Open Evidence Center`.

### Expected Result

- Working store shows available local execution tables or a clear empty-state message.
- After scans/executions, it should show normalized files, manifest summary/details, Guardian decisions, and write-back action related records.
- The page links back to Evidence Center for audit proof.

### What This Proves

```text
The Data Agent Core stores normalized execution state for analysis and evidence, not uncontrolled raw customer content.
```

## UI Test Case 23: Diagnostics

### Steps

1. Open `Diagnostics`.
2. Review visible status and raw payload panels.
3. Switch to day mode.
4. Confirm diagnostic payloads remain readable.
5. Trigger a few actions, such as dashboard refresh or source status refresh.
6. Return to Diagnostics and confirm status remains readable.

### Expected Result

- Diagnostic raw payload is readable in day and night mode.
- No black unreadable box appears in day mode.
- Diagnostics should be useful for support without disturbing normal business workflows.

### What This Proves

```text
Support/debug data is accessible and readable without being part of the normal user path.
```

## UI Test Case 24: Architecture Flow Page

### Steps

1. Open `Architecture Flow`.
2. Review the complete architecture view.
3. Review component communication.
4. Review flow/test-case visualization if available.
5. Explain the sequence:

```text
Portal intent
  -> Data Agent Core
  -> Connector
  -> Zero-copy access
  -> Local working store
  -> DSPM / AI signals
  -> Guardian decision
  -> HITL or approved action
  -> Evidence and provenance
```

### Expected Result

- Page gives a visual explanation of architecture and sequence.
- It supports client discussion without needing command-line output.

### What This Proves

```text
The product has a presentable architecture story aligned to the Data Agent Cell model.
```

## Apply And Write-back UI Test Preconditions

Use these cases only in a connected Azure Blob test environment where the customer has approved write-back testing.

Before starting, confirm:

- A saved Azure Blob source exists.
- The source points to a non-production or approved test container.
- The customer has one or more test blobs available.
- `Action mode` can be changed to `Apply`.
- `Write-back enabled` can be changed to `true`.
- The tester understands that write-back tests can modify metadata, index tags, tier, or object state depending on the selected action.

Recommended safe starting point:

```text
Start with metadata/index tag write-back.
Use restricted actions only in an approved test container.
Keep delete as a blocked-control test unless the customer explicitly approves deletion testing.
```

## UI Test Case 25: End-To-End Preview To Apply Mode Setup

### Steps

1. Open `Source Inventory`.
2. Load or save a valid Azure Blob source.
3. Click `Test Connection`.
4. Click `Discover Containers`.
5. Select one approved test container.
6. Click `Use Selected Containers`.
7. Open `Onboarding`.
8. Confirm selected source/container values are present.
9. Set `Action mode` to `Preview`.
10. Set `Write-back enabled` to `false`.
11. Click `Onboard Azure Blob Source`.
12. Open `Scans & Jobs`.
13. Click `Run Job`.
14. Open `Evidence Center` and click `Refresh Evidence Center`.
15. Return to `Onboarding`.
16. Change `Action mode` to `Apply`.
17. Change `Write-back enabled` to `true`.
18. Click `Onboard Azure Blob Source` again for the same approved test scope.

### Expected Result

- Preview run completes without modifying Azure Blob.
- Evidence Center shows preview/governed execution records.
- Apply mode is visibly selected only after the user changes it.
- Write-back is visibly enabled only after the user changes it.
- A new governed job/lease is created for apply-mode testing.
- The UI does not silently switch from Preview to Apply.

### What This Proves

```text
The customer intentionally moves from preview to apply.
Everest does not perform write-back unless the user selects apply mode and enables write-back.
```

## UI Test Case 26: Safe Metadata Write-back With Approval

### Steps

1. Complete `UI Test Case 25`.
2. Open `Scans & Jobs`.
3. Click `Run Job` for the apply-mode governed work.
4. Open `Action Center`.
5. Click `Refresh Action Center`.
6. In `Operation`, select `apply_blob_metadata`.
7. Select or enter a valid `Execution`.
8. Select or enter a valid `File ID`.
9. Set `Guardian decision` to `allow`.
10. Click `Request Approval`.
11. In `Approval queue`, click `Approve`.
12. Confirm HITL posture shows approved count increasing.
13. Click `Apply Approved Tags`.
14. Review `Action Result`.
15. Open `Evidence Center`.
16. Click `Refresh Evidence Center`.
17. Open Azure Blob Workbench.
18. Use the same source/container/blob.
19. Set operation to `view blob`.
20. Click `Run Operation`.
21. Review metadata in blob details.

### Expected Result

- Approval request is created with status `PENDING`.
- After approval, queue shows `APPROVED`.
- `Apply Approved Tags` completes or reports applied/skipped/failed counts.
- Action Result shows a completed or controlled result.
- Evidence Center records the write-back action.
- Workbench view shows updated metadata when Azure accepts the write-back.
- If Azure blocks metadata update due permissions, UI shows a clear failed result and evidence records the failure.

### What This Proves

```text
Safe metadata write-back requires apply mode, write-back enabled, Guardian allow, and HITL approval.
The customer can verify the result from Evidence Center and Workbench.
```

## UI Test Case 27: Azure Blob Index Tag Write-back With Approval

### Steps

1. Open `Action Center`.
2. Click `Refresh Action Center`.
3. In `Operation`, select `apply_blob_index_tags`.
4. Select or enter the same valid `Execution`.
5. Select or enter a valid `File ID`.
6. Set `Guardian decision` to `allow`.
7. Click `Request Approval`.
8. Approve the request in `Approval queue`.
9. Click `Apply Approved Tags`.
10. Review `Action Result`.
11. Open `Evidence Center`.
12. Click `Refresh Evidence Center`.
13. Open `Azure Blob Workbench`.
14. Select the same blob.
15. Set operation to `view blob`.
16. Click `Run Operation`.
17. Review `Index tags`.

### Expected Result

- Approval is required and can be approved from the UI.
- Apply operation runs only after approval.
- Action Result shows completed/applied or a clear failure reason.
- Evidence Center records action status and audit details.
- Workbench shows index tags if the Azure account/container supports them and permissions allow reading them.

### What This Proves

```text
Everest supports Azure-native governance tags through the same apply/write-back safety model.
```

## UI Test Case 28: Apply Downgrades To Plan When Write-back Is Disabled

### Steps

1. Open `Onboarding`.
2. Set `Action mode` to `Apply`.
3. Set `Write-back enabled` to `false`.
4. Open `Action Center`.
5. Select `apply_blob_metadata`.
6. Enter a valid `Execution`.
7. Enter a valid `File ID` or `Blob path`.
8. Set `Guardian decision` to `allow`.
9. Click `Preview Action`.
10. Review preview result.
11. Click `Execute Action`.

### Expected Result

- Preview result shows `planned`.
- Execute action remains a no-mutation plan because write-back is disabled.
- Result explains that Azure Blob was not modified.
- Azure Blob is not modified.
- Evidence/action history records a planned or blocked action.

### What This Proves

```text
Apply mode alone is not enough.
Write-back must be explicitly enabled before mutation; otherwise the product stays in a safe plan/no-change path.
```

## UI Test Case 29: Apply Blocked Without HITL Approval

### Steps

1. Open `Onboarding`.
2. Set `Action mode` to `Apply`.
3. Set `Write-back enabled` to `true`.
4. Open `Action Center`.
5. Select `apply_blob_metadata`.
6. Enter a valid `Execution`.
7. Enter a valid `File ID`.
8. Set `Guardian decision` to `allow`.
9. Do not request approval.
10. Click `Execute Action`.

### Expected Result

- UI shows `Approval required`.
- Execution does not proceed.
- Azure Blob is not modified.
- Action Center remains usable and shows the user can request approval.

### What This Proves

```text
Even safe metadata write-back is blocked until HITL approval exists when approval enforcement is active.
```

## UI Test Case 30: Apply Blocked Without Guardian Allow

### Steps

1. Open `Action Center`.
2. Select `apply_blob_metadata`.
3. Enter a valid `Execution`.
4. Enter a valid `File ID`.
5. Set `Guardian decision` to `pending` or `deny`.
6. Click `Request Approval`.
7. Approve the request if it appears.
8. Click `Execute Action`.

### Expected Result

- Approval alone does not bypass Guardian.
- Execute action is blocked because Guardian decision is not `allow`.
- Action Result explains Guardian enforcement.
- Azure Blob is not modified.
- Evidence/action history records the blocked action.

### What This Proves

```text
HITL approval and Guardian allow are separate controls.
Both are required before apply/write-back.
```

## UI Test Case 31: Delete Blocked Even In Apply Mode

### Steps

1. Open `Onboarding`.
2. Confirm `Action mode` is `Apply`.
3. Confirm `Write-back enabled` is `true`.
4. Open `Action Center`.
5. Set `Operation` to `delete`.
6. Enter a valid `Execution`.
7. Enter a valid `File ID` or `Blob path`.
8. Set `Guardian decision` to `deny` or `pending`.
9. Leave `Confirmation` blank.
10. Click `Preview Action`.
11. Review planned preview.
12. Click `Execute Action`.
13. Set `Confirmation` to `APPLY`.
14. Click `Execute Action` again.
15. Set `Confirmation` to `DELETE`.
16. Keep Guardian decision as `deny` or `pending`.
17. Click `Execute Action` again.

### Expected Result

- Preview is allowed as a plan and does not delete anything.
- Blank confirmation blocks execution.
- `APPLY` confirmation is not enough for delete.
- `DELETE` confirmation is still not enough unless Guardian allows and approval exists.
- Azure Blob object remains unchanged.
- Evidence/action history records blocked attempts.

### What This Proves

```text
Delete is treated as a destructive action with stricter controls than normal metadata write-back.
```

## UI Test Case 32: Restricted Archive Or Quarantine Approval Flow

### Steps

1. Open `Action Center`.
2. Select `archive` or `quarantine`.
3. Enter a valid `Execution`.
4. Enter a valid `File ID` or `Blob path`.
5. Set `Guardian decision` to `review_required` or `pending`.
6. Click `Preview Action`.
7. Review the planned action.
8. Click `Request Approval`.
9. In `Approval queue`, click `Reject`.
10. Confirm action remains blocked.
11. Click `Request Approval` again.
12. In `Approval queue`, click `Approve`.
13. Set `Guardian decision` to `allow`.
14. Enter confirmation `APPLY`.
15. Click `Execute Action`.

### Expected Result

- Preview returns `planned`.
- First HITL request can be rejected and action remains blocked.
- Second HITL request can be approved.
- Execute requires Guardian allow and confirmation.
- If the connector supports the selected action, result shows completed.
- If the action is not supported or not allowed in the current connector path, result shows a controlled block/failure.
- Evidence Center records preview, approval, rejection/approval, and final outcome.

### What This Proves

```text
Restricted actions are governed workflows with preview, HITL, Guardian, confirmation, action result, and evidence.
```

## UI Test Case 33: Post-Write-back Evidence Verification

### Steps

1. Complete one successful metadata or index tag write-back.
2. Open `Evidence Center`.
3. Click `Refresh Evidence Center`.
4. Review `Evidence status`.
5. Review `Decision Provenance Detail`.
6. Review `Audit timeline`.
7. Click `Guardian Export`.
8. Click `Compliance Export`.
9. Open `Guardian Agent`.
10. Click `Refresh Guardian Agent`.
11. Open `Action Center`.
12. Click `Refresh Action Center`.

### Expected Result

- Evidence status reflects action and audit counts.
- Provenance detail shows policy/rules/reasoning/Guardian/action fields.
- Audit timeline shows action event and actor.
- Guardian Agent shows updated action posture.
- Action Center history includes completed or blocked action status.
- Exports are available for review.

### What This Proves

```text
Write-back does not end at source mutation.
Everest also records proof for governance, audit, and customer review.
```

## UI Test Case 34: Recovery And Re-run After Planned, Failed, Or Blocked Apply

### Steps

1. Open `Action Center`.
2. Choose `apply_blob_metadata`.
3. Enter a valid `Execution`.
4. Enter a valid `File ID`.
5. Set `Action mode` to `Apply` from `Onboarding`.
6. Set `Write-back enabled` to `false`.
7. Click `Execute Action`.
8. Confirm it remains planned/no-change or blocked.
9. Return to `Onboarding`.
10. Set `Write-back enabled` to `true`.
11. Return to `Action Center`.
12. Set `Guardian decision` to `allow`.
13. Click `Request Approval`.
14. Approve the request.
15. Click `Execute Action` or `Apply Approved Tags`.
16. Review final result.
17. Open `Evidence Center` and refresh.

### Expected Result

- Initial apply is safely prevented from mutating the source.
- User can fix configuration through UI.
- Approval can be requested and approved.
- Re-run completes or reports connector-specific failure clearly.
- Evidence contains both the blocked attempt and the later successful/controlled attempt.

### What This Proves

```text
Everest supports safe recovery from blocked apply attempts without losing audit visibility.
```

## Recommended Client Demo Sequence

Use this shorter sequence for the actual client meeting:

1. `Enterprise Dashboard` -> `Refresh Dashboard`
2. `Source Inventory` -> Add or load Azure Blob source
3. `Discover Containers` -> select multiple containers -> `Use Selected Containers`
4. `Azure Blob Workbench` -> show saved source, list containers/list blobs, preview object
5. `Onboarding` -> create governed work in Preview mode
6. `Scans & Jobs` -> run job or show async scan lifecycle
7. `Evidence Center` -> show evidence, provenance, audit timeline
8. `Guardian Agent` -> show action posture and remediation queue
9. `Action Center` -> preview restricted action, request approval, show HITL
10. `AI Copilot` -> ask for recommendation, show no direct mutation
11. `Connector Catalog` -> show future connector extensibility
12. `AI Tools` -> show customer-configurable AI providers
13. Optional connected Azure apply path -> approved metadata/index tag write-back
14. `Evidence Center` -> verify post-write-back evidence and audit
15. `Architecture Flow` -> close with end-to-end architecture

## Current UI Coverage Summary

| Product Area | UI Screen | Current Test Coverage |
|---|---|---|
| Dashboard posture | Enterprise Dashboard | KPIs, role views, next steps |
| Source setup | Source Inventory | Friendly fields, saved source, credential hiding |
| Dynamic dropdowns | Source Inventory, Workbench, Action Center | Containers, jobs, executions, file IDs, providers |
| Workbench | Azure Blob Workbench | Saved sources, list/view/create/update/tag/tier/action controls |
| Governed work | Onboarding | Job and lease creation |
| Execution | Scans & Jobs | Run job, execution status, async scan lifecycle |
| Evidence | Evidence Center | Manifest, provenance, audit, export |
| Guardian | Guardian Agent | Action posture, queue, apply approved write-back gate |
| HITL/actions | Action Center | Preview, approval request, approve/reject, execute block |
| Apply/write-back | Onboarding, Action Center, Evidence Center, Workbench | Apply mode, write-back enabled, Guardian allow, HITL approval, metadata/index tag update, post-action proof |
| AI | AI Tools, Action Center | Provider config, runtime, recommendations, HITL preparation |
| Connectors | Connector Catalog | Catalog, runtime invoke, contract test, audit |
| Working state | Agent Working Store | Local normalized execution state |
| Support | Diagnostics, Support Data | Raw payload only when needed |

## Known UI Expectations For Current Build

- Source users should use friendly fields, not raw connection strings.
- Raw data/support payloads should be hidden unless explicitly enabled.
- Day/night themes should keep text readable.
- Notifications should appear in the top-right area and be clearable.
- Workbench advanced fields should appear only for operations that need them.
- Selected multiple containers should be visible as a selected-container summary.
- Restricted actions should preview first and require Guardian/HITL/confirmation before apply.
- AI should recommend or submit HITL tasks, not mutate data directly.

## What To Say If Azure Is Not Connected During Demo

Use this wording:

```text
This environment is currently not connected to a live Azure Blob account.
The same UI flow is ready for a connected tenant: save source, discover containers, scan, inspect, govern, and record evidence.
For this session, I will show the governed UI path, action gates, AI/HITL behavior, connector framework, and evidence model without applying changes to a live customer source.
```

## What This UI Demo Proves

```text
The current Everest implementation is already a governed enterprise portal path:

User-friendly source setup
  -> dynamic connector/source options
  -> Workbench for advanced inspection
  -> governed onboarding
  -> scan/job lifecycle
  -> Guardian/HITL/action gates
  -> AI recommendations
  -> evidence and audit
  -> future connector and AI extensibility
```

## Next Development After UI Validation

After UI testing, the next implementation work should be:

1. Route every UI action through a first-class Orchestrator request envelope.
2. Replace remaining Azure-specific backend assumptions with generic connector job contracts.
3. Integrate the real Guardian decision service with mandatory policy bundle hash.
4. Add richer UI status for `blocked`, `HITL required`, `approved`, `applied`, and `degraded`.
5. Add more UI automation tests for the browser path.
6. Improve the architecture GIF using the validated UI scenarios.
