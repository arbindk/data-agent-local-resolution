# Everest Troubleshooting Runbook

## Purpose

This runbook helps support engineers, developers, testers, and customer-facing teams diagnose Everest issues across the portal, Azure Blob operations, connector runtimes, AI tools, DSPM scans, Guardian/HITL, evidence, and deployment modes.

The runbook is portal-first because customer testers may not want to use command-line tools. Developer-only checks are clearly marked as optional.

## First Five Checks

When something fails, start here:

1. Check the top-right health badges in the portal.
   - `Agent: healthy`
   - `Control Plane: healthy`
   - `Mode: dry_run` or another configured mode
2. Open Diagnostics.
   - Confirm backend health.
   - Confirm connector runtime health.
   - Confirm AI provider health.
   - Confirm current deployment mode.
3. Confirm the selected source, container, file/object, policy, job, or provider is not empty.
   - Everest should prefer dropdowns populated from discovered data.
   - Empty dropdowns usually mean discovery has not run, credentials are insufficient, or the runtime is unavailable.
4. Confirm dry-run versus live mode.
   - In dry-run mode, actions should show planned results without changing customer data.
   - In live mode, Guardian/HITL and confirmation rules should apply.
5. Review Evidence Center and Action Center.
   - Evidence shows what happened.
   - Action Center shows pending, approved, rejected, or failed actions.

## Severity Guide

| Severity | Meaning | Examples | Response |
| --- | --- | --- | --- |
| Sev 1 | Customer data at risk or portal unavailable | destructive action executed incorrectly, production API down | stop automation, preserve logs, escalate immediately |
| Sev 2 | Major workflow blocked | scans cannot run, Azure Blob unavailable, approvals stuck | troubleshoot within active session, assign owner |
| Sev 3 | Feature degraded | AI provider unavailable, one connector down, evidence export failed | use fallback mode, document workaround |
| Sev 4 | Cosmetic or documentation issue | label unclear, empty table confusing, minor UI bug | add to backlog after confirming no functional impact |

## Portal Health Badges

| Badge | Healthy Meaning | Unhealthy Meaning | Next Step |
| --- | --- | --- | --- |
| Agent | Local/runtime agent can respond | Connector or agent runtime unavailable | Check Connector Catalog and Diagnostics |
| Control Plane | Backend API is reachable | Portal cannot reach API | Refresh, check backend process, confirm API URL |
| Mode | Current execution mode | User may expect real changes but dry-run is active | Confirm desired mode before testing actions |

## Common Portal Symptoms

### Dropdowns Are Empty

Likely causes:

- Discovery has not been run yet.
- Source connection was not saved.
- Azure Blob credentials cannot list containers or blobs.
- Storage account firewall or private endpoint blocks access.
- Connector runtime is offline.
- The selected deployment mode blocks that connector or provider.

Portal checks:

- Go to Source Inventory and confirm the source exists.
- Use Azure Blob Workbench to list containers.
- Select the source again after refresh.
- Check Diagnostics for connector health.
- Check Evidence Center for discovery result.

Expected behavior:

- If discovery succeeds, dropdowns should populate with existing containers, blobs, jobs, policies, providers, and action IDs wherever possible.
- If discovery fails, the UI should show a clear error and the required next action.

### Check, Result, Detail Table Has No Rows

This table is intended to show runtime validation checks returned by a connector, AI provider, or manifest validation.

Empty rows usually mean:

- The runtime call succeeded but returned only summary cards, not detailed validation rows.
- The sample runtime does not provide detailed check records for that operation.
- The selected operation has not been invoked yet.
- The response payload had no `checks`, `results`, or equivalent validation array.

How to explain this to a customer:

- It is not automatically an error.
- The status card above the table is the primary result.
- The table becomes useful when the runtime returns per-check diagnostics such as auth, schema, policy, endpoint, permission, and dry-run checks.

Recommended product improvement:

- Show an empty-state message: `No detailed validation checks were returned for this operation.`
- Hide the table until an invocation has returned detail rows.
- Add sample rows in demo mode so customers understand the purpose.

### Buttons Work But Nothing Changes

Likely causes:

- `dry_run` mode is enabled.
- Action requires approval and is pending in Action Center.
- Guardian blocked the action.
- The user did not enter the required confirmation phrase.
- Backend returned a planned action but did not execute because autonomous apply is disabled.

Portal checks:

- Look at the top-right mode badge.
- Open Action Center and filter by pending actions.
- Open Guardian Agent to see policy decision.
- Open Evidence Center for action receipt.

Expected behavior:

