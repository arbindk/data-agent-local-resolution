# DataDynamics Portal — UI Testing & Client Demo Script

**Product:** DataDynamics for Azure Blob (Data Agent Cell)
**Build under test:** `DataDynamics.dc.html` (served as `web/index.html`)
**Version:** 2.2 — browser-only walkthrough

## Purpose

Browser-only test steps for the redesigned DataDynamics portal. No PowerShell, curl, Postman, or CLI is required during the walkthrough. Use it for QA, client demos, and manager review.

```text
Portal URL (deployed):  http://localhost:8080/
Portal URL (dev file):  http://localhost:8080/DataDynamics.dc.html
```

Two services must be running:

```text
policy-agent.exe   -> http://localhost:8080   (data agent + serves the portal)
platform-api.exe   -> http://localhost:9090   (control plane: dashboard, approvals, governance)
```

## What changed from the previous (Everest) script

The portal was redesigned and rebranded. Screens were renamed and regrouped, and **all data is now live from the agent** — there is no demo/mock data. When the agent is unreachable, screens show real empty states and a clear "Agent not connected" banner, instead of fabricated numbers.

Screen name mapping (old -> new):

```text
Enterprise Dashboard      -> Executive            (group: Insights)
Architecture Flow         -> Pipelines            (group: Execution)
Source Inventory          -> Data Stores          (group: Discovery)
Azure Blob Workbench      -> Blob Workbench        (group: Discovery)
Scans & Jobs              -> Scans & Jobs          (group: Execution)
Agent Working Store       -> Agent Runs            (group: Execution)
DSPM / Intelligent Tagging-> Auto-Classification   (group: Governance)
Action Center             -> Approvals             (group: Governance)
Guardian Agent            -> Policy Governance      (group: Governance)
Evidence Center           -> Evidence              (group: Trust & proof)
(new)                     -> Risk Explorer         (group: Discovery)
(new)                     -> Data Residency        (group: Trust & proof)
(new)                     -> Data Passport         (group: Trust & proof)
(new)                     -> Datasets / Federation (group: Sharing)
AI Tools / Connectors     -> Integrations          (group: Insights)
Diagnostics               -> Diagnostics           (group: Insights)
```

Governed-work controls that the old script placed on an "Onboarding" screen (**Action mode** = Preview/Apply and **Write-back enabled**) now live on the **Blob Workbench** mode bar and gate every object action.

## Navigation reference (left rail, 18 screens)

```text
Discovery     : Data Stores | Blob Workbench | Risk Explorer
Execution     : Pipelines | Agent Runs | Scans & Jobs
Governance    : Policy Governance | Approvals | Auto-Classification
Trust & proof : Data Residency | Data Passport | Evidence
Sharing       : Datasets | Federation
Insights      : Executive | Compliance | Integrations | Diagnostics
```

## Test data modes

**Mode 1 — Connected Azure Blob.** The portal can test access, list blobs, run scans, classify, and preview/apply governed actions. Requires a storage account, an approved secret in the agent's secret provider, and one or more containers.

**Mode 2 — No-Azure walkthrough.** The portal still demonstrates the full governed UI path, gates, and empty states. Live Azure operations show a clear access/configuration message until a source is connected.

---

## Test Case 1: Shell, connection, and identity

### Steps
1. Open the portal URL.
2. Confirm the left rail shows all six groups and 18 items listed above.
3. Check the sidebar footer status dot.
4. Click the avatar (top-right) to open **Connection & identity**.
5. Set Agent `http://localhost:8080`, Platform `http://localhost:9090`, and Actor / Role / Tenant.
6. Click **Save & reconnect**.

### Expected result
- Portal loads without a blank screen; the Data Dynamics logo is visible.
- Footer dot is **green / "Agent connected"** when the agent is reachable, **amber / "Agent offline"** otherwise.
- When offline, a top banner reads "Agent not connected" with a Connection settings shortcut, and KPIs read 0 — no fabricated data.
- After saving, the portal reconnects and live values populate.

