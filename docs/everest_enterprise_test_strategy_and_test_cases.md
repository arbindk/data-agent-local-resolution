# Everest Enterprise Test Strategy And Test Cases

Version: 3.0  
Scope: Enterprise validation for Azure Blob sensitive-data operations, governed action execution, dashboard intelligence, AI recommendations, HITL approvals, evidence, and connector extensibility.

## 1. Test Strategy

Enterprise validation must prove that Everest can support real customer operations on sensitive storage environments. The test suite must validate:

- source isolation across multiple Azure Blob accounts
- sensitive-data discovery and classification
- tiered content scan behavior
- dashboard posture and recommendations
- AI-generated action plans
- Guardian/HITL decisions
- preview-first action safety
- approved action execution
- evidence and audit completeness
- async scan checkpoint and resume
- air-gapped and locked-down deployment modes
- connector/provider extensibility contracts

Connectivity tests are prerequisites only. They are not sufficient product validation.

## 2. Test Data Requirements

Use controlled enterprise test containers with realistic file types and content patterns.

| Dataset | Example files | Purpose |
| --- | --- | --- |
| PII | customer_export.csv, passport_scan.pdf, employee_master.xlsx | Personal data discovery, masking, severity scoring. |
| PHI | patient_notes.txt, insurance_claim.pdf | Healthcare-sensitive detection and HITL flow. |
| PCI | payment_extract.csv with tokenized and malformed card-like values | Payment-data policy checks and false-positive handling. |
| Secrets | appsettings.json, .env, key_dump.txt | API key/token/password detection. |
| Legal | contract_archive.pdf, litigation_hold_notice.docx | retention/legal-hold action governance. |
| Engineering | source_bundle.zip, terraform.tfstate | secrets and infrastructure exposure checks. |
| Ransomware indicators | unusual extensions, ransom-note-like text | quarantine recommendation and security escalation. |
| Public exposure | public-container objects with sensitive metadata | exposure risk and dashboard posture. |
| Large scale | thousands of objects across prefixes | async scan, pagination, checkpoint, resume. |
| Cross-region | source and target region metadata | residency and transfer-policy validation. |

## 3. Core Enterprise Scenarios

### TC-001 Sensitive PII In Public Container

Steps:

1. Select an Azure Blob source containing a public or broad-access container.
2. Run container discovery.
3. Run metadata scan.
4. Run DSPM rules/content scan on selected objects.
5. Open Enterprise Dashboard.
6. Open AI recommendations.
7. Preview recommended tagging/quarantine action.
8. Submit restricted action to HITL if required.

Expected:

- Dashboard shows public exposure and sensitive-data risk.
- Enterprise posture endpoint shows sensitive findings and high-risk evidence counts.
- PII findings are grouped by severity.
- AI recommendation includes rationale and target object scope.
- Quarantine/delete requires approval.
- Evidence records include scan, finding, recommendation, decision, and action preview.

### TC-002 Secrets In Engineering Artifact

Steps:

1. Select source and prefix containing `.env`, JSON config, or tfstate-like files.
2. Run tiered content scan.
3. Use keyword/entity pattern matching.
4. Review DSPM findings and suggested metadata/tags.
5. Preview tag update.

Expected:

- SECRET/API key/token entity findings appear.
- Dashboard flags high-risk engineering artifact exposure.
- Top risks include secret-related evidence when severity is high or critical.
- Suggested tags include restricted/confidential classification.
- Approved tagging can be applied without exposing raw secrets.

### TC-003 PHI In Incorrect Container

Steps:

1. Select healthcare-sensitive dataset.
2. Run content scan.
3. Ask AI for remediation recommendation.
4. Review Guardian decision.
5. Preview move-prep or quarantine.

Expected:

- PHI finding appears with evidence summary.
- Recommendation is approval-gated.
- Guardian blocks direct destructive action.
- Evidence shows policy context and HITL requirement.

### TC-004 Cross-Border Copy Or Move

Steps:

1. Select a sensitive object with source region metadata.
2. Prepare copy or move-prep to target region/container.
3. Run preview.
4. Submit approval if policy allows HITL.

Expected:

- Dashboard shows residency risk.
- Guardian blocks or routes action to HITL.
- Direct apply is not allowed without required approval.
- Evidence records include source region, target region, policy, and decision.

### TC-005 Legal Hold And Archive Tier

Steps:

1. Select legal/contract records.
2. Run DSPM scan.
3. Request archive tier transition.
4. Preview and attempt apply.

Expected:

- Legal-hold/retention metadata is recognized.
- Archive action is blocked or approval-gated.
- Dashboard recommends review by records owner.
- Evidence records retain action result and decision.

### TC-006 Rehydrate Sensitive Archive

Steps:

1. Select archived sensitive object.
2. Request rehydrate with standard and high priority.
3. Preview impact.
4. Submit approval if restricted.

Expected:

- Rehydrate priority appears in preview.
- Sensitive class requires HITL where policy requires.
- Action receipt/evidence is created after approved execution.

### TC-007 Ransomware Indicator Response

Steps:

1. Scan prefix containing suspicious extension patterns or ransom-note-like content.
2. Review dashboard risk cluster.
3. Generate AI recommendation.
4. Preview quarantine tagging or copy-to-investigation action.

Expected:

- Dashboard highlights high-risk cluster.
- Recommendation prioritizes containment and investigation.
- Unsafe delete is blocked.
- Evidence links findings to recommended action.

### TC-008 Multi-Source Isolation

Steps:

1. Save Azure Blob source A.
2. Save Azure Blob source B.
3. Select source A in Workbench and list containers.
4. Select source B in Workbench and list containers.
5. Run a preview action on each source.

