# Item 13 — Contract Pack
## Agent Core Runtime Contracts

**Owner:** Arbind  
**Purpose:** Define the minimum viable contracts required for Agent Core to integrate with Item 9, Connectors, Item 6, Item 14, Item 10, Guardian, Item 8, Item 12, and Item 15.

---

## 1. Contract Principles

1. Contracts first, integration second.
2. Agent Core is the central orchestrator.
3. Components should be instruction-driven by Agent Core.
4. Connector is the normalization and write-back boundary.
5. Guardian approval is mandatory before write-back.
6. Processing Tier controls depth, cost, and persistence.
7. All major decisions must generate provenance.
8. Contracts must be lightweight enough for tomorrow's demo but extensible for product.

---

## 2. ExecutionState Contract

**Direction:** Item 9 → Item 13  
**Purpose:** Agent Core receives a complete prepared execution state before processing.

```json
{
  "execution_id": "exec-001",
  "lease_id": "lease-001",
  "batch_id": "batch-001",
  "data_store_id": "prod-blob-finance",
  "tenant_id": "tenant-001",
  "connector": {
    "connector_id": "blob-connector-001",
    "connector_type": "azure_blob",
    "connector_version": "0.1.0"
  },
  "policy_ref": {
    "compiled_policy_version": "cpv-2026.05.29.001",
    "compiled_git_tag": "compiled/cpv-2026.05.29.001",
    "compiled_git_commit": "",
    "compiled_artifact_hash": "sha256:abc"
  },
  "workflow_ref": {
    "workflow_id": "mvs-linear-workflow",
    "workflow_version": "v1",
    "workflow_git_tag": "workflow/mvs-linear-v1"
  },
  "processing_tier": "standard_signal",
  "runtime_limits": {
    "max_batch_records": 10000,
    "max_content_files": 500,
    "max_ephemeral_disk_mb": 2048,
    "max_memory_mb": 1024
  },
  "indexing_mode": "signals_only",
  "allow_last_known_good": true
}
```

Validation rules:

- `execution_id`, `lease_id`, `batch_id`, `data_store_id` are required.
- `compiled_policy_version` and `compiled_artifact_hash` are required.
- one of `compiled_git_tag` or `compiled_git_commit` is required.
- `workflow_id` and `workflow_version` are required.
- `processing_tier` must be one of `rolled_up_dashboard`, `standard_signal`, `full_investigation`.

---

## 3. PolicyContext Contract

**Direction:** Item 13 → downstream skills  
**Purpose:** Provide verified active policy context.

```json
{
  "data_store_id": "prod-blob-finance",
  "compiled_policy_version": "cpv-2026.05.29.001",
  "source_policy_id": "finance-data-protection",
  "source_policy_version": "v2.4",
  "artifact_hash": "sha256:abc",
  "signature_verified": true,
  "artifact_hash_verified": true,
  "entities_of_interest": {
    "entity_types": ["PII.Email", "Financial.BankAccount"],
    "keyword_groups": [
      { "group": "finance_terms", "keywords": ["invoice", "salary"] }
    ],
    "regex_groups": [
      { "group": "national_id", "patterns": ["[0-9]{4}-[0-9]{4}-[0-9]{4}"] }
    ]
  },
  "sovereignty_rules": [
    {
      "rule_id": "rule-sovereignty-01",
      "data_movement": "local_only",
      "allow_cross_boundary_transfer": false
    }
  ],
  "activation_source": "local_git",
  "fallback_used": false,
  "local_execution_confirmed": true
}
```

---

## 4. WorkflowContext Contract

**Direction:** Compiled Git → Item 13  
**Purpose:** Define the simple preconfigured MVS workflow.

```json
{
  "workflow_id": "mvs-linear-workflow",
  "workflow_version": "v1",
  "steps": [
    "resolve_policy",
    "resolve_workflow",
    "quick_scan",
    "metadata_scoring_sampling",
    "content_intelligence",
    "entity_aggregation",
    "guardian_evaluation",
    "connector_write_back",
    "batch_summary",
    "cleanup"
  ],
  "defaults": {
    "guardian_required_before_writeback": true,
    "one_batch_at_a_time": true,
    "partial_batch_success_allowed": true
  }
}
```

---

## 5. NormalizedBatch Contract

**Direction:** Connector / Quick Scan → Item 13  
**Purpose:** Provide normalized first-level metadata for batch processing.

```json
{
  "batch_id": "batch-001",
  "data_store_id": "prod-blob-finance",
  "source_system": "azure_blob",
  "records": [
    {
      "file_id": "file-001",
      "path": "finance/report1.pdf",
      "name": "report1.pdf",
      "size_bytes": 1048576,
      "last_modified": "2026-05-28T10:00:00Z",
      "last_accessed": "2026-05-20T10:00:00Z",
      "last_accessed_source": "native",
      "content_type": "application/pdf",
      "extension": "pdf",
      "is_folder": false,
      "permission_class": "broad_internal",
      "permission_risk_score": 72,
      "has_broad_access": true,
      "source_system": "azure_blob"
    }
  ]
}
```

---

## 6. SamplingRequest / SamplingResponse Contract

**Direction:** Item 13 ↔ Item 6

Request:

```json
{
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "processing_tier": "standard_signal",
  "entities_of_interest": ["PII.Email", "Financial.BankAccount"],
  "duckdb_table": "normalized_file_record",
  "instructions": {
    "score_full_batch": true,
    "sampling_aggressiveness": "balanced",
    "max_content_files": 500
  }
}
```