- Dry-run should show planned impact and no customer data mutation.
- Pending actions should be visible to approvers.
- Blocked actions should include reason and policy name.

### Runtime Status Is `ok` But Data Looks Missing

Likely causes:

- Runtime connectivity is healthy, but selected scope is empty.
- Credentials can reach the account but cannot list a specific container or prefix.
- Object filters exclude the data.
- The connector returned partial data due paging, timeout, or permission.

Portal checks:

- Reset filters.
- Select a broader scope.
- Refresh source inventory.
- Use Azure Blob Workbench to list containers and blobs directly.
- Review job result details and evidence.

## Azure Blob Troubleshooting

### User Has Azure Access But Does Not Know Container Name

Expected portal behavior:

- The user adds or selects a saved Azure Blob source using simplified account/access fields.
- Everest discovers containers automatically when permissions allow.
- Container dropdown is populated from discovered containers.
- The user can select a container without knowing the name ahead of time.

If containers are not discovered:

| Possible Cause | How To Confirm | Resolution |
| --- | --- | --- |
| Connection string invalid | Test connection fails | Re-enter or rotate credential |
| Credential lacks list permission | Account reachable but no containers listed | Use account key, SAS with list permission, or RBAC with list permission |
| Firewall blocks access | Timeout or network error | Allow Everest network, private endpoint, or run local runtime inside network |
| Private endpoint DNS issue | Host resolves incorrectly | Fix private DNS zone or runtime placement |
| Storage account disabled | Auth/account error | Check Azure storage account status |

### Cannot List Containers

Portal checks:

- Azure Blob Workbench: run container discovery.
- Diagnostics: confirm Azure connector is available.
- Source Inventory: confirm source connection status.

Common root causes:

- SAS token does not include `list`.
- SAS token is scoped to one container and cannot list account-level containers.
- Account key is wrong or rotated.
- Storage account firewall allows only selected networks.
- Private endpoint is configured but Everest runtime is outside the private network.

Expected output when fixed:

- Container dropdown contains one or more container names.
- Evidence Center records successful discovery.
- Source Inventory shows the Azure Blob source as available.

### Cannot List Blobs

Likely causes:

- Container name is wrong.
- SAS token allows container list but not object list.
- Prefix filter excludes all objects.
- Container has no blobs.
- Archive or immutable policies affect specific operations.

Expected output when fixed:

- Blob/file ID dropdown contains existing blob names.
- Metadata, tags, size, tier, and last modified values are visible where available.

### Upload Works But Scan Does Not Find The File

Likely causes:

- Inventory has not been refreshed after upload.
- Scan scope points to a different container or prefix.
- Uploaded file type is excluded by scan settings.
- Content scan tier is metadata-only.

Portal fix:

1. Refresh Azure Blob Workbench list.
2. Refresh Source Inventory.
3. Select the uploaded object from dropdown.
4. Run metadata scan first.
5. Run content or tiered scan.

### Delete, Archive, Or Tier Change Does Not Execute

Likely causes:

- Dry-run mode.
- Guardian requires approval.
- Confirmation phrase missing.
- Storage account policy blocks operation.
- Object has legal hold, immutability, lease, or versioning behavior.

Expected enterprise behavior:

- Destructive actions are never silent.
- The portal should show planned impact, approval status, and final receipt.
- If blocked by Azure, Evidence Center should include Azure error summary.

## Connector Runtime Troubleshooting

### Sample Plugin Runtime Shows PowerShell `NativeCommandError`

Observed output may look like a PowerShell error even when the runtime is actually listening:

```text
Everest sample plugin runtime listening on http://localhost:7071
connector endpoint: http://localhost:7071/connector
AI endpoint: http://localhost:7071/ai
```

Reason:

- Go logging often writes to stderr.
- PowerShell may display stderr from native commands as `NativeCommandError`.
- If the endpoint is listening and the portal can invoke it, this is not necessarily a runtime failure.

Expected behavior:

- Connector Catalog runtime status becomes `ok`.
- AI Tools runtime status becomes `ok`.
- `http://localhost:7071/connector` and `http://localhost:7071/ai` are available to the backend.

### Connector Invocation Fails

Likely causes:

- Runtime endpoint URL is wrong.
- Runtime process is not running.
- Manifest schema is invalid.
- Operation ID is not supported by the selected connector.
- Deployment mode blocks external process or HTTP runtime.

Portal checks:

- Connector Catalog: select connector.
- Confirm runtime endpoint.
- Validate sample manifest.
- Select action ID from dropdown.
- Invoke runtime in dry-run first.

Expected output:

