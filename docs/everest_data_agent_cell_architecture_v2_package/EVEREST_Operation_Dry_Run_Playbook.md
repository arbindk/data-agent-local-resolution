# EVEREST — Operation Dry-Run Playbook

## Purpose

Use this file to dry-run any EVEREST operation.

Every operation follows this structure:

```text
Why customer does it
→ Input
→ Orchestrator behavior
→ Connector behavior
→ Worker behavior
→ Guardian behavior
→ Result storage
→ Failure handling
```

---

# 1. Discover / Inventory

## Why

Customer wants to know what sources and objects exist.

## Flow

```text
Client → Orchestrator → Connector Adapter → Zero-Copy Accessor → Metadata Cache → Response Composer → Provenance
```

## Result

```text
inventory summary
object list
source capabilities
metadata coverage
```

---

# 2. Quick Scan / Metadata Scan

## Why

Customer wants fast visibility without deep content inspection.

## Flow

```text
Orchestrator → Connector → metadata-only read → local result cache → search index → evidence
```

## Result

```text
object count
metadata summary
risk hints
source health
dashboard data
```

---

# 3. Classify / Tag / Label

## Why

Customer wants to understand sensitivity and apply safe labels/tags.

## Flow

```text
Orchestrator → Connector → metadata/features → DSPM/classification → Guardian → Action Executor → Provenance
```

## Result

```text
classification
risk score
recommended tag
approved tag result
action receipt
```

---

# 4. Profile / Analyze

## Why

Customer wants posture, trend, and readiness insight.

## Flow

```text
metadata/cache → analysis workers → telemetry → result rollup → dashboard
```

## Result

```text
size distribution
risk distribution
owner coverage
stale data
over-shared data
migration readiness
```

---

# 5. Access / Read Metadata

## Why

Customer wants safe inspection without download.

## Flow

```text
Orchestrator → Connector handle → Zero-Copy Accessor → metadata response → provenance
```

## Result

```text
metadata details
permissions
tags
timestamps
audited access event
```

---

# 6. Monitor / Audit

## Why

Customer wants proof.

## Flow

```text
Telemetry Collector + Provenance Writer → Evidence Store → Response Composer
```

## Result

```text
who/what/when/why
decision
outcome
policy version
environment
integrity proof
```

---

# 7. Migration Actions

## Why

Customer wants safe movement or replication.

## Flow

```text
Orchestrator → source connector → analysis → Guardian → target connector → verification → provenance
```

## Result

```text
migration plan
approved action
copy/move receipt
verification result
rollback/cutover evidence
```

---

# 8. Failure Handling

| Failure | Behavior |
|---|---|
| Guardian unavailable | Mutation fails closed, analysis may continue |
| Connector unavailable | Retry/backoff, affected scope failed, evidence recorded |
| Policy unavailable | Use LKG if trusted, otherwise block |
| Local state failure | Stop execution, record degraded failure |
| Telemetry unavailable | Continue if safe, retry publish |
| Elasticsearch unavailable | Keep evidence/result locally, retry publish |
| Worker failure | Retry batch from checkpoint |
| Write conflict | ETag/version/idempotency prevents unsafe duplicate |

---

# 9. Dry-Run Questions

For every scenario ask:

```text
Who initiated it?
What source is involved?
Which connector is used?
What is the operation?
Is it read-only or mutating?
What policy applies?
Is Guardian needed?
What gets stored locally?
What gets stored permanently?
What evidence proves the outcome?
What happens if each component fails?
```
