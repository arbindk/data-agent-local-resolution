# Everest Field Dictionary

Version: Draft 1  
Product: Everest Enterprise Data Operations - Azure Blob Edition

## Purpose

This dictionary explains important fields across UI, API payloads, runtime configuration, local data objects, and external Azure Blob mappings.

Column meaning:

- Field Name: Name as shown in UI or API.
- Layer: UI, API, DB/Data, Azure, Config, or Runtime.
- Module: Where the field is used.
- Description: What the field means.
- Why Needed: Why Everest needs it.
- Type: Expected type or format.
- Mandatory: Whether the field is required.
- Sample Value: Example value.
- Allowed Values: Valid values or range.
- Source: Where the value comes from.
- Target / Mapping: Where the value goes.
- Impact If Wrong: What breaks if incorrect.

## 1. Azure Blob Source Configuration Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| connector | UI/API | Source Inventory | Selected connector type. | Determines runtime path. | string | Yes | azureblob | azureblob, future connectors | User/catalog | Connector runtime | Wrong connector blocks correct source flow. |
| deployment_mode | UI/API | Source Inventory | Customer network model. | Guides cloud/private/air-gapped behavior. | string | Yes | private_network | cloud_connected, private_network, air_gapped, hybrid | User | Config/context | Wrong assumptions for AI/connectivity. |
| storage_account | UI | Source Inventory | Friendly Azure storage account name. | Helps identify source. | string | No | hulk01 | Any display string | User | UI/context | Low impact unless used in reports. |
| connection_string | UI/API/Secret | Source Inventory, Workbench | Azure Storage credential. | Required to access Azure Blob. | secret string | Yes initially | DefaultEndpointsProtocol=... | Valid Azure Storage connection string | User/env var | Secret store | Azure calls fail or hit wrong account. |
| credential_set | API/UI | Source Inventory | Whether credential exists. | Shows readiness without exposing secret. | boolean | Yes read-only | true | true, false | Secret store | UI status | User may think source is configured when not. |
| container | UI/API/Azure | Source, Workbench, Scan | Azure Blob container name. | Defines scan/action scope. | string | Yes for blob ops | customer-demo | Azure container naming rules | User/discovery | Azure SDK calls | Empty or wrong data scope. |
| prefix | UI/API/Azure | Source, Workbench, Scan | Blob prefix filter. | Narrows list/scan scope. | string | No | hr/payroll | Any blob prefix | User | Azure list pager | Hides or includes wrong objects. |
| data_store_id | UI/API/Data | Source, Execution, Policy | Everest logical source ID. | Links source, policies, evidence, actions. | string | Yes | azureblob-hulkb01 | Unique source ID | User/config | Execution/evidence/policy | Evidence and policy mapping breaks. |
| batch_id | API/Data | Scan, Store | ID for normalized scan batch. | Groups scan records. | string | Yes for scan | batch-azure-001 | Unique batch ID | Config/runtime | Local working store | Records not traceable. |
| action_mode | UI/API | Governance, Actions | Whether actions are dry-run or apply. | Prevents accidental writes. | string | Yes | dry_run | dry_run, apply | User/config | Action execution | Wrong value may apply or block changes. |
| writeback_enabled | UI/API | Governance, Actions | Whether write-back is allowed. | Controls mutation of Azure Blob. | string/bool | Yes | false | true, false | User/config | Action execution | Approved actions may not apply or may be too permissive. |
| source_region | UI/API | Governance | Source data region. | Used for residency and policy. | string | Recommended | IN | Region/country code | User | Governance context | Wrong policy outcome. |
| target_region | UI/API | Governance | Target region for movement. | Used in migration/residency checks. | string | For migration | AE | Region/country code | User | Governance context | Wrong migration decision. |

