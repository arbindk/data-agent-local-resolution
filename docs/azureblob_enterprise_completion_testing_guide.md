# Azure Blob Enterprise Completion Testing Guide

This guide validates the Azure Blob customer portal as an enterprise operations surface. It covers connected Azure environments, private network deployments, and air-gapped validation operation where actions are previewed and recorded without modifying customer storage.

## Scope

- Source configuration and credential handling
- Dynamic dropdowns for containers, blobs, file IDs, jobs, policies, and AI providers
- Container and object workbench operations
- Metadata, index tag, copy, move-prep, tier, rehydrate, archive, quarantine, migration, and delete scenarios
- DSPM content scan, keyword matching, entity extraction, and tiered scan modes
- AI recommendations, automatic safe actions, and HITL approval routing
- Evidence Center, audit timeline, and action history

## Pre-Test Setup

1. Start the Data Agent portal.
2. Open the portal in a browser.
3. Add or select a saved Azure Blob source in Source Inventory using simplified account/access fields.
4. Confirm the portal shows saved source access without exposing raw secret values.
5. Keep Workbench preview enabled until the preview output is understood.
6. For apply tests, use a non-production test storage account or a disposable approved container.
7. Run `node scripts/verify_portal_enterprise.js` before UI validation to catch customer-facing terminology and portal syntax regressions.

Expected output:

- Agent status shows healthy.
- Source credential shows configured after saving.
- Portal option dropdowns are initially empty or show previously discovered state.
- Evidence Center can be empty before the first operation.

## Scenario 1: Discover Containers

Steps:

1. Open Azure Blob Workbench.
2. Click `Discover Containers`.
3. Select a returned container with `Use`.

Expected output:

- Containers table shows container name, modified time, and lease state when Azure returns it.
- Container dropdowns are populated across Source Inventory, Onboarding, Workbench, and Action Center.
- Evidence Center records `azureblob.workbench.list_containers`.
- Audit timeline records the same operation.

## Scenario 2: List and Select Blobs

Steps:

1. Select a container.
2. Optionally enter a prefix.
3. Click `List Blobs`.
4. Click `Use` on one blob.

Expected output:

- Blob table shows blob path, size, type, and modified time.
- Blob path and file ID suggestions are populated.
- Action Center file ID is prefilled as `azureblob:<container>:<blob>`.
- Evidence Center records `azureblob.workbench.list_blobs`.

## Scenario 3: Create or Update Test Blob

Steps:

1. Select `Create blob` or `Update blob`.
2. Enter blob path, content type, metadata, tags, and inline content or upload file.
3. Keep `dry-run` checked and run the operation.
4. If the preview is correct, uncheck `dry-run` and run against a test container.

Expected output:

- Dry run returns `status: preview` and `would_upload_bytes`.
- Apply returns `status: ok` and object details.
- Dynamic blob/file ID dropdowns update after listing or viewing.
- Action history records create/update as planned or executed.

## Scenario 4: View and Download Blob

Steps:

1. Select a blob.
2. Run `View blob`.
3. Run `Download blob`.
4. Click `Download Result`.

Expected output:

- Object details show size, content type, access tier, metadata, and index tags.
- Text preview appears for text-compatible content.
- Download result produces the selected blob payload.
- Evidence records view/download events.

## Scenario 5: Metadata and Index Tag Updates

Steps:

1. Select a blob.
2. Enter `owner=qa,purpose=everest` in Metadata.
3. Run `Set metadata` in dry-run.
4. Enter `everest_test=true,classification=internal` in Index tags.
5. Run `Set tags` in dry-run.
6. For apply, use Action Center approval flow or uncheck dry-run only in a test account.

Expected output:

- Dry run returns `status: preview`.
- Applied metadata/tags appear in object details after `View blob`.
- Evidence and audit show metadata/tag operation.
- Action Center shows planned/executed records.

## Scenario 6: Copy Blob

Steps:

1. Select a source blob.
2. Click `Copy`.
3. Confirm target container and target prefix.
4. Run with `dry-run` checked.
5. Apply only in a test account after verifying target.

Expected output:

- Preview shows source container/blob and target container/blob.
- Apply starts Azure copy and returns copy ID/status when Azure provides it.
- Evidence records `azureblob.workbench.copy_blob`.
- Action history includes copy operation.

## Scenario 7: Move Preparation

Steps:

1. Select a source blob.
2. Select `Move prep`.
3. Enter target container/prefix and confirmation `MOVE`.
4. Run in dry-run first.
5. Apply only after validating the copy destination.

Expected output:

- Portal prepares move as copy-first.
- Source deletion is not silently performed.
- Output shows `source_delete_required: true`.
- Final source delete must be separately approved/executed.

## Scenario 8: Set Tier and Rehydrate

Steps:

1. Select a blob.
2. Click `Tier`; default target tier becomes `archive`.
3. Run dry-run.
4. Select `Rehydrate`; target tier becomes `hot`.
5. Choose `standard` or `high` priority and run dry-run.
6. Apply in test only with confirmation `APPLY`.