### What it proves
The portal is a stable, installable shell that reads live agent data and degrades honestly when disconnected.

---

## Test Case 2: Executive dashboard

### Steps
1. Open **Executive** (Insights group).
2. Review the four KPI cards: Objects in scope, Sensitive findings, Governed actions, Awaiting approval.
3. Review **Regulatory coverage** (per-framework bars).
4. Review **Governance pipeline** (Discovered -> Sensitive -> Actions taken -> Approved).
5. Review the **Data residency** ring (local-execution %).
6. Review **Priority actions** and click one.

### Expected result
- KPIs show live counts (0 when nothing has been scanned yet).
- **The numbers are real, not mock.** They are aggregated by the agent's `/v1/analytics/enterprise-posture` endpoint from the portal state store: saved Azure sources, discovered containers, listed blobs, recorded evidence, audit events, action records, and async scan jobs. With no agent connected, every value is 0 and the "Agent not connected" banner shows.
- Regulatory coverage shows a bar per framework returned by the agent, or an explanatory empty state if none.
- Pipeline bars scale to live counts.
- Residency ring shows the live local-execution percentage, or "—" if unavailable.
- Priority actions come from the agent; clicking one routes to Risk Explorer. If there are none, an empty-state line explains why.

### What it proves
DataDynamics opens on a board-ready posture view built entirely from live signals.

---

## Test Case 3: Connect an Azure Blob data store

### Steps
1. Open **Data Stores** (Discovery group).
2. In **Connect an Azure Blob source**, enter Storage account, Cloud suffix, Containers (comma-separated, optional), Prefix scope, Data store ID.
3. Set **Action mode** (Preview or Autonomous governed) and **Write-back mode**.
4. Click **Test access**.
5. Click **Save source**.
6. Review the **Saved sources** list.

### Expected result
- No raw connection string is required — only friendly fields.
- **Test access** reports reachable (connected mode) or a clear "Agent unreachable / check connection settings" message.
- **Save source** adds a card to Saved sources showing account, region, container count, action mode, write-back, and credential state. Saved sources are read live from the agent's portal state (`/v1/portal/options`).
- When nothing is saved yet, Saved sources shows an explicit empty state.
- Each saved card has **Open in Workbench**.

### What it proves
Technical connection-string complexity is hidden; the agent builds and stores per-source access, and the portal reflects real saved state.

---

## Test Case 3b: Discover and select containers (users who don't know container names)

### Steps
1. Open **Data Stores**; ensure a source with valid access is saved (Test Case 3).
2. Under **Don't know your containers?**, click **Discover containers**.
3. Review the discovered container list.
4. Tick one or more containers.
5. Click **Use selected containers**.
6. Confirm the **Containers** field is populated with the chosen names; click **Save source**.
7. Open **Blob Workbench** and confirm the same containers are selectable.

### Expected result
- **Discover containers** calls the agent (`GET /v1/connectors/azureblob/containers`), which lists containers from the live Azure account and persists them to portal state.
- Discovered containers appear as a checkbox list (button shows "Discovering…" while in flight).
- **Use selected containers** writes the ticked names into the source scope; a confirmation toast appears.
- Without valid access, a clear "Discovery unavailable — connect a source first" message appears; the page stays usable.
- The selected containers carry forward to Blob Workbench and scan scope.

### What it proves
Users never need to know container names up front — the agent discovers them and the user selects a governed scope, exactly as the backend `ListContainers` utility supports.

---

## Test Case 4: Blob Workbench — listing and inspection

### Steps
1. Open **Blob Workbench** (Discovery group), or click **Open in Workbench** from a saved store.
2. Choose Source, Container, and an optional Prefix.
3. Click **List blobs**.
4. Click a row in the object table.
5. Review the inspector (size, tier, tag count) and the governed-action grid.

### Expected result
- Source/container selectors are populated from saved stores.
- Object table lists blobs with size and access tier (connected mode), or a clear access message otherwise.
- Selecting an object shows its metadata and a grid of governed actions; restricted/destructive actions (Move cross-region, Delete) carry an **HITL** marker.