## 2. Azure Blob Workbench Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| operation | UI/API | Workbench | Workbench operation to run. | Selects backend behavior. | string | Yes | view_blob | list_containers, list_blobs, view_blob, create_blob, update_blob, set_metadata, set_tags, download_blob, delete_blob | User/button | `/v1/tools/azureblob/workbench` | Wrong operation may mutate or fail. |
| blob | UI/API/Azure | Workbench | Blob path/name. | Identifies target object. | string | Yes for object ops | everest-test/sample.txt | Any valid blob path | User/list | Azure Blob client | Wrong object changed or viewed. |
| limit | UI/API | Workbench | Max rows for list. | Controls response size. | integer | No | 100 | 1 to 5000 recommended | User/default | Azure pager | Too high can slow UI. |
| content | UI/API | Workbench | Inline text content for create/update. | Allows creating test blobs. | string | No | Everest test | Any text | User | Azure upload | Wrong content affects test. |
| content_base64 | API | Workbench | Browser file content encoded as base64. | Uploads binary files. | base64 string | No | JVBERi0x... | Base64 | Browser FileReader | Azure upload | Invalid base64 fails upload. |
| content_type | UI/API/Azure | Workbench | MIME type. | Helps classification and downstream handling. | string | No | text/plain | MIME type | User/file | Azure blob HTTP header | Missing type reduces metadata quality. |
| metadata | UI/API/Azure | Workbench, Actions | Blob metadata key-values. | Supports classification/audit/ownership. | object/string pairs | For set_metadata | {"owner":"qa"} | Azure metadata key rules | User/AI/DSPM | Azure SetMetadata | Replaces metadata set. |
| tags | UI/API/Azure | Workbench, Actions | Azure Blob index tags. | Supports search, policy, DSPM classification. | object/string pairs | For set_tags | {"classification":"restricted"} | Azure tag key/value rules | User/AI/DSPM | Azure SetTags | Replaces tag set. |
| max_bytes | UI/API | Workbench | Preview byte limit. | Avoids loading full large blobs. | integer | No | 4096 | >=0 | User/default | Preview reader | Too high may slow response. |
| overwrite | UI/API | Workbench | Allows create over existing blob. | Protects existing test data. | boolean | No | false | true, false | User checkbox | Upload behavior | If true, content may be overwritten. |
| dry_run | UI/API | Workbench, Actions | Preview only. | Safety control. | boolean | No, default true in UI | true | true, false | UI checkbox | Backend action logic | If false, mutation can occur. |
| confirm | UI/API | Workbench, Actions | Confirmation text. | Required for delete/restricted action. | string | For delete | DELETE | DELETE | User | Delete/action validation | Delete blocked if missing. |
| download_name | API/UI | Workbench | Suggested browser download file name. | Saves downloaded blob. | string | No | sample.txt | File name | Backend | Browser download | Wrong file name only affects local save. |

## 3. AI Recommendation And Action Canvas Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| prompt | UI/API | AI Workbench | Optional instruction. | Side channel for non-standard guidance. | string | No | Recommend safest action | Any text | User | AI recommendation API | Poor prompt may reduce recommendation quality. |
| candidate_action | UI | AI Workbench | User/AI selected action. | Maps recommendation to action API. | string | Yes | apply_blob_index_tags | Azure action IDs | User/AI | Action payload | Wrong action could route incorrectly. |
| automation_mode | UI/API | AI Workbench | AI automation level. | Controls safe apply vs HITL. | string | Yes | hitl_required | hitl_required, preview_only, auto_apply_safe | User/config | AI/action behavior | Too permissive can risk unsafe apply. |
| recommendation.operation | API/UI | AI Workbench | Recommended action. | Primary AI output. | string | Yes | apply_blob_metadata | Supported actions | AI/fallback | Action Center | Wrong action impacts governance. |
| recommendation.rationale | API/UI | AI Workbench | Explanation. | Needed for audit and review. | string | Yes | Sensitive object needs tags. | Any text | AI/DSPM | HITL/action canvas | Missing rationale weakens audit. |
| recommendation.confidence | API/UI | AI Workbench | Confidence score. | Helps user trust/review. | number/string | No | 0.78 | 0 to 1 or text | AI | UI | Misleading confidence affects decisions. |
| recommendation.hitl_required | API/UI | AI Workbench | Whether approval is needed. | Protects risky actions. | boolean | Yes | true | true, false | AI/DSPM/policy | UI/action flow | False negative can permit unsafe action. |
| action_canvas.metadata | UI | AI Workbench | Editable generated metadata. | Lets user inspect/modify AI output. | key-value text | No | owner=qa | key=value lines/CSV | AI/DSPM/user | Workbench metadata field | Wrong metadata applied if not reviewed. |
| action_canvas.tags | UI | AI Workbench | Editable generated tags. | Lets user inspect/modify tags. | key-value text | No | classification=restricted | key=value lines/CSV | AI/DSPM/user | Workbench tags field | Wrong tags affect policy/search. |
| action_canvas.rationale | UI | AI Workbench | Editable explanation. | Supports HITL and audit. | text | No | Restricted payroll object | Any text | AI/DSPM/user | Approval/action reason | Missing context slows approval. |

