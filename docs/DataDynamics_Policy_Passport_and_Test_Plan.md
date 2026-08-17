# DataDynamics — Policy Signing, Data Passport, and Full Scenario Test Plan

**Scope:** How compiled policies are created & signed, how the data passport for sensitive data is created/managed/used, and an exhaustive test plan that seeds real Azure Blob data with the built-in utility and exercises every scenario through the portal.

**Services**

```text
policy-agent.exe   -> http://localhost:8080   (data agent + serves the portal at /)
platform-api.exe   -> http://localhost:9090   (control plane: dashboard, approvals, governance)
azureblob-test-tool.exe  (cmd/azureblob-test-tool) -> seeds & inspects real blobs
policy-tool.exe          (cmd/policy-tool)          -> keygen / sign / verify policies
```

---

# Part A — How policies are created and signed

DataDynamics never trusts an unsigned policy. The chain is **author → compile → sign → publish → resolve+verify → enforce**, and the verified bundle hash travels with every action into evidence.

## A.1 The signing keys (Ed25519)

`policy-tool keygen` generates an Ed25519 keypair.

```bash
bin\policy-tool.exe keygen --out demo/keys
# writes demo/keys/ed25519-public.b64  and  demo/keys/ed25519-private.b64
```

- The **private** key signs compiled policies (kept by the policy authority / conductor).
- The **public** key is shipped to every agent and used to verify signatures. The agent reads it from `POLICY_PUBLIC_KEY_PATH` (default `demo/keys/ed25519-public.b64`).

## A.2 The compiled policy artifact

A compiled policy is a JSON file at:

```text
demo/compiled-policy-repo/policies/<version>/compiled-policy.json
demo/compiled-policy-repo/policies/<version>/compiled-policy.sig   (detached signature)
```

The artifact contains the rules the agent actually enforces:

- **metadata** — `compiled_policy_version`, `source_policy_id`, `source_policy_version`, `compiled_git_tag`, `hitl_approval_id`.
- **compiled_policy_logic** — `classification_rules` (entity → sensitivity label + risk score + whether human review is required), `remediation_rules` (risk trigger → allowed actions + whether approval is required), `sovereignty_rules` (`data_movement: local_only`, `allow_cross_boundary_transfer: false`).
- **entities_of_interest** — entity types, keyword groups, regex groups the scanner/DSPM looks for.
- **signature** — `algorithm: Ed25519`, `key_id`, `signature_file`.

## A.3 Signing and verifying

```bash
# sign the compiled artifact
bin\policy-tool.exe sign --artifact demo/compiled-policy-repo/policies/<version>/compiled-policy.json \
  --private demo/keys/ed25519-private.b64 \
  --out    demo/compiled-policy-repo/policies/<version>/compiled-policy.sig

# verify (what the agent does internally)
bin\policy-tool.exe verify --artifact .../compiled-policy.json \
  --public demo/keys/ed25519-public.b64 --sig .../compiled-policy.sig
# -> "signature verified"
```

## A.4 How the agent resolves and trusts a policy at runtime

On every governed run the agent's policy resolver:

1. Locates the artifact for the active version (`ACTIVE_COMPILED_POLICY_VERSION`, default `cpv-2026.05.29.001`) in the compiled-policy repo.
2. **Verifies the SHA-256 artifact hash** against `ACTIVE_COMPILED_POLICY_HASH`.
3. **Verifies the Ed25519 signature** with the public key. If either check fails it security-rejects or falls back to last-known-good (never silently runs an untrusted policy).
4. Parses the compiled logic and attaches `compiled_policy_version` + `policy_bundle_hash` to the Orchestrator's `ActionRequestEnvelope`, which flows to the Guardian decision and into evidence.

The portal surfaces this on **Policy Governance** (bundle version, signature status, hash, distribution) by calling `GET /v1/policy/artifact/trust`, which runs the live signature verification and returns `signature_verified` / `signature_status`.

## A.5 Config that controls policy trust

```text
COMPILED_REPO_PATH              demo/compiled-policy-repo
POLICY_PUBLIC_KEY_PATH          demo/keys/ed25519-public.b64
ACTIVE_COMPILED_POLICY_VERSION  cpv-2026.05.29.001
ACTIVE_COMPILED_POLICY_TAG      compiled/cpv-2026.05.29.001
ACTIVE_COMPILED_POLICY_HASH     sha256:f5050ea9...d8e73b
```