### What it proves
The Workbench supports object inspection and governed operations using saved source context.

---

## Test Case 5: Governed-work mode bar (Preview vs Apply, Write-back)

### Steps
1. In **Blob Workbench**, locate the mode bar: **Action mode** (Preview | Apply) and **Write-back enabled**.
2. Keep **Preview** selected; select an object; click **Set tags** (a safe action).
3. Read the preview card; click **Dismiss**.
4. Switch **Action mode** to **Apply** but leave **Write-back** off; preview the same action.
5. Enable **Write-back**; preview again.

### Expected result
- The mode bar shows a live note: "Plan only — no changes are written" (Preview), "Apply, write-back off — still plan only" (Apply + write-back off), or "Apply + write-back — mutations gated by Guardian & approval" (both on).
- In Preview or Apply-without-write-back, the preview decision is **PLAN** (blue) and the card states the object is not modified — there is **no Apply button**.
- With Apply + write-back, a safe action shows **ALLOW** and an **Apply approved action** button appears.

### What it proves
Apply mode alone never mutates data; write-back must be explicitly enabled, and the UI never silently switches from plan to apply.

---

## Test Case 6: Workbench — restricted and destructive gating

### Steps
1. In **Blob Workbench**, set Apply + Write-back on.
2. Select an object; click **Move (cross-region)**; review the preview; click **Submit for HITL approval**.
3. Select a flagged/high-risk object; click **Delete**; review the preview.

### Expected result
- **Move (cross-region)** returns **HITL REQUIRED**, offers **Submit for HITL approval**, and on submit routes to **Approvals** with a confirmation toast — nothing changes yet.
- **Delete** on a flagged object returns **BLOCK** with a policy explanation and offers only **Dismiss** (no apply).

### What it proves
Restricted actions require human approval, and destructive actions on protected objects are blocked outright.

---

## Test Case 7: Scans & Jobs — async lifecycle

### Steps
1. Open **Scans & Jobs** (Execution group).
2. Choose a Source and Mode (Metadata scan / Content scan), then click **Queue async scan**.
3. Review the job cards (progress bar + checkpoints: Pages, Objects, Lease, Retries, Throttles).
4. Use **Pause / Resume / Cancel** on a running job.

### Expected result
- Queue posts a scan to the agent (connected mode) and the job list refreshes; offline, a clear message appears.
- Each job shows live status, a progress bar, and checkpoint counters.
- Pause/Resume/Cancel update the job status via the agent.

### What it proves
The portal drives a checkpoint-aware async scan lifecycle from the UI; the agent runs the work.

---

## Test Case 8: Agent Runs and Batch detail

### Steps
1. Open **Agent Runs** (Execution group).
2. Review KPIs (Active leases, Jobs tracked, Objects processed, Local execution) and the lease/run table.
3. Click **Open** on a run.
4. In the **Batch run** detail, review the pipeline steps, Guardian decisions, and batch summary.
5. Use **Roll-up provenance** and **In Compliance** to jump to Evidence / Compliance.

### Expected result
- KPIs and the table reflect live jobs; empty state when none.
- Batch detail header shows the selected run's store, lease, policy, and counts.
- Pipeline steps, decisions, and summary stats are present; **Run again** queues a re-run.

### What it proves
Agent execution is transparent end-to-end, from lease to per-object decision to summary roll-up.

---

## Test Case 9: Auto-Classification

### Steps
1. Open **Auto-Classification** (Governance group).
2. Review the entity KPI tiles (derived from live findings).
3. Review the sensitive-findings table (object, entity, severity, recommended action).
4. Review the detection model profiles.
5. Click **Run content scan**.

### Expected result
- Entity tiles and the findings table populate from live evidence; an empty state appears when there are no findings.
- Each finding shows a severity chip and a recommended action that routes to Approvals.

### What it proves
Classification, entities, and recommended tags are surfaced from real scan output, near the data.

---

## Test Case 10: Risk Explorer and Decision Record