## 4. DSPM Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| source_system | API | DSPM | Source type. | Connector-agnostic scan context. | string | No | azure_blob | Any source ID | UI/connector | DSPM result | Missing context reduces traceability. |
| connector_id | API | DSPM | Connector identifier. | Links result to connector. | string | No | azureblob | Connector IDs | UI/connector | DSPM result | Harder to map result. |
| file_id | API/Data | DSPM | Normalized object ID. | Traceability. | string | Recommended | azureblob:container:path.txt | Source-specific format | Workbench/scan | DSPM/evidence | Result cannot be linked. |
| path | API | DSPM | Object path. | Path-based risk and keyword checks. | string | Recommended | hr/payroll.txt | Any path | Workbench | DSPM engine | Keyword/path checks missed. |
| sample_text | API | DSPM | Text sample for content scan. | Enables entity/rules matching. | string | For rules/model/deep | jane@example.com | Any text | Blob preview/user | DSPM detectors | No content entities detected. |
| scan_tier | UI/API | DSPM | Scan depth. | Controls cost/detail. | string | Yes | rules | metadata, rules, model, deep | User | DSPM engine | Too shallow misses risk; too deep may need runtime. |
| model_profile | UI/API | DSPM | Model profile. | Supports BERT/local/cloud model strategy. | string | Yes | builtin-rules | builtin-rules, local-bert-ner, domain-bert-dspm, cloud-language-service | User | DSPM/model readiness | Wrong runtime expectation. |
| enabled_checks | UI/API | DSPM | Optional check filter. | Allows targeted scans. | string array | No | content_secret | Supported check IDs | User | DSPM engine | Planned finer filtering may miss checks. |
| keyword_profile | UI/API | DSPM | Built-in keyword pack. | Business-sensitive keyword matching. | string | No | finance-hr | builtin-default, enterprise-sensitive, legal-contracts, finance-hr, technology-secrets, none | User | DSPM keyword engine | Wrong pack creates misses/noise. |
| keyword_patterns | UI/API | DSPM | Customer-defined patterns. | Captures customer-specific sensitive terms. | array | No | Project X|high|CUSTOM_ENTITY|phrase | Pattern objects | User | DSPM keyword engine | Custom risks missed. |
| keyword_exclusions | UI/API | DSPM | False-positive suppression. | Avoids matching sample/template contexts. | string array | No | sample,template | Any strings | User | DSPM keyword engine | Too broad exclusions hide real risk. |
| classification | API/UI | DSPM | Data class. | Drives governance. | string | Output | restricted | public, internal, confidential, restricted | DSPM | UI/tags/HITL | Wrong class affects policy. |
| risk_score | API/UI | DSPM | Risk score. | Prioritizes action. | integer | Output | 85 | 0 to 100 | DSPM | UI/HITL | Wrong risk affects review. |
| next_tier | API/UI | DSPM | Recommended next scan depth. | Guides deeper inspection. | string | Output | model | rules, model, deep, empty | DSPM | UI/user | Missing next tier may stop investigation. |
| requires_hitl | API/UI | DSPM | Approval recommendation. | Protects risky data. | boolean | Output | true | true, false | DSPM | HITL/action flow | Wrong value impacts safety. |
| recommended_metadata | API/UI | DSPM | Metadata to apply. | Writes DSPM context to object. | object | Output | everest_dspm_tier=rules | key-value | DSPM | Workbench metadata | Wrong metadata affects audit. |
| recommended_tags | API/UI | DSPM | Index tags to apply. | Writes classification/risk to Azure. | object | Output | everest_dspm_classification=restricted | key-value | DSPM | Workbench tags | Wrong tags affect policy/search. |

## 5. Keyword Pattern Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| pattern | UI/API | DSPM Keywords | Keyword or regex. | Detects customer-specific risk. | string | Yes | Project Everest | Any non-empty string | User/pack | Keyword engine | No match if wrong. |
| severity | UI/API | DSPM Keywords | Risk severity. | Affects risk score/HITL. | string | No | high | low, medium, high, critical | User/pack | Risk model | Too low hides risk, too high creates noise. |
| entity_type | UI/API | DSPM Keywords | Entity type to emit. | Maps keyword to business category. | string | No | CUSTOM_ENTITY | KEYWORD, CUSTOM_ENTITY, BUSINESS_SENSITIVE, LEGAL_TERM, FINANCIAL_TERM, SECRET | User/pack | Entity result | Poor mapping affects reporting. |
| match_mode | UI/API | DSPM Keywords | How pattern matches. | Controls precision. | string | No | phrase | contains, phrase, word, regex | User/pack | Keyword engine | Wrong mode causes misses/noise. |
| case_sensitive | API | DSPM Keywords | Whether case matters. | Supports exact terms. | boolean | No | false | true, false | API/custom pack | Keyword engine | Unexpected misses. |
| scopes | API | DSPM Keywords | Where to match. | Limits path/content/metadata/tag matching. | string array | No | path,content | path, metadata, tags, content | API/custom pack | Keyword engine | Wrong scope misses data. |
| description | API | DSPM Keywords | Explanation. | Helps user/auditor understand match. | string | No | M&A sensitive term | Any text | Pack/user | Entity rationale | Missing context. |

