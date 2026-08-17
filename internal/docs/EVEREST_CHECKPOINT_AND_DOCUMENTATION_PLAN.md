# Everest Checkpoint and Documentation Plan

Checkpoint date: 2026-06-13

## Purpose

This file captures the current Everest application checkpoint and the documentation package required before the next implementation phase resumes.

The attached documentation prompt asks for a complete self-learning document for a fresher/new joiner. The target reader should understand the business problem, system architecture, modules, screens, fields, APIs, data flow, deployment, security, testing, troubleshooting, and learning path without needing direct help from architects, developers, QA, or business users.

## Current Application Checkpoint

Application name: Everest Enterprise Data Operations - Azure Blob Edition

Business purpose:

Everest is an enterprise data operations and governance portal focused first on Azure Blob Storage. It helps customers discover storage sources, scan objects, evaluate risk, generate evidence, run AI/DSPM recommendations, route risky actions through HITL approvals, and apply governed Azure Blob actions.

Current primary users:

- Data operations user
- Data steward
- Security officer
- Compliance/audit user
- Platform administrator
- Solution architect
- QA/tester
- New connector/plugin developer

Current product modules:

- Enterprise Dashboard
- Source Inventory
- Azure Blob Workbench
- Connector Catalog
- AI Tools
- Onboarding
- Scans and Jobs
- Evidence Center
- Guardian Agent
- Action Center
- Policy Governance
- Agent Working Store
- Diagnostics
- Azure Blob connector runtime
- Connector/plugin framework
- AI provider framework
- DSPM content intelligence engine
- Control plane and data agent APIs
- Local working store

Current major capabilities:

- Save Azure Blob connection string securely in the local agent secret store.
- Discover Azure Blob containers.
- List, view, create, update, download, tag, and delete Azure Blob test objects through UI.
- Use dry-run by default for Workbench operations.
- Require explicit `DELETE` confirmation for delete.
- Provide a day/night theme toggle.
- Provide AI action verbs instead of making chat the default.
- Generate AI-assisted action recommendations.
- Produce editable action canvas artifacts.
- Route risky actions to HITL or Action Center.
- Execute governed Azure Blob actions.
- Support connector catalog and sample plugin runtime.
- Support AI provider catalog/config/test/runtime invocation.
- Support DSPM scan tiers: `metadata`, `rules`, `model`, `deep`.
- Support DSPM built-in rule detection for common entities.
- Support DSPM keyword pattern matching with built-in packs and custom patterns.
- Generate recommended metadata and index tags from DSPM results.

Recent implementation checkpoint:

- `internal/dspm/engine.go` added connector-agnostic DSPM logic.
- `internal/api/dspm_handlers.go` added DSPM catalog and content scan APIs.
- `web/index.html` added Azure Blob Workbench, AI verbs, action canvas, DSPM controls, keyword pattern controls, and day/night theme.
- `internal/api/azureblob_workbench_handlers.go` added Azure Blob Workbench backend operations.
- `docs/DSPM_CONTENT_INTELLIGENCE_ARCHITECTURE.md` documented DSPM design.
- `docs/AI_INTERFACE_ARCHITECTURE_DECISION.md` documented verb-first AI interface direction.
- `docs/AZURE_BLOB_TEST_TOOL.md` documented CLI and UI testing flows.

Known validation gap:

Local PowerShell process creation was blocked in the Codex sandbox, so final `gofmt`, `go test ./...`, and browser verification must be run in the user's Azure-connected environment.

Known product gaps still open:

- Persist Workbench operation history.
- Persist AI recommendation artifacts.
- Persist DSPM scan results in local working store.
- Add background delegated deep scan jobs.
- Connect `local-bert-ner` and `domain-bert-dspm` model profiles to real model runtimes.
- Add customer-managed keyword dictionary persistence.
- Convert Azure Blob Workbench into generic Connector Workbench contract.
- Add role-based access enforcement for all enterprise deployment modes.
- Add full authentication/authorization model.
- Add production deployment topology and environment-specific config.
- Add formal audit export for Workbench and DSPM actions.

## Documentation Package Required

The prompt asks for one complete self-learning document. Because Everest is now multi-module, the recommended output is:

1. A master self-learning guide.
2. Supporting appendices for fields, APIs, tests, troubleshooting, and glossary.

The master guide should be written so a fresher can read it end to end. The appendices should be detailed enough for QA, support, and implementation teams.

## Recommended Documentation Files

### 1. `everest_self_learning_master_guide.md`

Purpose:

Primary self-learning document for a new joiner.

Required sections:

- Executive Summary
- Business Journey
- System Overview
- Module Breakdown
- Component Interaction Flow
- Screen-by-Screen Guide
- Field Dictionary summary
- API and Event Mapping summary
- Database and Data Flow summary
- Deployment and Environment View
- Security and Access Model
- Test Strategy summary
- Troubleshooting summary
- Fresher Learning Path
- Quick Reference Appendix
- Open Questions
- Missing Information Checklist
- Suggested Review Sequence