### Steps
1. Open **Risk Explorer** (Discovery group).
2. Review the summary header (severity distribution, high-risk count, coverage, attention).
3. Use the left filters (Data store, Risk level, Entities).
4. Click a file row to open the **Decision Record** slide-over.
5. Review policy lineage, Guardian evaluation, entities, and action; try **Copy receipt**, **Open audit log**, **Attach to Passport**.

### Expected result
- The table lists live findings; empty state when none.
- The slide-over shows the selected object's real decision, risk, entities, and policy lineage; sections hide when data is absent.
- The three footer actions copy a receipt, jump to Evidence, and jump to Data Passport respectively.

### What it proves
Every flagged object has a provable, drill-through decision record.

---

## Test Case 11: Approvals (HITL queue)

### Steps
1. Open **Approvals** (Governance group).
2. Review the KPI strip (Awaiting approval, Approved today, Blocked by policy).
3. For a queued request, review operation, object, source, policy, requester, and rationale.
4. Click **Approve & execute** or **Block**; use **View data passport**.

### Expected result
- The queue reflects live pending actions; empty state when none.
- Approve/Block record a decision (toast confirmation) and the posture counters reflect it.

### What it proves
Risky actions are queued for human decision with full policy rationale before anything runs.

---

## Test Case 12: Policy Governance (Guardian)

### Steps
1. Open **Policy Governance** (Governance group).
2. Review the active policy bundle card (version, signature, hash, distribution).
3. Review deployment posture (cloud-connected / private network / hybrid / air-gapped).
4. Review recent Guardian decisions (allow / block / HITL).

### Expected result
- Bundle metadata is read from the agent's policy-artifact trust endpoint; "—" where unavailable.
- The decisions list reflects live Guardian activity; empty state when none.

### What it proves
Policy enforcement is a visible, signed, non-bypassable control.

---

## Test Case 13: Data Residency

### Steps
1. Open **Data Residency** (Trust & proof group).
2. Review the local-execution ring and the residency stats.
3. Review residency zones and recent cross-border activity.

### Expected result
- Ring and stats reflect live sovereignty data, or "—" / empty state when unavailable.
- Zones and cross-border events list real entries when present.

### What it proves
Residency and cross-border control are first-class, measurable governance outcomes.

---

## Test Case 14: Data Passport

### Steps
1. Open **Data Passport** (Trust & proof group), or arrive via **Attach to Passport** from a Decision Record.
2. Review the passport card (object, store/region/tier, trust score, classification labels).
3. Review classification bindings, residency, and the lineage timeline.

### Expected result
- The passport reflects the highest-priority live object; an empty state appears when there are no findings.
- Labels and fields are derived from real evidence.

### What it proves
Identity, classification, residency, and governance history are bound to an object and provable.

---

## Test Case 15: Evidence

### Steps
1. Open **Evidence** (Trust & proof group).
2. Review the KPI strip (Evidence records, Audit events, Export ready).
3. Review the audit-grade evidence table (object, actor, decision, status, receipt).
4. Click **Export evidence package**.

### Expected result
- KPIs and the table reflect live evidence/audit data; empty state when none.
- Export opens/downloads the agent's signed evidence package (connected mode).

### What it proves
DataDynamics records audit-grade proof of what happened, why, and who/what was involved.

---

## Test Case 16: Compliance

### Steps
1. Open **Compliance** (Insights group).
2. Review the KPI strip and **Regulatory coverage** bars.
3. Review recent attestations (if provided by the agent).

### Expected result
- Coverage bars mirror the agent's framework data; empty state when none.

### What it proves
Compliance posture is continuously derived from governed activity, not hand-maintained.

---

## Test Case 17: Integrations (AI providers)

### Steps
1. Open **Integrations** (Insights group).
2. Review the provider cards and posture chips (cloud-connected / private-network / air-gapped).
3. Review the active-provider config rows.
4. Click **Test provider runtime**.
5. Review the evidence-grounded recommendation panel.

