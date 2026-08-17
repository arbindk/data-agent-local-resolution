# AI Interface Architecture Decision

## Decision

Everest should not make chat the default interface for repeat enterprise operations. Chat remains available as an exploration and explanation side channel, but operational Azure Blob workflows should be exposed through:

- Verb surfaces for common actions
- Editable action artifacts/canvases
- Delegated assessments for multi-step work
- Ambient suggestions based on selected objects and evidence
- HITL gates for risky or destructive actions

## Why

Enterprise operators usually know the job they need to complete: discover containers, inspect objects, tag sensitive data, prepare archive, quarantine, migrate, or delete. A blank AI chat box adds avoidable work because the user has to restate context that Everest already has.

The preferred product shape is:

- The user selects a container, blob, job, policy, or evidence row.
- Everest offers the relevant verbs.
- AI generates an editable action artifact.
- Safe actions can be previewed or applied under policy.
- Risky actions are routed to HITL approval.
- The audit trail captures recommendation, rationale, decision, and final action.

## Applied Changes

The Azure Blob Workbench now uses AI behind operational verbs:

- Recommend Tags
- Complete Metadata
- Prepare Archive
- Prepare Quarantine
- Delegate Assessment

The Workbench also includes:

- Ambient suggestions when an object is selected or viewed
- An editable action canvas for AI-produced metadata, tags, and rationale
- A safe-action preview/apply path
- HITL handoff for restricted actions
- Optional chat/instruction field as a side channel, not the primary workflow

## Product Rule

For future Everest AI features:

1. If the workflow is repeated, build a button or action menu.
2. If the AI produces output the user must edit, render it as an artifact/canvas.
3. If the task has multiple steps, make it a delegated job with status and audit.
4. If context is already visible in Everest, do not require the user to type it again.
5. Keep chat only for ambiguous exploration, explanation, and exception handling.

## Testing Expectations

- A new user should complete a real Azure Blob test workflow in under 90 seconds without typing a free-form AI prompt.
- AI recommendations should be actionable from selected container/blob context.
- Safe metadata and index tag recommendations should support dry-run and controlled apply.
- Archive, quarantine, migration, and delete recommendations should require HITL or explicit confirmation.
- Every AI-assisted action should preserve rationale and audit context.