- Runtime status card shows `ok`.
- Connector name, runtime kind, dry-run flag, and contract version are visible.
- Detailed check rows appear when provided by the runtime.

### Custom Connector Does Not Appear

Likely causes:

- Manifest was not loaded.
- Manifest ID duplicates another connector.
- Required fields are missing.
- Capability names do not match supported operations.
- Runtime kind is not allowed in the current deployment mode.

Fix:

- Validate manifest.
- Confirm connector ID, display name, runtime kind, actions, and capabilities.
- Refresh Connector Catalog.
- Confirm air-gapped/customer-code flags match the environment.

## AI Tools Troubleshooting

### Provider List Is Empty

Likely causes:

- AI providers have not been configured.
- Provider catalog failed to load.
- Deployment mode disables cloud AI.
- Private LLM gateway is unreachable.

Portal checks:

- AI Tools: Refresh Providers.
- Load Config.
- Select sample runtime provider.
- Test Provider.
- Diagnostics: confirm AI runtime health.

Expected behavior:

- Provider catalog shows built-in, private, custom, or local providers.
- Data boundary is visible for each provider.
- Air-gapped support is clearly marked.

### Test Provider Succeeds But Prompt Result Is Missing

Likely causes:

- Runtime returned health but no generated content.
- Prompt sending is disabled by policy.
- Provider is configured for metadata-only mode.
- Sample provider returns static or minimal response.

Expected behavior:

- Runtime status shows `ok`.
- The UI should show whether prompt was sent.
- Evidence should record the provider, task, boundary, and summary.

### AI Recommendation Does Not Execute

Likely causes:

- AI is advisory only.
- Autonomous apply is blocked.
- Guardian requires HITL.
- The recommendation is low confidence.
- Policy prevents action.

Expected behavior:

- Recommendation should appear in Evidence Center.
- Pending action should appear in Action Center if HITL is required.
- No customer data should change until approval and execution conditions are met.

## DSPM And Content Scan Troubleshooting

### No Findings Returned

Likely causes:

- Selected content does not match detectors.
- Scan tier is metadata-only.
- Keyword packs are disabled.
- Custom keyword pattern is wrong.
- File type is unsupported.
- Content was not available to scanner due permissions.

Portal checks:

- Select tiered content scan.
- Enable built-in keyword pack.
- Add a simple custom keyword pattern for test.
- Use a known test file with expected sensitive sample content.
- Confirm scan scope includes the object.

Expected output:

- Findings should include type, severity, confidence, tier, and evidence reference.
- Keyword matches should include pattern or pack name.

### Keyword Pattern Matching Does Not Work

Checklist:

- Pattern is enabled.
- Pattern syntax is valid.
- Case sensitivity matches expectation.
- Exclusions do not remove the target.
- Scan tier includes rule/keyword analysis.
- File content is readable as text or supported parsed content.

Expected behavior:

- Built-in packs should work without custom patterns.
- Custom patterns should produce named findings.
- Invalid patterns should fail validation before scan starts.

### BERT Or Model Scan Does Not Produce Model Findings

Current expected behavior:

- Model profiles may be represented in catalog/configuration before real local model inference is fully wired.
- Rule and keyword findings can still run.

Enterprise target behavior:

- Local or configured model runtime should be invoked.
- Model version, profile, confidence, and explanation should be stored.
- Air-gapped environments should use local models only.

## Jobs And Schedules Troubleshooting

### Job Stuck In Running

Likely causes:

- Worker crashed after job started.
- Connector paging stalled.
- Source throttling.
- Long-running deep scan.
- No heartbeat timeout implemented yet.

Portal checks:

- Scans and Jobs: open job details.
- Check last update time.
- Check completed versus total task count.
- Open Diagnostics for worker/runtime status.

Expected enterprise behavior:

- Job should show heartbeat.
- Stale job should be recoverable, cancellable, or marked failed.
- Checkpoints should allow resume where possible.

### Job Completes But Evidence Missing

Likely causes:

- Evidence write failed.
- Job ran in a mode that does not create evidence yet.
- Evidence filter is hiding records.
- Subject ID mismatch between job and evidence.

Fix:

- Refresh Evidence Center.
- Filter by job ID, source ID, or object ID.
- Re-run in dry-run with evidence enabled.
- Developer should check API handler writes evidence for that job type.

## Guardian And HITL Troubleshooting

### Action Is Pending Forever

Likely causes:

- No approver assigned.
- Approval routing rule missing.
- Approver lacks permission.
- Approval expiration not visible.
- Action Center filter hides pending item.

Portal checks:

- Open Action Center.
- Filter by `pending`.
- Check assigned approver or group.
- Check policy decision reason.
- Approve or reject with comment.

