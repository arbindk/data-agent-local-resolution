# Everest Connector Framework Architecture

## Product Intent

Everest must support customers across cloud-connected, private-network, hybrid, and air-gapped environments. Connectors are therefore treated as pluggable capability providers, not hard-coded UI pages. Azure Blob is the first fully implemented connector, while future and customer-owned connectors use the same manifest, normalized object, evidence, action, Guardian, and HITL contracts.

## Deployment Modes

| Mode | Customer Environment | Everest Behavior |
| --- | --- | --- |
| cloud_connected | Public cloud APIs are reachable | Connector can use public SDK/API endpoints and online health checks. |
| private_network | Private endpoints, VPN, ExpressRoute, intranet shares | Data Agent runs inside the customer network and uses local/private routing. |
| air_gapped | No internet path | Customer imports policy bundles, workflow bundles, connector manifests, and runtime packages manually. Evidence can be exported out-of-band. |
| hybrid | Mixed online/offline control | Control Plane can sync when reachable; Data Agent keeps local execution and evidence working when disconnected. |

## Connector Contract

Every connector is described by `contracts/connector_manifest.schema.json`.

The manifest declares:

- connector identity and runtime type
- supported deployment and network modes
- authentication schemes and secret fields
- user-facing configuration fields
- scan/write/action capabilities
- supported object model and `file_id` format
- action safety controls
- AI recommendation posture
- support ownership for Data Dynamics, partner, or customer-owned code

## Runtime Shapes

| Runtime | Use Case |
| --- | --- |
| `builtin` | Product-owned connector compiled into the Data Agent, such as Azure Blob. |
| `external_process` | Customer or partner ships an executable that the Data Agent invokes locally. |
| `http` | Customer exposes a local/private connector service endpoint. |
| `external_process_or_http` | Planned or template connector that can be implemented either way. |

## Safety Model

AI can recommend actions, prefill action requests, and submit pending HITL approvals. AI cannot silently mutate a customer source.

Real connector actions must pass:

1. policy and role evaluation
2. Guardian decision
3. HITL approval when required
4. action mode set to `apply`
5. write-back enabled
6. connector-specific confirmation such as `APPLY` or `DELETE`

## Customer Connector Onboarding

1. Customer creates a connector manifest.
2. Customer validates manifest through `/v1/connectors/manifest/validate`.
3. Customer installs the connector runtime inside its environment.
4. Connector emits normalized records using `contracts/normalized_batch.schema.json`.
5. Connector actions follow `contracts/connector_action.schema.json`.
6. Everest portal displays capabilities and only enables safe workflows based on the manifest.

## Current Implementation

- Azure Blob: available, built-in, end-to-end metadata scan and governed actions.
- Customer Filesystem sample: custom manifest example, contract validation only.
- S3, SMB, NFS, SharePoint, OneDrive, GCS: planned framework entries to show future connector shape and customer extensibility.