## 6. Azure Blob Action Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| execution_id | UI/API/Data | Actions/Evidence | Execution context. | Links action to evidence. | string | Recommended | exec-azureblob-20260613 | Execution ID | Scan/job | Action/evidence | Audit link missing. |
| file_id | UI/API | Actions | Normalized file ID. | Resolves container/path. | string | For normalized action | azureblob:container:path.txt | azureblob:{container}:{blob_path} | Scan/Workbench | Azure action resolver | Wrong object action. |
| blob_path | UI/API | Actions | Direct path fallback. | Used when file_id absent. | string | Required if file_id absent | path.txt | Blob path | User | Azure action resolver | Wrong object action. |
| target_container | UI/API | Actions | Destination container. | Needed for migration/quarantine. | string | For migration/quarantine | quarantine | Container name | User/policy | Copy target | Copy/action fails. |
| target_prefix | UI/API | Actions | Destination prefix. | Builds target blob path. | string | No | everest/quarantine | Prefix | User | Copy target | Wrong destination. |
| target_blob_path | UI/API | Actions | Exact destination path. | Overrides target prefix. | string | No | quarantine/a.txt | Blob path | User | Copy target | Wrong destination. |
| target_tier | UI/API/Azure | Actions | Azure access tier. | Archive/retrieve/rehydrate. | string | For tier action | archive | hot, cool, cold, archive | User | Azure SetTier | Wrong storage tier. |
| rehydrate_priority | API/Azure | Actions | Rehydrate priority. | Controls archive retrieval priority. | string | No | standard | standard, high | User/API | Azure SetTier | Cost/timing impact. |
| guardian_decision | API | Actions | Guardian decision. | Safety gate. | string | For apply | allow | allow, pending, deny | Guardian/user | Action execute | Unsafe or blocked action. |
| confirmation | API/UI | Actions | Confirmation text. | Required for restricted actions. | string | For apply/delete | APPLY, DELETE | API-specific | User | Action validation | Action blocked. |
| preview_only | API/UI | Actions | Whether to preview. | Prevents mutation. | boolean | No | true | true, false | UI | Action handler | Wrong mutation mode. |

## 7. AI Provider Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| provider_id | UI/API | AI Tools | AI provider ID. | Selects provider config/runtime. | string | Yes | customer-private-llm | Catalog provider IDs | User/catalog | AI config | Wrong provider invoked. |
| endpoint_url | UI/API | AI Tools | Runtime endpoint. | Calls AI provider. | URL | Depends provider | http://localhost:7071/ai | URL | User | AI runtime client | Runtime unavailable. |
| model | UI/API | AI Tools | Model name. | Documents inference target. | string | No | local-bert-ner | Provider-specific | User/catalog | AI runtime payload | Wrong model result. |
| deployment_name | UI/API | AI Tools | Cloud/private deployment ID. | Needed by some providers. | string | No | prod-gpt | Provider-specific | User | AI runtime payload | Provider error. |
| auth_scheme | UI/API | AI Tools | Authentication method. | Secures provider call. | string | Yes | api_key | none, api_key, bearer, custom | User/catalog | AI runtime | Auth failure. |
| network_mode | UI/API | AI Tools | Network environment. | Cloud/private/air-gapped routing. | string | Yes | air_gapped | cloud_connected, private_network, air_gapped, hybrid | User | AI config | Wrong deployment expectation. |
| redaction_mode | UI/API | AI Tools | Prompt data protection. | Controls sensitive content exposure. | string | Yes | strict | none, standard, strict | User/policy | AI runtime | Sensitive data exposure. |
| prompt_audit_enabled | UI/API | AI Tools | Whether prompts are audited. | Compliance trace. | boolean | Yes | true | true, false | User/policy | Audit | Missing audit evidence. |
| allow_hitl_submission | UI/API | AI Tools | AI may submit HITL. | Governs automation. | boolean | Yes | true | true, false | User/policy | HITL flow | AI handoff blocked or too permissive. |
| allow_autonomous_apply | UI/API | AI Tools | AI may apply changes. | Controls automation risk. | boolean | Yes | false | true, false | User/policy | Action flow | Unsafe if true without controls. |