Expected behavior:

- Pending actions should show who can approve.
- Rejected actions should not execute.
- Approved actions should move to execution or planned execution.

### Guardian Blocks Action

Likely causes:

- Destructive action.
- High-risk finding.
- AI recommendation below confidence threshold.
- Action outside deployment boundary.
- Policy requires manual review.

Expected behavior:

- Guardian decision shows policy name and reason.
- The action is either denied or moved to HITL.
- Evidence records the decision.

## Evidence Center Troubleshooting

### Evidence Table Is Empty

Likely causes:

- No evidence-producing action has run.
- Current filters hide results.
- Backend evidence persistence is not enabled for that module.
- Operation returned only transient UI result.

Portal checks:

- Clear filters.
- Run a simple source discovery.
- Run a dry-run action.
- Run a content scan.
- Refresh Evidence Center.

Expected behavior:

- At minimum, source discovery, scan, AI recommendation, policy decision, approval, and action execution should produce evidence in the enterprise target state.

### Evidence Shows Redacted Values

This is usually correct.

Reason:

- Sensitive content should not be stored in full by default.
- Evidence should preserve enough context for audit without leaking secrets, PII, PHI, PCI, or confidential content.

Expected behavior:

- Evidence shows finding type, location hint, severity, confidence, policy, action, and decision.
- Raw content is hidden unless customer policy allows retention.

## Deployment Mode Troubleshooting

### Works In Local Dev But Fails In Customer Network

Likely causes:

- Storage account firewall.
- Private endpoint DNS.
- Proxy or outbound network restriction.
- AI provider endpoint not reachable.
- Connector runtime deployed outside data boundary.

Fix:

- Move runtime closer to customer data.
- Use private network endpoint.
- Disable cloud AI or switch to private LLM gateway.
- Confirm allowed ports and DNS.

### Air-Gapped Mode Blocks Cloud Provider

Expected behavior:

- Air-gapped mode should not call cloud AI, SaaS telemetry, online update services, or internet-hosted connector endpoints.

Fix:

- Use local connector runtime.
- Use local/private model provider.
- Import signed connector packages offline.
- Export evidence bundles manually.

## Optional Developer Checks

Use these only when command-line access is acceptable in a development environment:

| Goal | Check |
| --- | --- |
| Confirm backend compiles | `go test ./...` |
| Run portal backend | project-specific backend run command |
| Run sample plugin runtime | `go run ./cmd/sample-plugin-runtime` |
| Run Azure Blob test tool | `go run ./cmd/azureblob-test-tool` with safe test credentials |
| Search endpoint handlers | inspect `internal/api` |
| Inspect DSPM engine | inspect `internal/dspm` |
| Inspect portal UI | inspect `web/index.html` |

Do not run destructive Azure operations from CLI against customer data unless the test plan explicitly allows it.

## Evidence To Collect For Support

For any bug report, capture:

- Deployment mode.
- Browser and portal URL.
- User role.
- Source connector type.
- Source/container/object selected.
- Operation attempted.
- Dry-run or live mode.
- Guardian decision.
- Approval status.
- Evidence record ID, if available.
- Error text from the portal.
- Timestamp and timezone.
- Whether sample runtime or real customer runtime was used.

## Customer-Safe Recovery Actions

| Situation | Safe Recovery |
| --- | --- |
| UI stale | Refresh page and reload provider/source lists |
| Dropdown empty | Re-run discovery and clear filters |
| Job stuck | Cancel if supported, then re-run smaller scope |
| AI unavailable | Continue with deterministic policy/rule checks |
| Connector unavailable | Switch to local runtime or validate endpoint |
| Action pending | Approve/reject in Action Center |
| Action blocked | Review Guardian decision and policy |
| Evidence missing | Re-run operation in dry-run and check filters |

## Escalation Template

Use this format when escalating to engineering:

```text
Title:
Environment:
Deployment mode:
Tenant/customer:
User role:
Module:
Source/connector:
Operation:
Expected result:
Actual result:
Portal health badges:
Guardian decision:
Approval status:
Evidence record:
Steps to reproduce:
Screenshots:
Logs or API response summary:
Data sensitivity:
Customer impact:
```

## Known Product Gaps To Track

- Durable persistence for all UI state and histories.
- Background worker recovery and job heartbeat.
- Full model runtime integration for local BERT/domain models.
- Evidence persistence coverage for every module.
- RBAC and tenant isolation hardening.
- More explicit empty-state messages for check/result/detail tables.
- Production deployment packaging and upgrade flow.
- Offline connector package import and signature validation.