### Expected result
- Provider cards come from the agent's provider catalog; empty state when none.
- The recommendation panel is grounded in a live finding and shows **auto-apply blocked**; an empty state appears when there is no finding. **Submit recommended action for approval** routes to Approvals.

### What it proves
AI is configurable per environment and acts only as a governed, evidence-grounded assistant — never an autonomous executor.

---

## Test Case 18: Pipelines (architecture)

### Steps
1. Open **Pipelines** (Execution group).
2. Toggle **Approved flow** vs **Blocked flow**.
3. Review the deployment posture cards.

### Expected result
- The flow diagram updates between the allowed path and the Guardian-blocked / HITL path.
- The banner explains each path.

### What it proves
The product has a clear, presentable Data Agent Cell story for both allowed and blocked paths.

---

## Test Case 19: Diagnostics

### Steps
1. Open **Diagnostics** (Insights group).
2. Review runtime health, contract checks, and notifications.

### Expected result
- Health cards reflect the agent runtime status (or "unknown" when offline).
- Contract checks and notifications reflect live data; empty states when none. No secrets are shown.

### What it proves
Support/runtime data is accessible without exposing secrets or disturbing the business path.

---

## Apply / write-back preconditions (connected Azure only)

Before write-back testing, confirm: a saved source pointing at a **non-production / approved** container; one or more test blobs; the customer approves write-back; and start with metadata/index-tag write-back. Keep Delete as a blocked-control test unless deletion is explicitly approved.

---

## Test Case 20: End-to-end preview -> apply (governed write-back)

### Steps
1. **Data Stores** -> save/select an approved test source.
2. **Blob Workbench** -> List blobs -> select a test object.
3. Keep **Action mode = Preview**, **Write-back = off**; click **Set tags**; confirm **PLAN** (no change); Dismiss.
4. Switch **Action mode = Apply**, enable **Write-back**.
5. Click **Set tags** again; on **ALLOW**, click **Apply approved action** (or, for a restricted action, **Submit for HITL approval**).
6. For the HITL path, open **Approvals** and **Approve & execute**.
7. Open **Evidence** and confirm a new record; reopen the object in **Blob Workbench** to verify the tags.

### Expected result
- Preview/Apply-without-write-back never modify the object.
- Apply + write-back + (Guardian allow / HITL approval) commits the change and writes a signed evidence record.
- Workbench reflects the updated tags when Azure accepts the write-back; a clear failure + evidence record if it does not.

### What it proves
Write-back requires Apply mode, write-back enabled, Guardian allow, and (for restricted actions) HITL approval — with end-to-end proof.

---

## Test Case 21: Apply blocked without gates

### Steps
1. **Blob Workbench** -> Apply + Write-back on.
2. Preview a **Delete** on a flagged object -> confirm **BLOCK** (no apply).
3. Preview a **Move (cross-region)** -> confirm **HITL REQUIRED**; submit; in **Approvals**, **Block** it; confirm it does not execute.

### Expected result
- Blocked actions never offer Apply.
- HITL actions never execute without an approval; a rejected request stays blocked.
- Evidence/audit records the blocked and rejected attempts.

### What it proves
Guardian allow and HITL approval are separate, both-required controls; destructive actions are stricter.

---

## Recommended client demo sequence (short)

```text
1.  Executive            -> live posture, coverage, residency
2.  Data Stores          -> add/select Azure Blob source (friendly fields)
3.  Blob Workbench       -> list blobs, inspect, show Preview vs Apply + write-back
4.  Scans & Jobs         -> queue async scan, show checkpoints
5.  Auto-Classification  -> findings + recommended actions
6.  Risk Explorer        -> open a Decision Record
7.  Approvals            -> preview restricted action, request + approve (HITL)
8.  Policy Governance    -> signed bundle + Guardian decisions
9.  Data Residency       -> local-execution + cross-border blocks
10. Evidence / Compliance-> audit-grade proof + export
11. Integrations         -> AI recommend-only, auto-apply blocked
12. Pipelines            -> close with approved vs blocked architecture flow
```

