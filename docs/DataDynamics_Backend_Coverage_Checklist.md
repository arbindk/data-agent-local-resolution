# DataDynamics — Backend Coverage & Verification Checklist

**Question answered:** "Is everything implemented, and is anything mocked?"

**Short answer:** The **UI is fully built and contains no mock data** — it reads live `/v1` data and shows empty states when the agent is offline. Verified against `internal/api/server.go`, **every endpoint the portal calls exists**. Most screens render **native backend data**; a small set is **derived** in the portal from live records because the backend has no dedicated rollup yet (these auto-switch to backend fields if/when added). Nothing returns fabricated values.

Legend: ✅ **native** (backend returns it) · ⚠️ **derived** (portal computes from live records) · ❌ **missing** (no source yet → empty state)

---

## 1. Endpoint existence (verified in server.go)

| Endpoint | Handler | Status |
|---|---|---|
| `/v1/analytics/enterprise-posture` | `enterprisePosture` | ✅ |
| `/v1/portal/options` | `portalOptions` | ✅ |
| `/v1/connectors/azureblob/config` (GET/POST) | `azureBlobConfigHandler` | ✅ |
| `/v1/connectors/azureblob/test` | `testAzureBlob` | ✅ |
| `/v1/connectors/azureblob/containers` | `listAzureBlobContainers` | ✅ |
| `/v1/connectors/azureblob/actions/execute` | `azureBlobActionExecute` | ✅ |
| `/v1/connectors/azureblob/writeback` (+`/apply-approved`) | `azureBlobWriteback` | ✅ |
| `/v1/tools/azureblob/workbench` | `azureBlobWorkbench` | ✅ |
| `/v1/scan/azureblob/jobs` (+ `/{id}/...`) | `azureBlobAsyncScanJobs` | ✅ |
| `/v1/dspm/catalog` / `/v1/dspm/content-scan` | `dspmCatalog` / `dspmContentScan` | ✅ |
| `/v1/ai/providers/catalog` / `/test` | `aiProviderCatalog` / `testAIProvider` | ✅ |
| `/v1/ai/azureblob/recommendations` | `azureBlobAIRecommendations` | ✅ |
| `/v1/evidence` / `/v1/evidence/package` | `portalEvidence` / `enterpriseEvidencePackage` | ✅ |
| `/v1/audit/events` | `portalAuditEvents` | ✅ |
| `/v1/actions/pending` / `/v1/actions/history` | `portalActionHistory` | ✅ |
| `/v1/guardian/decisions` | `guardianDecisions` | ✅ |
| `/v1/policy/artifact/trust` | `policyArtifactTrust` | ✅ |
| `/v1/agent/runtime/status` | `agentRuntimeStatus` | ✅ |
| `/v1/datasets` | — | ❌ none |
| `/v1/federation` | — | ❌ none |

**Confirmed in `enterprisePostureResponse`:** `metrics`, `risk_by_severity`, `trends` (date, evidence_records, high_risk_evidence, sensitive_findings, actions, async_records), `top_risks`, `recommendations`, `job_status`, `action_status`, `approval_status`. **Not present:** `framework_coverage`, `sovereignty`, `residency_zones`, `datasets`, `federation`.

---

## 2. Per-screen status + how to verify

For each: the verify **step** and the **expected** result. "Seed" steps use `azureblob-test-tool` (see test plan, Part C).

### Data Stores — ✅ native
- **Step:** Save a source; reload.
- **Expected:** Source persists via `/v1/connectors/azureblob/config` and appears from `/v1/portal/options`. **Discover containers** lists real containers from `/v1/connectors/azureblob/containers`.

### Blob Workbench — ✅ native
- **Step:** List blobs; select one; (Apply + write-back) run Set tags.
- **Expected:** Objects from `/v1/tools/azureblob/workbench`; action commits via `/v1/connectors/azureblob/actions/execute` + `/writeback`; re-listing shows the new tag.

### Risk Explorer — ✅ native
- **Step:** After a content scan, open the screen; click a row.
- **Expected:** Findings come from `/v1/evidence`; Decision Record assembles from the evidence record's decision/policy fields.

### Scans & Jobs / Agent Runs / Batch — ✅ native
- **Step:** Queue an async scan; pause/resume/cancel.
- **Expected:** Job lifecycle from `/v1/scan/azureblob/jobs(/{id}/...)`; checkpoints (pages, objects, retries, throttles) reflect live counters.

