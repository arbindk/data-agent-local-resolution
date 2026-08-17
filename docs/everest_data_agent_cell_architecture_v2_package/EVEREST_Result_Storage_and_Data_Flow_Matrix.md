# EVEREST — Result Storage and Data Flow Matrix

## Purpose

This file answers the specific question:

```text
Where are scan data, metadata, manifest, decisions, action results, and telemetry stored?
```

---

# 1. Storage Categories

| Category | Storage | Permanent or transient | Example data |
|---|---|---|---|
| Customer data | Customer source system | Permanent customer-owned | files, blobs, DB rows, API payloads |
| Runtime context | Local State | Transient/local durable | request context, active execution state |
| Metadata cache | Metadata & Cache | Transient/local cache | resource cache, content index cache, result cache |
| Checkpoints | Local State | Local durable until cleanup | batch markers, retry markers |
| Scan results | Result Cache | Transient until published | normalized metadata rows |
| Classification results | Result Cache / Search Index | Transient then published | classification, risk, matched rules |
| Manifest summary | Evidence Store | Permanent | execution summary |
| Manifest details | Evidence Store / Search Index | Permanent | per-object/per-batch evidence |
| Guardian decisions | Provenance Store | Permanent | allow/deny/obligations/reasoning |
| Action results | Provenance Store + Action Receipt Store | Permanent | completed/skipped/blocked/failed |
| Telemetry | Telemetry Store | Permanent or retained by TTL | metrics/logs/traces/health |
| Searchable output | Elasticsearch/OpenSearch | Permanent/rebuildable index | metadata/results/dashboard query data |
| Final response | Client response | Transient | returned results/metadata |

---

# 2. Scan Data Storage

## Quick scan / metadata scan

| Stage | Store |
|---|---|
| Source metadata read | Runtime memory |
| Normalized object | Local State / Result Cache |
| Batch checkpoint | Local State |
| Summary rollup | Result Cache |
| Search document | Elasticsearch/OpenSearch |
| Evidence | Evidence Store |
| Audit/provenance | Provenance Store |

## Deep scan

| Stage | Store |
|---|---|
| Raw content | Customer source by default |
| Stream/range buffer | Runtime memory only |
| Extracted features | Local State / Result Cache |
| Entity/classification output | Result Cache / Search Index |
| Redacted snippets, if allowed | Evidence Store |
| Full content copy | Not stored by default |

---

# 3. Manifest Storage

## Manifest Summary

Contains:

```text
execution_id
request_id
source_id
connector_id
operation
object_count
batch_count
success_count
failure_count
decision_count
action_count
duration
policy_version
policy_hash
```

Stored in:

```text
Evidence Store
Search Index
Control Plane summary record if platform has central UI
```

## Manifest Details

Contains:

```text
object reference
metadata hash
classification result
decision reference
action receipt reference
error status
batch id
timestamp
```

Stored in:

```text
Evidence Store
Elasticsearch/OpenSearch reference/index
```

---

# 4. Provenance Storage

Decision provenance records:

```text
request
actor
policy version
policy hash
rules applied
decision
obligations
action
outcome
timestamp
environment
integrity hash/signature
```

Stored in:

```text
Provenance Store
Evidence Store
Search Index for lookup
```

---

# 5. Telemetry Storage

Telemetry includes:

```text
metrics
logs
traces
health signals
worker counts
batch duration
connector latency
Guardian latency
AI/model latency
Elasticsearch publish latency
memory/cpu usage
```

Stored in:

```text
Telemetry Store / Observability backend
Local State during execution
```

---

# 6. What Must Not Be Stored by Default

```text
full raw customer files
full database rows
unredacted sensitive content
raw secrets
credentials
private keys
large unbounded content samples
```

---

# 7. Practical Rule

```text
Customer data stays in source.
EVEREST stores metadata, signals, decisions, results, evidence, and telemetry.
Full content moves only when explicitly allowed, approved, audited, and linked to provenance.
```