---

# Part B — Data passport for sensitive data

## B.1 What the passport is (and where it lives)

There is **no separate "passport" store** in the backend — and that is by design. A data passport is an **assembled, provable view** of one object, built from records the agent already writes during governed work:

- **Evidence records** (`/v1/evidence`) — classification result, severity, decision, policy version + `policy_bundle_hash`, reasoning, rules applied, final action.
- **Audit events** (`/v1/audit/events`) — who/what acted, when, target, operation, dry-run flag.
- **Guardian decisions** (`/v1/guardian/decisions`) — allow / block / HITL with rationale.
- **Manifest / provenance** (per execution) — normalized object state + compiled policy version.

The portal's **Data Passport** screen and the **Decision Record** slide-over assemble these into one object-scoped story: identity (store/region/tier), classification labels (detected entities), residency (local execution, cross-border posture), trust score (from risk), and a **lineage timeline** (every evidence + audit event for that object, in order).

> Engineering note: because the passport is assembled from existing signed records, it is inherently tamper-evident — each stamp carries the policy bundle hash and is backed by an evidence/audit row. If you later want a first-class `GET /v1/passport?object=…` endpoint, it should return exactly this assembly; the portal already prefers a backend field when present.

## B.2 How the passport is created (lifecycle)

```text
1. Discover  -> object appears (scan / list)            => evidence: discovered
2. Classify  -> DSPM finds entities (PII/PHI/PCI/secret) => evidence: classified + severity
3. Decide    -> Guardian allow / HITL / block           => guardian decision + policy hash
4. Act       -> tag / tier / quarantine / move / delete  => audit event + action receipt
5. Prove     -> all of the above bound to the object     => passport + evidence package
```

## B.3 How the passport is used

- **Investigate:** Risk Explorer → click an object → **Decision Record** shows policy lineage, Guardian decision, entities, action; **Attach to Passport** / **Open audit log** / **Copy receipt**.
- **Govern:** an approver reviewing an action opens the object's passport to see full history before approving.
- **Prove:** **Evidence → Export evidence package** (`/v1/evidence/package`) emits the signed bundle for audit.

---

# Part C — Seeding real test data with the built-in utility

`azureblob-test-tool` (`cmd/azureblob-test-tool`) talks to a real Azure Blob account (or Azurite emulator) and supports: `list-containers, list, view, create, update, download, set-metadata, set-tags, delete`.

Set access once:

```bash
set AZURE_STORAGE_CONNECTION_STRING=<your connection string or Azurite>
```

> Local option with no cloud: run **Azurite** and point the connection string at it. The same commands work.

## C.1 Seed sensitive content that triggers the DSPM detectors

The DSPM engine detects (built-in regex/keywords): **EMAIL, SSN, INDIA_PAN, PHONE, credit-card (PAN), and secrets (api_key/secret/token/password)**, plus finance keywords (salary, invoice, bank account). Seed files that hit each:

```bash
# PII - email + phone + SSN
bin\azureblob-test-tool.exe --op create --container finance --blob hr/employee_records.csv ^
  --content "name,email,phone,ssn\nA Rao,a.rao@acme.com,+1-415-555-0100,123-45-6789"

# India PAN (DPDP)
bin\azureblob-test-tool.exe --op create --container finance --blob tax/pan_list.txt ^
  --content "PAN: ABCDE1234F  salary statement attached"

# PCI - credit card / PAN
bin\azureblob-test-tool.exe --op create --container payments --blob batch/cards.csv ^
  --content "card,exp\n4111 1111 1111 1111,12/27"

# Secret exposure
bin\azureblob-test-tool.exe --op create --container devops --blob env/prod.env ^
  --content "API_KEY=EXAMPLE_SECRET_VALUE_NOT_A_REAL_KEY"

# Finance keyword + bank account
bin\azureblob-test-tool.exe --op create --container finance --blob ap/invoice_0912.txt ^
  --content "invoice total due, remit to bank account 1234-5678-9012"

# Clean / low-risk control object
bin\azureblob-test-tool.exe --op create --container marketing --blob copy/banner.txt ^
  --content "summer sale headline, no personal data"
```