Expected:

- Each source uses source-specific account access.
- Workbench status shows saved source access.
- Containers and objects are scoped to the selected source.
- Evidence records contain correct source ID.

### TC-009 Async Large Account Scan Resume

Steps:

1. Queue async scan for a large source with multiple containers/prefixes.
2. Resume job.
3. Allow worker to list Azure Blob pages.
4. Review checkpoint updates.
5. Pause the job.
6. Resume again from the persisted checkpoint.
7. Cancel or complete.

Expected:

- Job status transitions are visible.
- Azure Blob pages are listed by the worker.
- Portal state receives discovered objects.
- Normalized result artifact is written under `EVEREST_AZUREBLOB_ASYNC_RESULTS_PATH`.
- The `Records` action opens `/v1/scan/azureblob/jobs/<job_id>/normalized`.
- Checkpoint persists to configured job state path after each page.
- Worker lease owner, heartbeat, and lease expiry are visible on the job.
- Retry count and throttle count are tracked when Azure returns retryable errors.
- Stale worker leases can be reclaimed and the job moves to paused for safe resume.
- Resume continues from latest checkpoint/page marker.
- Pause stops progress between pages without deleting job state.
- Dashboard shows job posture.
- Enterprise posture shows async active jobs and async records.
- Evidence and audit records exist for lifecycle changes.

### TC-010 Air-Gapped Operation

Steps:

1. Configure local file secret provider.
2. Configure local/private AI provider.
3. Save Azure Blob source using customer-controlled network path or emulator-equivalent enterprise endpoint.
4. Run scan and Workbench preview.
5. Export evidence.

Expected:

- Runtime status shows air-gapped-compatible provider posture.
- No external cloud AI dependency is required.
- Secrets are not displayed.
- Evidence export is available for offline audit transfer.

### TC-011 Locked-Down Secret Operation

Steps:

1. Configure environment secret provider.
2. Pre-provision source-specific secret environment variable.
3. Select saved source and run Workbench list operation.
4. Attempt to overwrite secret from portal.

Expected:

- Read operations can use pre-provisioned secret.
- Portal cannot overwrite read-only secret provider.
- Dashboard shows provider is read-only.
- Evidence contains operational results without secret values.

### TC-012 AI-Assisted Bulk Tagging

Steps:

1. Run scan on mixed sensitivity dataset.
2. Open AI recommendations.
3. Review recommended tags and metadata.
4. Apply recommendation to Workbench form.
5. Preview bulk tagging action.
6. Submit HITL for restricted objects.

Expected:

- Recommendations are grouped by risk and confidence.
- Safe tags can be previewed and applied where allowed.
- High-risk or destructive actions require approval.
- Action Center and Evidence Center update.

### TC-013 Delete Prevention

Steps:

1. Select sensitive object.
2. Choose delete action.
3. Run preview.
4. Attempt apply without required confirmation/approval.

Expected:

- Preview shows expected impact without mutation.
- Apply is blocked without confirmation and approval.
- Dashboard records prevented destructive action.
- Object remains in Azure Blob.

### TC-014 Auditor Evidence Review

Steps:

1. Complete any governed scan/action workflow.
2. Open Evidence Center.
3. Filter by source, job, object, policy, and action.
4. Export evidence package.

Expected:

- Evidence includes source ID, actor, role, object, operation, policy decision, timestamp, status, and action receipt.
- Raw secrets are never exported.
- Raw content is excluded unless governed export policy allows it.
- `/v1/evidence/package` returns enterprise evidence package with sources, objects, actions, jobs, policies, and audit records.

### TC-015 Custom Connector Contract

Steps:

1. Register or validate connector manifest.
2. Invoke connector runtime contract in preview mode.
3. Review contract checks and runtime audit.
4. Confirm connector appears in catalog as customer-owned/extensible.

Expected:

- Connector contract validates.
- Runtime audit records invocation.
- Unsupported actions are rejected clearly.
- Connector does not bypass Guardian/HITL or evidence requirements.

### TC-016 Enterprise Authorization Audit And Enforce

Steps:

1. Run with `EVEREST_AUTH_MODE=audit`.
2. Send a non-preview Workbench delete/tier action as role `data_analyst`.
3. Review audit events.
4. Run with `EVEREST_AUTH_MODE=enforce`.
5. Repeat the same action as `data_analyst`.
6. Repeat the action as `security_officer`.

Expected:

- In audit mode, unauthorized action is recorded as authorization audit event.
- In enforce mode, unauthorized action returns HTTP 403.
- Authorized role can proceed to normal Guardian/HITL/confirmation checks.
- Evidence and audit records do not expose secrets or raw object content.

## 4. Acceptance Criteria

- All sensitive-data scenarios produce dashboard posture changes.
- `/v1/analytics/enterprise-posture` returns metrics, top risks, and recommendations.
- `/v1/playbooks/enterprise` returns scenario playbooks for sensitive-data and governed-action operations.
- Every preview/apply action produces evidence and audit records.
- High-risk, destructive, and cross-border actions are blocked or HITL-gated.
- Dynamic dropdowns populate from saved sources, containers, blobs, jobs, policies, and providers.
- Source-specific credentials prevent cross-account leakage.
- Async scan jobs persist checkpoint state and resume safely.
- Secret provider posture is visible without exposing secrets.
- Authorization audit/enforce mode works for high-risk source, Workbench, and async scan actions.
- Enterprise evidence package can be generated without exposing raw secrets.
- Official documentation contains product behavior, not informal presentation guidance.