### 2. `everest_business_journey_and_roles.md`

Purpose:

Explain business workflows and user personas in plain English.

Coverage:

- Why Everest exists
- Customer problems solved
- Azure Blob governance journey
- AI-assisted action journey
- DSPM content intelligence journey
- HITL approval journey
- Auditor/evidence journey
- Roles and responsibilities

### 3. `everest_architecture_and_component_map.md`

Purpose:

Explain system architecture and component communication.

Coverage:

- Browser UI
- Data Agent API
- Control Plane API
- Azure Blob connector
- Connector runtime framework
- AI provider framework
- DSPM engine
- Policy resolver
- Guardian Agent
- Local working store
- Secret store
- Plugin/sample runtime
- External Azure Blob Storage
- Mermaid architecture diagram
- Mermaid sequence diagrams
- Mermaid flowcharts

### 4. `everest_module_breakdown.md`

Purpose:

Explain every module in the prompt's required format.

For each module include:

- Module name
- Purpose
- Why it was created
- Key responsibilities
- Upstream dependencies
- Downstream dependencies
- Internal components
- Main business rules
- Failure impact
- Logging/monitoring points
- How to test it

Modules to cover:

- Dashboard
- Source Inventory
- Azure Blob Workbench
- Connector Catalog
- AI Tools
- Onboarding
- Scans and Jobs
- Evidence Center
- Guardian Agent
- Action Center
- Policy Governance
- Agent Working Store
- Diagnostics
- Azure Blob connector
- Workbench backend API
- DSPM engine
- AI provider framework
- Connector/plugin framework
- Control plane
- Local store
- Secret store

### 5. `everest_screen_by_screen_guide.md`

Purpose:

Explain every UI screen, field, button, dropdown, and expected behavior.

Screens to cover:

- Enterprise Dashboard
- Source Inventory
- Azure Blob Workbench
- Connector Catalog
- AI Tools
- Onboarding
- Scans and Jobs
- Evidence Center
- Guardian Agent
- Action Center
- Policy Governance
- Agent Working Store
- Diagnostics

For each screen include:

- Purpose
- User role
- Navigation path
- Fields and controls
- Why each field/control exists
- Validations
- API calls
- Data affected
- Common mistakes
- Test scenarios

### 6. `everest_field_dictionary.md`

Purpose:

Detailed field dictionary across UI, API, and data objects.

Required columns:

- Field Name
- Layer
- Module
- Description
- Why Needed
- Type
- Mandatory
- Sample Value
- Allowed Values
- Validation Rule
- Source
- Target / Mapping
- Impact If Wrong

Important field groups:

- Azure Blob config fields
- Workbench operation fields
- AI recommendation fields
- HITL/action fields
- DSPM scan fields
- Keyword pattern fields
- Connector manifest fields
- AI provider manifest/config fields
- Execution/job/evidence fields
- Policy/governance fields

### 7. `everest_api_event_mapping.md`

Purpose:

Detailed API/event map with request/response and tests.

API groups to cover:

- `/v1/connectors/azureblob/config`
- `/v1/connectors/azureblob/test`
- `/v1/connectors/azureblob/scan`
- `/v1/connectors/azureblob/status`
- `/v1/connectors/azureblob/containers`
- `/v1/connectors/azureblob/actions/execute`
- `/v1/tools/azureblob/workbench`
- `/v1/dspm/catalog`
- `/v1/dspm/content-scan`
- `/v1/ai/providers/catalog`
- `/v1/ai/providers/config`
- `/v1/ai/providers/test`
- `/v1/ai/runtime/invoke`
- `/v1/ai/azureblob/recommendations`
- `/v1/connectors/catalog`
- `/v1/connectors/runtime/invoke`
- `/v1/plugins/contract-test`
- `/v1/execution/azureblob/start`
- `/v1/execution/start`
- `/v1/actions/plan`
- `/v1/guardian/decisions`
- `/v1/localstore/normalized-files`
- `/v1/policy/resolve`
- `/v1/policy/artifact/trust`
- `/healthz`

For each API include:

- Purpose
- Request source
- Consumer
- Method
- Input fields
- Output fields
- Auth/security notes
- Error cases
- Retry/idempotency behavior
- Sample request
- Sample response
- Test cases

### 8. `everest_database_and_data_flow.md`

Purpose:

Explain local data objects, working store, evidence, runtime state, and data lifecycle.

Coverage:

- Normalized file records
- Manifest summary/details
- Action plans
- Guardian decisions
- Runtime audit
- Approval/HITL data
- Azure Blob config in runtime store
- Secrets in secret store
- DSPM result persistence gap
- Workbench history persistence gap
- Data flow from UI to API to store and back

### 9. `everest_deployment_environment_security.md`

Purpose:

Explain environments, configuration, secrets, deployment, health checks, and security.

Coverage:

- DEV/QA/UAT/PROD expectations
- Local demo mode
- Air-gapped/private-network/cloud-connected modes
- Environment variables
- Secret handling
- Azure Blob credentials
- AI provider credentials
- Health checks
- Startup dependencies
- Rollback considerations
- Security model
- Sensitive fields
- Audit trail
- Abuse/misuse scenarios

### 10. `everest_test_strategy_and_test_cases.md`

Purpose:

Provide QA-ready test scenarios.

Required sections:

- Smoke tests
- Happy path tests
- Negative tests
- Boundary tests
- Integration failure tests
- Role-based tests
- Performance/basic load tests
- Recovery tests
- Data validation tests
- Regression checklist

Each test must include:

- Test ID
- Scenario name
- Module
- Preconditions
- Steps
- Test data
- Expected result
- Failure indicators
- Priority

Must include tests for:

- Azure Blob connection string save
- Container discovery
- Blob list/view/create/update/download/delete
- Dry-run behavior
- Delete confirmation
- AI recommendation verbs
- HITL handoff
- DSPM scan tiers
- Keyword pattern matching
- BERT/model profile readiness
- Air-gapped assumptions
- Connector runtime contract
- AI provider runtime contract

### 11. `everest_troubleshooting_runbook.md`

Purpose:

Help support/new joiners diagnose failures.

Issues to cover:

- Portal does not start
- Agent API unavailable
- Azure connection fails
- Container list empty
- Blob view/download fails
- Metadata/tags fail due Azure permissions
- Delete blocked
- AI provider unavailable
- Sample plugin runtime unavailable
- DSPM scan returns no entities
- Keyword patterns not matching
- HITL request not created
- Workbench history missing
- Local store empty
- Control plane unavailable

For each issue include:

- Symptom
- Possible root cause
- How to verify
- Logs/API/DB checks
- Resolution steps
- Escalation owner

### 12. `everest_fresher_learning_path_and_quick_reference.md`

Purpose:

Give a new joiner a structured one-day learning path.

Coverage:

- Hour 1: business purpose
- Hour 2: architecture
- Hour 3: modules and screens
- Hour 4: APIs and data flow
- Hour 5: Azure Blob Workbench hands-on
- Hour 6: DSPM, AI, HITL, Guardian
- Hour 7: testing and troubleshooting
- Hour 8: independent walkthrough

Appendix:

- Acronyms
- Important URLs
- Important APIs
- Important jobs/services
- Key data objects
- Critical business rules
- Common production issues
- Do's and don'ts

## Recommended Next Execution Plan

### Phase 1: Freeze and Validate Current Build

Run in Azure-connected environment:

```powershell
go test ./...
go run ./cmd/policy-agent
```

Validate:

- Portal loads.
- Day/night theme works.
- Source config saves.
- Containers are discovered.
- Azure Blob Workbench operations work.
- AI verbs render recommendations.
- DSPM scan works with keyword matching.
- HITL handoff works or Action Center prefill works.

### Phase 2: Generate Master Documentation Draft

Create `everest_self_learning_master_guide.md` first.

The master guide should use the attached prompt's 15-section structure exactly.

### Phase 3: Generate Appendices

Create the field dictionary, API mapping, data flow, test strategy, troubleshooting, and quick reference documents.

### Phase 4: Review Cycle

Suggested review order:

1. Architect review:
   - Architecture correctness
   - Module boundaries
   - Future connector extensibility
   - Security assumptions

2. Developer review:
   - API names
   - Payloads
   - Actual code behavior
   - Missing implementation notes

3. QA review:
   - Test scenario completeness
   - Negative cases
   - Edge cases
   - Expected outputs

4. Business Analyst/Product review:
   - Business journey clarity
   - User roles
   - Customer language
   - Training readiness

## Open Questions Before Writing Final Docs

1. Should the final self-learning guide be one large markdown document, or split into the files listed above?
2. Should we include screenshots from the running portal?
3. Should the documentation describe current implementation only, or also clearly mark planned enterprise capabilities?
4. Which customer roles should be treated as primary: data steward, security, compliance, operator, admin, or all?
5. Do we need Word/PDF export after markdown is complete?
6. Should we include exact Azure permissions/SAS requirements for every Workbench action?
7. Should DSPM model profiles be documented as planned contracts or current runtime capabilities?
8. Should we document air-gapped mode as supported for rules/keyword scans only until model runtime is integrated?

## Missing Information Checklist

- Final application/product name.
- Final target customer personas.
- Production authentication and authorization model.
- Production deployment topology.
- Database choice for production persistence.
- Audit retention requirements.
- Log locations and monitoring tools.
- Azure permissions model for customer environments.
- AI provider list approved for production.
- BERT/local model runtime selection.
- Data retention and purge policy.
- Compliance standards to mention.
- Role matrix and permission boundaries.
- Screenshots, if required.

## Resume Point

Resume from here by generating:

1. `everest_self_learning_master_guide.md`
2. `everest_field_dictionary.md`
3. `everest_api_event_mapping.md`
4. `everest_test_strategy_and_test_cases.md`

Recommended first file: `everest_self_learning_master_guide.md`.