## If Azure is not connected during the demo

```text
This environment is not connected to a live Azure Blob account right now.
The same UI flow is ready for a connected tenant: save a source, scan, classify,
inspect, govern, and record evidence. For this session I will show the governed UI
path, the Preview/Apply and Guardian/HITL gates, AI recommend-only behavior, and the
evidence model — without changing a live customer source.
```

## Build & deploy reference

```text
Go 1.26.3 required.
go build -o bin\platform-api.exe  .\cmd\platform-api
go build -o bin\policy-agent.exe  .\cmd\policy-agent

Start (two terminals):
  bin\platform-api.exe        # :9090 control plane
  bin\policy-agent.exe        # :8080 agent + serves web\

Deploy the portal (single file): copy the bundled index.html to web\index.html
  -> loads at http://localhost:8080/

No Go changes are required: every endpoint the portal calls is already registered in
internal/api/server.go. Datasets and Federation have no dedicated route yet and will
show empty states unless that data is supplied.
```

## Coverage summary

| Product area | Screen (new) | Coverage |
|---|---|---|
| Posture | Executive | KPIs, regulatory coverage, governance pipeline, residency ring |
| Source setup | Data Stores | Friendly fields, saved sources, credential hiding |
| Inspection | Blob Workbench | List/inspect, Preview/Apply + write-back, Guardian/HITL gates |
| Risk drill-down | Risk Explorer | Findings table, filters, Decision Record slide-over |
| Execution | Scans & Jobs / Agent Runs | Async lifecycle, leases, batch detail |
| Classification | Auto-Classification | Entities, findings, recommended actions |
| HITL | Approvals | Queue, approve/block, rationale |
| Policy | Policy Governance | Signed bundle, deployment posture, decisions |
| Residency | Data Residency | Local-execution, zones, cross-border |
| Lineage | Data Passport | Object identity, classification, timeline |
| Proof | Evidence | Records, audit, export |
| Compliance | Compliance | Framework coverage, attestations |
| AI | Integrations | Provider catalog, recommend-only, grounding |
| Architecture | Pipelines | Approved vs blocked flow |
| Support | Diagnostics | Runtime health, contract checks, notifications |

## Notes on this build

- All screens read live `/v1` data from the agent; there is **no mock data**. Empty states and an "Agent not connected" banner appear when the agent is unreachable.
- The Executive dashboard numbers come from `/v1/analytics/enterprise-posture`, computed by the agent from its portal state (saved sources, discovered containers, listed blobs, evidence, audit, actions, jobs).
- **Regulatory coverage, Data Residency, Data Passport lineage, Datasets, and Federation are derived in the portal from live data** (evidence, guardian decisions, saved sources, runtime) when the agent does not yet expose a dedicated rollup. The moment the backend returns server-side fields (`framework_coverage`, `sovereignty`, `residency_zones`, `datasets`, `federation`), the portal uses those instead automatically — no UI change needed.
- Saved sources and discovered containers are read from `/v1/portal/options`; container discovery uses `GET /v1/connectors/azureblob/containers`.
- Governed-work controls (Action mode, Write-back) live on the Blob Workbench mode bar.
- Identity headers (`X-Everest-Actor` / `-Role` / `-Tenant`) are sent on every call and are set under the avatar menu.
- No Go changes are required: every endpoint the portal calls is already registered in `internal/api/server.go`.

### How each derived screen is computed (until backend rollups exist)
- **Regulatory coverage:** per-framework governed-vs-total from evidence entity types (PII→GDPR/CCPA/DPDP, PHI→HIPAA, PCI/secrets→PCI-DSS).
- **Data Residency / sovereignty:** zero-copy architecture ⇒ local execution; cross-border counts from guardian decisions; zones from saved-source regions.
- **Data Passport lineage:** evidence + audit records matched to the object, ordered into a timeline.
- **Datasets:** governed data products derived from saved sources + their classification findings.
- **Federation:** the connected cell from runtime/trust (single-cell honest default; multi-cell when backend supplies it).