## C.2 Seed objects for residency / tiering / lifecycle scenarios

```bash
# large cold object (tiering candidate)
bin\azureblob-test-tool.exe --op create --container backups --blob 2019/ledger.bak --file .\big.bin
bin\azureblob-test-tool.exe --op set-tags --container backups --blob 2019/ledger.bak --tags lifecycle=cold

# legal-hold object (delete should be blocked)
bin\azureblob-test-tool.exe --op create --container legal --blob hold/case_7741.txt --content "under legal hold"
bin\azureblob-test-tool.exe --op set-tags --container legal --blob hold/case_7741.txt --tags legal_hold=true
```

## C.3 Verify what you seeded

```bash
bin\azureblob-test-tool.exe --op list-containers
bin\azureblob-test-tool.exe --op list --container finance --prefix hr/
bin\azureblob-test-tool.exe --op view --container devops --blob env/prod.env
```

---

# Part D — Full scenario test matrix (all paths)

Run each through the portal at `http://localhost:8080/`. Legend: **[C]** needs connected Azure, **[A]** works with agent only, **[U]** UI-only.

## D.1 Connection & shell
- T1 [A] Open portal; sidebar dot **green/Agent connected**. Stop the agent → **amber/Agent offline** + banner; KPIs read 0. Restart → recovers.
- T2 [A] Avatar → set actor/role/tenant → Save & reconnect → identity reflected; headers sent on every call.

## D.2 Source + discovery
- T3 [C] Data Stores → enter account (no raw connection string) → **Test access** OK.
- T4 [C] **Save source** → appears in Saved sources from `/v1/portal/options`.
- T5 [C] **Discover containers** → lists the containers you seeded (finance, payments, devops, legal, backups, marketing).
- T6 [C] Tick containers → **Use selected containers** → populates scope → Save.
- T7 [A] Open the saved source in **Blob Workbench**.

## D.3 Scan & classify
- T8 [C] Scans & Jobs → **Queue async scan** (metadata) → job appears with checkpoints; **Pause/Resume/Cancel** each work.
- T9 [C] Queue **content scan (DSPM)** → findings populate Auto-Classification.
- T10 [A] Auto-Classification → confirm entity tiles + findings for each seeded type: **EMAIL, SSN, PAN, PCI, secret, finance keyword**; clean object is low/absent.
- T11 [A] Agent Runs → run row → **Batch run** detail shows pipeline steps, Guardian decisions, summary.

## D.4 Policy creation, signing, trust
- T12 [author] Create/edit a compiled policy JSON (new `<version>` folder).
- T13 [author] `policy-tool sign` it; set `ACTIVE_COMPILED_POLICY_VERSION` + `ACTIVE_COMPILED_POLICY_HASH`; restart agent.
- T14 [A] Policy Governance → bundle version + **signature: verified** + hash + distribution shown.
- T15 [tamper] Edit the signed JSON by one byte → restart → Policy Governance shows **signature failed / not verified**; governed actions are rejected/fall back. (Revert after.)
- T16 [A] Guardian decisions list shows allow/block/HITL entries with rationale and policy hash.

## D.5 Workbench governed actions — the Preview/Apply + Write-back gates
- T17 [A] Mode bar: **Preview** + write-back off → any action previews as **PLAN**, no Apply button.
- T18 [A] **Apply** + write-back **off** → still **PLAN** (no mutation).
- T19 [C] **Apply** + write-back **on**, safe action (**Set tags / Set metadata**) → **ALLOW** → **Apply approved action** → writes; re-open object → tags/metadata updated.
- T20 [C] **Change tier** on the cold backup object → governed tier change.
- T21 [C] **Move (cross-region)** → **HITL REQUIRED** → **Submit for HITL approval** → routes to Approvals (no change yet).
- T22 [C] **Delete** on the **legal-hold** object → **BLOCK** (policy), Dismiss only.
- T23 [C] **Quarantine** a secret-exposed object → restricted → HITL.