### Auto-Classification — ✅ native
- **Step:** Run content scan on seeded data.
- **Expected:** Entity tiles + findings from `/v1/dspm/*` + `/v1/evidence` (EMAIL/SSN/PAN/PCI/secret/finance).

### Approvals + Dashboard "Needs your decision" — ✅ native
- **Step:** Create a HITL action; open Approvals.
- **Expected:** Items from `/v1/actions/pending`; approve/block recorded; history via `/v1/actions/history`.

### Policy Governance — ✅ native
- **Step:** Open the screen; then tamper-test (Part D, T15).
- **Expected:** Bundle version/signature/hash from `/v1/policy/artifact/trust`; decisions from `/v1/guardian/decisions`; tampered policy shows **signature failed**.

### Evidence — ✅ native
- **Step:** Review table; Export package.
- **Expected:** Records from `/v1/evidence`; export from `/v1/evidence/package`.

### Integrations (AI) — ✅ native
- **Step:** Review providers; Test runtime; view recommendation.
- **Expected:** Providers from `/v1/ai/providers/catalog`; recommendation from `/v1/ai/azureblob/recommendations`; **auto-apply blocked**.

### Diagnostics — ✅ native
- **Step:** Open the screen.
- **Expected:** Health from `/v1/agent/runtime/status`; notifications from `/v1/audit/events`.

### Executive Dashboard — ✅ native (KPIs, trend, pipeline, severity) / ⚠️ derived (coverage ring, residency ring)
- **Step:** Run scans across time; open Executive.
- **Expected:** KPIs, **Posture trend graph**, governance pipeline, severity split all from `/v1/analytics/enterprise-posture` (incl. `trends`). **Regulatory coverage** and **Data residency ring** are ⚠️ derived (see below).

### Compliance / Regulatory coverage — ⚠️ derived
- **Source today:** computed in portal from `/v1/evidence` entity types (PII→GDPR/CCPA/DPDP, PHI→HIPAA, PCI/secret→PCI-DSS), governed-vs-total.
- **Verify:** seed PII + PCI + secret objects, run content scan → bars appear with sensible ratios.
- **To make native:** add `framework_coverage: [{name, coverage, controls}]` to the posture response; portal will prefer it automatically.

### Data Residency — ⚠️ derived
- **Source today:** zero-copy ⇒ local execution; cross-border counts from `/v1/guardian/decisions`; zones from saved-source regions.
- **Verify:** trigger a cross-region move (HITL) → block count increments.
- **To make native:** add `sovereignty: {local_execution_pct, cross_border_violations, cross_border_blocks, zones:[...]}` to posture.

### Data Passport (lineage) — ⚠️ derived
- **Source today:** assembled from `/v1/evidence` + `/v1/audit/events` matched to the object.
- **Verify:** classify then act on one object → its timeline shows discovery → classify → decision → action.
- **To make native (optional):** add `GET /v1/passport?object=…` returning the same assembly.

### Datasets — ⚠️ derived / ❌ no endpoint
- **Source today:** derived from saved sources + their classification findings.
- **To make native:** add `GET /v1/datasets`.

### Federation — ⚠️ derived / ❌ no endpoint
- **Source today:** single connected cell from runtime/trust; empty when offline.
- **To make native (multi-cell):** add `GET /v1/federation`.

---

## 3. The only "fallback" in the portal — and why it is not a mock

When the agent is unreachable, every fetch fails and the portal renders **empty states + the "Agent not connected" banner** — never fabricated numbers. The derived screens above are **computations over real agent records**, not invented data. There is no hardcoded business data anywhere in the build.

**Prove it in 30 seconds:** stop `policy-agent.exe` → every KPI reads 0 and the banner shows. Start it with seeded data → values populate. If anything were mocked, it would still show numbers while disconnected.

---

## 4. Net summary for the Go team

- **No backend code changes are required to run the portal.** Every called route exists.
- **To convert the 5 derived areas to native**, add these to the posture response (or as new endpoints): `framework_coverage`, `sovereignty`(+`zones`)/`residency_zones`, `datasets`, `federation`, and optionally `GET /v1/passport`. The portal already prefers the backend field the moment it appears — no UI change needed.
- **Everything else is native and unmocked today.**