Expected output:

- Dry run shows target tier and rehydrate priority.
- Apply requests Azure tier change.
- Evidence records `set_tier` or `rehydrate_blob`.
- Object details eventually reflect Azure tier state after Azure completes the operation.

## Scenario 9: DSPM Content Scan and Keyword Matching

Steps:

1. View or create a blob with sample sensitive text.
2. Select DSPM scan tier: `rules`, `model`, or `deep`.
3. Choose keyword pack.
4. Add optional custom pattern lines: `pattern|severity|entity_type|match_mode`.
5. Click `Run DSPM Scan`.

Expected output:

- Results show classification, risk score, detected entities, checks, recommended metadata, and tags.
- Keyword pattern hits appear as checks/entities depending on detector.
- High risk results require HITL where configured.
- Evidence records `dspm.content_scan.completed`.

## Scenario 10: AI Recommendation and HITL

Steps:

1. Select a blob with enough context.
2. Click `Ask AI` in Workbench or Action Center.
3. Review recommended operation, metadata, tags, and rationale.
4. Use `Submit HITL` for risky/destructive recommendations.

Expected output:

- AI recommendation is rendered with operation, confidence, mode, and next action.
- Safe metadata/tag recommendations can be staged.
- Risky actions are submitted as pending approval.
- Evidence records `ai.recommendation.created`.
- Action Center pending count increases.

## Scenario 11: Governed Action Preview

Steps:

1. Open Action Center.
2. Select operation such as `archive`, `quarantine`, `migration`, or `delete`.
3. Select file ID from dropdown.
4. Fill target container/prefix/tier if required.
5. Click `Preview Action`.

Expected output:

- Status is `planned`.
- Azure Blob is not modified.
- Action history records planned operation.
- Evidence/audit records action preview.

## Scenario 12: Governed Action Approval and Execution

Steps:

1. In Action Center, select operation and object.
2. Click `Request Approval`.
3. Approve through the approval queue or Control Plane.
4. Set action mode to apply and writeback enabled in onboarding/source settings.
5. Set Guardian decision to `allow`.
6. Enter confirmation:
   - `APPLY` for archive, retrieve, rehydrate, quarantine, migration
   - `DELETE` for delete
7. Click `Execute Action`.

Expected output:

- Without approval, execution is blocked.
- Without Guardian allow, execution is blocked.
- Without required confirmation, execution is blocked.
- With all gates satisfied, supported action returns `completed`.
- Evidence, audit, and action history show the final status.

## Scenario 13: Delete Blob and Delete Container

Steps:

1. Use only disposable test objects/containers.
2. For blob delete, select `Delete blob`, keep dry-run, and confirm preview.
3. To apply, uncheck dry-run and enter `DELETE`.
4. For container delete, select `Delete container`, keep dry-run, and confirm preview.
5. To apply, uncheck dry-run and enter `DELETE_CONTAINER`.

Expected output:

- Dry run shows required confirmation and no changes.
- Apply marks blob/container for deletion in Azure.
- Evidence and audit show delete operation.
- Action history shows planned or executed status.

## Scenario 14: Evidence and Audit Package Review

Steps:

1. Run at least one workbench operation, one DSPM scan, one AI recommendation, and one action preview.
2. Open Evidence Center.
3. Refresh Evidence Center.
4. Open Action Center and refresh.

Expected output:

- Evidence object count is greater than zero.
- High-risk count reflects DSPM/action risk.
- Action count reflects planned/executed actions.
- Audit timeline shows recent portal operations.
- Action Center shows pending, approved/executed, or blocked/failed posture.

## Scenario 15: Air-Gapped or Private Network Mode

Steps:

1. Configure local/private AI provider.
2. Use saved local connector/runtime config.
3. Keep Workbench and Action Center in dry-run if Azure endpoint is unavailable.
4. Run DSPM using built-in rules or local model profile.

Expected output:

- Portal remains usable without internet.
- AI provider status can show local/private/air-gapped.
- DSPM rules and keyword matching work locally.
- Evidence/audit/action records are persisted locally.
- Azure apply operations fail clearly if endpoint or credentials are unavailable.

## Failure Expectations

- Missing saved source access: API returns a clear credential error.
- Missing container/blob: API returns required field error.
- Existing blob without overwrite: create operation returns conflict-style error.
- Restricted action without approval: Action Center returns blocked.
- Delete without confirmation: API returns confirmation required.
- Cross-container copy without target container: API returns target required.

## Completion Criteria

The Azure Blob module is considered enterprise-complete for this pass when:

- All dropdowns populate from discovered or persisted state.
- Customer can perform all major object operations from UI.
- Risky actions are previewed and gated by HITL/Guardian.
- AI can recommend and stage actions.
- DSPM can classify content and produce metadata/tag recommendations.
- Evidence, audit, and action history capture every major customer operation.
- Air-gapped/private mode remains functional for local scans, recommendations, previews, and evidence.