## D.6 Approvals / HITL (every branch)
- T24 [A] Approvals → pending item shows operation, object, source, policy, requester, rationale.
- T25 [A] **Approve & execute** → status approved; evidence recorded; posture counters move.
- T26 [A] **Block** a pending item → blocked; reason recorded.
- T27 [A] Dashboard → **Needs your decision** → Approve / Block / Passport inline; verify it matches Approvals.
- T28 [A] Guardian-allow vs HITL-approval are **separate**: an item approved in HITL but Guardian-pending must still not execute (and vice-versa).

## D.7 Passport & evidence (sensitive-data proof)
- T29 [A] Risk Explorer → click a seeded sensitive object → **Decision Record**: policy lineage, Guardian decision, entities, action.
- T30 [A] **Attach to Passport** → Data Passport shows identity, classification labels (the detected entities), residency, trust score, and a **lineage timeline** built from that object's evidence + audit.
- T31 [A] **Copy receipt** and **Open audit log** work.
- T32 [A] Evidence → records list (object, actor, decision, status, receipt) → **Export evidence package** downloads.

## D.8 AI (recommend-only, governed)
- T33 [A] Integrations → provider catalog from `/v1/ai/providers/catalog`; **Test provider runtime**.
- T34 [A] Evidence-grounded recommendation shows for a real finding; **auto-apply blocked** indicator present.
- T35 [A] **Submit recommended action for approval** → routes to Approvals (never executes directly).

## D.9 Residency / sovereignty / posture
- T36 [A] Data Residency → local-execution ring, cross-border blocks, zones (from source regions).
- T37 [A] Executive → KPIs, **Regulatory coverage** (per framework from findings), **Governance pipeline**, **Posture trend** graph (after scans across days), residency ring.
- T38 [A] Compliance → framework coverage + (when present) attestations.

## D.10 Sharing & extensibility
- T39 [A] Datasets → governed data products derived from sources + classification.
- T40 [A] Federation → connected cell(s) from runtime/trust.
- T41 [A] Pipelines → toggle **Approved flow** vs **Blocked flow**; deployment posture cards.
- T42 [A] Diagnostics → runtime health, contract checks, notifications (no secrets shown).

## D.11 Recovery / negative paths
- T43 [C] Apply with write-back off → blocked → enable write-back → request approval → approve → re-run succeeds. Evidence holds **both** the blocked attempt and the later success.
- T44 [C] Reject a HITL request → action stays blocked; request again → approve → proceeds.
- T45 [A] Disconnect agent mid-session → empty states + banner, no crash; reconnect → data returns.

---

# Part E — Quick API sanity (optional, no UI)

```bash
curl http://localhost:8080/v1/analytics/enterprise-posture
curl http://localhost:8080/v1/connectors/azureblob/containers
curl http://localhost:8080/v1/portal/options
curl http://localhost:8080/v1/policy/artifact/trust
curl http://localhost:8080/v1/evidence?limit=20
curl http://localhost:8080/v1/guardian/decisions?limit=20
```

If these return JSON, the portal populates; if they fail, the portal shows honest empty states.

---

# Coverage map (scenario → screen → backend)

| Scenario | Screen | Backend |
|---|---|---|
| Connect / discover / select containers | Data Stores | `/v1/connectors/azureblob/config`, `/containers`, `/v1/portal/options` |
| Scan lifecycle | Scans & Jobs / Agent Runs | `/v1/scan/azureblob/jobs(/...)` |
| Classification | Auto-Classification | `/v1/dspm/*`, `/v1/evidence` |
| Policy trust | Policy Governance | `/v1/policy/artifact/trust` |
| Governed actions + gates | Blob Workbench | `/v1/tools/azureblob/workbench`, `/v1/connectors/azureblob/actions/execute`, `/writeback` |
| HITL | Approvals + Dashboard | `/v1/actions/pending`, `/v1/actions/history` |
| Guardian | Policy Governance | `/v1/guardian/decisions` |
| Passport / lineage | Data Passport / Risk Explorer | assembled from `/v1/evidence` + `/v1/audit/events` |
| Evidence export | Evidence | `/v1/evidence`, `/v1/evidence/package` |
| AI recommend-only | Integrations | `/v1/ai/providers/catalog`, `/v1/ai/azureblob/recommendations` |
| Posture + trend graph | Executive | `/v1/analytics/enterprise-posture` (incl. `trends`) |
