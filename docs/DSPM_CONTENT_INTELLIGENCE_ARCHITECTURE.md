# DSPM Content Intelligence Architecture

## Goal

Everest should support data security posture management across metadata, sampled content, model-assisted content intelligence, and deep scans. Azure Blob is the first implementation surface, but the DSPM module is connector-agnostic so future connectors can reuse the same request and result contract.

## Scan Tiers

| Tier | Name | Purpose | Typical Runtime |
|---|---|---|---|
| 0 | `metadata` | Path, extension, size, content type, tags, metadata, ownership signals | Always local |
| 1 | `rules` | Sampled content scan with deterministic entity detectors | Local/offline |
| 2 | `model` | BERT/NER/domain model entity extraction and classification | Local ONNX, private model service, or cloud model |
| 3 | `deep` | Full-content or large-sample delegated analysis | Background job with policy/HITL gates |

Tiering matters because enterprise customers have different environments:

- Cloud-connected customers can use cloud language services.
- Private-network customers can use internal model gateways.
- Air-gapped customers can use local rules and local BERT/ONNX profiles.
- Regulated customers can require HITL before deep or destructive workflows.

## Current Backend

Added:

- `internal/dspm/engine.go`
- `GET /v1/dspm/catalog`
- `POST /v1/dspm/content-scan`

The first engine version supports:

- Metadata-sensitive path checks
- Built-in keyword pattern packs
- Customer-defined keyword patterns
- Keyword exclusions
- Email detection
- Phone detection
- Credit-card detection with Luhn validation
- US SSN-like detection
- India PAN-like detection
- Secret/API key/token/password detection
- Classification and risk scoring
- Recommended Azure Blob metadata and index tags
- Next-tier recommendation
- HITL recommendation
- Model profile catalog for local BERT, domain BERT, and cloud language services

## Model Profiles

Current catalog profiles:

- `builtin-rules`
- `local-bert-ner`
- `domain-bert-dspm`
- `cloud-language-service`

Only `builtin-rules` performs local detection today. The other profiles are product contracts for customer-managed model runtimes. They should be connected through the same plugin/runtime framework used for AI providers and custom connectors.

## Entity Mapping

Detected entities map into canonical DSPM entity types:

- `EMAIL`
- `PHONE`
- `CREDIT_CARD`
- `SSN`
- `INDIA_PAN`
- `SECRET`
- `KEYWORD`
- `BUSINESS_SENSITIVE`
- `LEGAL_TERM`
- `FINANCIAL_TERM`
- `PERSON`
- `ORG`
- `LOCATION`
- `CUSTOM_ENTITY`

Entity results are converted into:

- Risk score
- Classification: `public`, `internal`, `confidential`, `restricted`
- Recommended metadata
- Recommended index tags
- HITL requirement
- Next-tier scan recommendation

## Workbench UX

Azure Blob Workbench now includes:

- Scan tier selector
- Model profile selector
- Keyword pack selector
- Custom keyword pattern editor
- Keyword exclusions
- Optional check selector
- `Run DSPM Scan`
- `Use DSPM Tags`
- Entity/check result view
- Action canvas with generated metadata and index tags
- HITL handoff when risk is high

## Keyword Pattern Matching

Keyword matching is supported across:

- Blob path and file id
- Metadata keys and values
- Index tag keys and values
- Sampled content for `rules`, `model`, and `deep` tiers

Built-in keyword packs:

- `enterprise-sensitive`
- `legal-contracts`
- `finance-hr`
- `technology-secrets`
- `builtin-default`, which applies all built-in packs
- `none`, which applies only customer-provided custom patterns

Custom pattern format in the UI:

```text
pattern|severity|entity_type|match_mode
```

Examples:

```text
Project Everest|high|CUSTOM_ENTITY|phrase
acquisition target|critical|BUSINESS_SENSITIVE|phrase
customer-[0-9]{5}|medium|CUSTOM_ENTITY|regex
```

Supported match modes:

- `contains`
- `phrase`
- `word`
- `regex`

Supported severities:

- `low`
- `medium`
- `high`
- `critical`

Keyword exclusions allow customers to suppress known false-positive contexts such as templates, samples, or test fixtures.

## Next Implementation Steps

1. Persist DSPM scan results in the local working store.
2. Persist customer keyword dictionaries and exclusions.
3. Add background delegated deep scan jobs.
4. Add model runtime invocation for `local-bert-ner` and `domain-bert-dspm`.
5. Add entity mapping policy configuration per customer/region.
6. Add connector-agnostic DSPM scan hooks so SMB, SharePoint, S3, NFS, and custom connectors can call the same engine.
7. Add redaction and evidence retention controls for regulated environments.
8. Add dashboard widgets for DSPM posture: sensitive objects, restricted objects, secrets, model coverage, HITL queue.

## Testing Contract

Request:

```json
{
  "source_system": "azure_blob",
  "connector_id": "azureblob",
  "file_id": "azureblob:container:path/payroll.txt",
  "container": "container",
  "path": "path/payroll.txt",
  "content_type": "text/plain",
  "sample_text": "employee email jane@example.com token=abcdef1234567890",
  "scan_tier": "rules",
  "model_profile": "builtin-rules",
  "keyword_profile": "builtin-default",
  "keyword_patterns": [
    {
      "pattern": "Project Everest",
      "severity": "high",
      "entity_type": "CUSTOM_ENTITY",
      "match_mode": "phrase"
    }
  ]
}
```

Expected response:

```json
{
  "status": "completed",
  "scan_tier": "rules",
  "model_profile": "builtin-rules",
  "classification": "restricted",
  "risk_score": 80,
  "requires_hitl": true,
  "entities": [
    {
      "type": "EMAIL",
      "detector": "builtin_regex_email"
    },
    {
      "type": "SECRET",
      "detector": "builtin_regex_secret"
    }
  ],
  "recommended_tags": {
    "everest_dspm_classification": "restricted"
  }
}
```