Response:

```json
{
  "batch_id": "batch-001",
  "scoring_completed": true,
  "sampled_file_ids": ["file-001", "file-008"],
  "risk_summary": {
    "critical": 2,
    "high": 10,
    "medium": 80,
    "low": 500
  },
  "provenance": {
    "decision_type": "sampling",
    "reasoning": "Selected high-risk files based on permission risk and financial indicators"
  }
}
```

---

## 7. ContentIntelligenceRequest / Response Contract

**Direction:** Item 13 ↔ Item 14

Request:

```json
{
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "processing_tier": "standard_signal",
  "sampled_file_ids": ["file-001"],
  "entities_of_interest": {
    "entity_types": ["PII.Email", "Financial.BankAccount"],
    "keyword_groups": [],
    "regex_groups": []
  },
  "input_location": {
    "duckdb_path": "data/work/exec-001/batch.duckdb",
    "ephemeral_disk_path": "data/work/exec-001/files"
  },
  "output_table": "file_entities"
}
```

Response:

```json
{
  "batch_id": "batch-001",
  "status": "completed",
  "files_processed": 1,
  "files_failed": 0,
  "signals_written_to_duckdb": true,
  "summary": {
    "entities_detected": 4,
    "high_confidence_entities": 3
  }
}
```

---

## 8. EntityAggregationResponse Contract

**Direction:** Item 10 → Item 13

```json
{
  "batch_id": "batch-001",
  "policy_version": "cpv-2026.05.29.001",
  "entities": {
    "PII.Email": {
      "total_count": 18,
      "files_with": 7,
      "distinct_count": 14
    },
    "Financial.BankAccount": {
      "total_count": 3,
      "files_with": 2,
      "distinct_count": 3
    }
  }
}
```

---

## 9. GuardianDecision Contract

**Direction:** Item 13 ↔ Guardian

Request:

```json
{
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "policy_context": {
    "compiled_policy_version": "cpv-2026.05.29.001",
    "source_policy_version": "v2.4"
  },
  "signals": [
    {
      "file_id": "file-001",
      "risk_score": 91,
      "entities": ["PII.Email", "Financial.BankAccount"],
      "recommended_actions": ["apply_dd_protection_flags"]
    }
  ]
}
```

Response:

```json
{
  "decision_id": "dec-001",
  "batch_id": "batch-001",
  "approved_actions": [
    {
      "file_id": "file-001",
      "action": "apply_dd_protection_flags",
      "value": 3
    }
  ],
  "blocked_actions": [],
  "reasoning": "High confidence financial and PII signals under active policy",
  "human_review_required": false,
  "local_execution_confirmed": true,
  "timestamp": "2026-05-30T10:00:00Z"
}
```

---

## 10. ConnectorAction Contract

**Direction:** Item 13 → Connector, only after Guardian approval

```json
{
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "guardian_decision_id": "dec-001",
  "actions": [
    {
      "file_id": "file-001",
      "path": "finance/report1.pdf",
      "action": "apply_dd_protection_flags",
      "parameters": {
        "dd_protection_flags": 3
      },
      "idempotency_key": "exec-001-file-001-ddflags"
    }
  ]
}
```

Response:

```json
{
  "batch_id": "batch-001",
  "status": "completed",
  "results": [
    {
      "file_id": "file-001",
      "action": "apply_dd_protection_flags",
      "status": "success"
    }
  ]
}
```

---

## 11. BatchSummary Contract

**Direction:** Item 13 → Item 15

```json
{
  "batch_summary_id": "bsum-001",
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "lease_id": "lease-001",
  "data_store_id": "prod-blob-finance",
  "processed_at": "2026-05-30T10:00:00Z",
  "processing_tier": "standard_signal",
  "file_count": 1000,
  "policy_version": "cpv-2026.05.29.001",
  "workflow_version": "v1",
  "entities": {
    "PII.Email": { "total_count": 18, "files_with": 7 }
  },
  "risk_summary": {
    "critical": 2,
    "high": 10,
    "medium": 80,
    "low": 908
  },
  "sovereignty": {
    "local_execution_confirmed": true,
    "violations": 0
  },
  "actions_taken": {
    "tagged": 1,
    "flagged_for_review": 0,
    "blocked": 0
  },
  "degradation": {
    "degraded": false,
    "reasons": []
  }
}
```

---

## 12. ProvenanceRecord Contract

**Direction:** Item 13 → Item 12 / audit storage

```json
{
  "decision_id": "dec-001",
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "decision_type": "guardian_action_approval",
  "source_policy_version": "v2.4",
  "compiled_policy_version": "cpv-2026.05.29.001",
  "rules_applied": ["rule-pii-01", "rule-finance-01"],
  "reasoning": "High confidence PII and financial entities detected",
  "local_execution_confirmed": true,
  "hitl_status": "not_required",
  "timestamp": "2026-05-30T10:00:00Z",
  "actor": "agent-core"
}
```

---

## 13. Error / Degradation Contract

```json
{
  "execution_id": "exec-001",
  "batch_id": "batch-001",
  "status": "degraded",
  "component": "content_intelligence",
  "reason": "content_intelligence_unavailable",
  "fallback_behavior": "metadata_only_processing",
  "partial_batch_success": true,
  "files_impacted": 125,
  "provenance_recorded": true
}
```