## 8. Execution, Evidence, And Store Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| lease_id | API/Data | Execution | Assigned lease ID. | Tracks work ownership. | string | Yes | lease-azureblob-... | Unique ID | Control plane | Execution state | Job trace broken. |
| job_id | API/Data | Execution | Job ID. | Tracks job. | string | Yes | job-azureblob-... | Unique ID | Control plane | Execution state | Job status broken. |
| records_scanned | API/UI | Scan | Number of scanned blobs. | Shows scan result. | integer | Output | 12 | >=0 | Scan | UI/evidence | Misleading status. |
| records_written | API/UI | Scan | Records stored locally. | Confirms persistence. | integer | Output | 12 | >=0 | Store | UI/evidence | Store mismatch. |
| path | Data | Normalized File | Blob path. | Object identity/display. | string | Yes | hr/a.txt | Blob path | Azure scan | Local store/evidence | Wrong object evidence. |
| name | Data | Normalized File | File/base name. | Display/search. | string | Yes | a.txt | File name | Azure scan | Local store | Display issue. |
| size_bytes | Data/API | Normalized/DSPM | Object size. | Risk/scan strategy. | integer | Yes | 4096 | >=0 | Azure props | DSPM/UI | Large object handling wrong. |
| content_type | Data/API | Normalized/DSPM | MIME type. | Classification/context. | string | No | application/pdf | MIME type | Azure props | DSPM/UI | Risk estimation weaker. |
| permission_risk_score | Data | Normalized | Estimated permission risk. | Prioritization. | integer | Yes | 70 | 0 to 100 | Scanner | Action plan | Wrong priority. |
| scan_timestamp | Data | Normalized | Scan time. | Audit. | timestamp | Yes | 2026-06-13T... | RFC3339 | Scanner | Evidence | Audit gap. |

## 9. Connector Runtime And Manifest Fields

| Field Name | Layer | Module | Description | Why Needed | Type | Mandatory | Sample Value | Allowed Values | Source | Target / Mapping | Impact If Wrong |
|---|---|---|---|---|---|---|---|---|---|---|---|
| connector_id | API/Manifest | Connector Catalog | Unique connector ID. | Runtime selection. | string | Yes | azureblob | Connector IDs | Manifest/catalog | Runtime invoke | Wrong connector. |
| display_name | Manifest/UI | Connector Catalog | Human name. | User selection. | string | Yes | Azure Blob Storage | Any string | Manifest | UI | Confusing UI. |
| runtime_endpoint | UI/API | Connector Runtime | Custom connector endpoint. | Invokes external runtime. | URL | For custom | http://localhost:7071/connector | URL | User/catalog | Runtime client | Runtime call fails. |
| runtime_operation | UI/API | Connector Runtime | Operation to invoke. | Tests connector behavior. | string | Yes | discover_sources | health, discover_sources, scan_metadata, preview_action, execute_action | User | Runtime payload | Wrong operation. |
| deployment_modes | Manifest | Connector Catalog | Supported environments. | Architecture fit. | array | Yes | air_gapped | cloud/private/air-gapped/hybrid | Manifest | UI | Wrong customer expectation. |

## 10. Validation Rules Summary

| Field Group | Validation Rule |
|---|---|
| Azure connection string | Must be non-empty and valid for Azure SDK. |
| Container | Required for blob list/view/create/update/actions. |
| Blob path | Required for single-object operations. |
| Delete | Requires `confirm=DELETE`. |
| Create | Fails if blob exists unless overwrite is allowed. |
| Metadata/tags | Must contain key-value pairs. |
| DSPM scan tier | Must be metadata, rules, model, or deep. |
| Keyword pattern | Must have non-empty pattern. |
| Keyword severity | Defaults to medium if missing or unknown. |
| Keyword match mode | Defaults to contains if missing. |
| Action apply | Should require Guardian allow, policy, and HITL where relevant. |

## Open Field Dictionary Gaps

| Gap | Why It Matters |
|---|---|
| Production DB schema is not finalized. | DB layer fields need final table/column names. |
| Production RBAC fields are not finalized. | Role/permission dictionary is planned. |
| Model runtime payloads are contract-level. | BERT/domain model runtime fields need final integration detail. |
| Workbench history persistence is planned. | History table/entity fields need to be added later. |
| DSPM result persistence is planned. | DSPM table/entity fields need to be added later. |
