# Everest Azure Blob — QA Scope and Scenario Checklist

## Purpose

This checklist defines what QA should validate once Azure Blob implementation is complete enough for team testing.

---

## 1. Source Inventory

### Scenario 1 — No saved config

Expected:

- Friendly setup message
- No console error
- User guided to enter source details

### Scenario 2 — Load saved config

Expected:

- Storage account loads
- Container loads
- Prefix loads if present
- Data store ID loads
- Credential shown as configured/not configured
- Secret not visible

### Scenario 3 — Test connection success

Expected:

- Connection OK
- Source status Ready or almost ready

### Scenario 4 — Test connection failure

Expected:

- Friendly failure reason
- No raw stack trace
- No page crash

### Scenario 5 — Discover containers

Expected:

- Container list shown
- User can continue to onboarding

---

## 2. Onboarding

### Scenario 1 — Valid onboarding

Expected:

- Source created/updated
- Job created
- Lease created
- Data Agent assigned
- Next step opens Scans & Jobs

### Scenario 2 — Missing required field

Expected:

- Inline validation message
- API not called until fixed

### Scenario 3 — Backup

Expected:

- Backup success message
- Backup ID/path if available

### Scenario 4 — Reset Jobs & Evidence

Expected:

- Jobs cleared
- Leases cleared
- Evidence cleared
- Source config retained
- User guided to validate source

---

## 3. Scans & Jobs

### Scenario 1 — Run without lease

Expected:

- Friendly no job/lease message

### Scenario 2 — Run with valid lease

Expected:

- Execution ID returned
- Job completed
- Evidence published

### Scenario 3 — Azure source unavailable

Expected:

- Execution fails gracefully
- Evidence marks failure
- UI shows source failure

### Scenario 4 — Dry-run

Expected:

- No write-back
- Actions planned only
- Evidence generated

---

## 4. Policy Artifact Trust / Item 13

### Scenario 1 — Normal policy verification

Expected:

- Policy version visible
- Git tag/commit visible
- Checksum verified
- Signature verified
- Active policy available
- LKG available

### Scenario 2 — Git unavailable

Expected:

- LKG fallback if valid
- No unsafe execution without trusted policy

### Scenario 3 — Hash mismatch

Expected:

- Execution blocked or fallback
- Clear reason shown

### Scenario 4 — Signature failure

Expected:

- Execution blocked or fallback
- Clear reason shown

---

## 5. Evidence Center

### Scenario 1 — No execution

Expected:

- Friendly empty state

### Scenario 2 — After execution

Expected:

- Manifest summary
- Manifest detail rows
- Action results
- Decision provenance
- Audit timeline

### Scenario 3 — High-risk object

Expected:

- Object visible
- Risk reason visible
- Policy/rules visible
- Guardian decision visible

---

## 6. Guardian Agent

### Scenario 1 — Dry-run mode

Expected:

- Actions planned
- No mutation

### Scenario 2 — Apply mode writeback disabled

Expected:

- No mutation

### Scenario 3 — Apply mode writeback enabled

Expected:

- Only safe metadata/index tag actions applied

### Scenario 4 — Destructive action

Expected:

- Blocked or dry-run only

### Scenario 5 — Guardian Agent unavailable

Expected:

- Fail closed for mutation
- Scan may continue in dry-run if policy allows

---

## 7. Policy Governance

### Scenario 1 — India to UAE migration

Expected:

- Decision block
- HITL required
- Contract owner / approver shown

### Scenario 2 — Safe tagging

Expected:

- Allow/review
- Guardian Agent still gates runtime action

### Scenario 3 — Policy precedence conflict

Expected:

- Higher precedence policy wins
- Explanation visible

---

## 8. Agent Working Store

### Scenario 1 — No scan

Expected:

- Empty state

### Scenario 2 — After scan

Expected:

- Normalized object rows
- Manifest rows
- Action rows
- Latest execution ID

### Scenario 3 — Large number of objects

Expected:

- Pagination/limit
- No browser freeze
- No full in-memory UI load

---

## 9. Reset / Re-run

Scenario:

- Run job
- Review evidence
- Backup
- Reset
- Onboard again
- Run again

Expected:

- New execution ID
- Old evidence not mixed
- Latest run clear

---

## 10. Build Acceptance

Before sending to QA:

- No blocking console errors
- No raw JSON on customer pages
- No duplicate actions
- No unavailable API calls from visible pages
- Known limitations documented
- Dry-run warning visible
- Apply mode warning visible
