# Email Draft To Cuong

Subject: Request for Review: Everest Azure Blob Architecture Alignment

Hi Cuong,

I have prepared an architecture review comparing our current Everest Azure Blob implementation with the proposed Data Agent Cell / DSPM policy enforcement architecture.

The review focuses on whether our current implementation is aligned with the target enterprise architecture, especially around:

- Orchestrator-first control flow
- Guardian Agent as the mandatory policy enforcement point
- Zero-copy data access expectations
- Connector-agnostic design
- AI recommendation boundaries
- HITL approval flow
- Tamper-evident provenance and evidence
- UI simplification for enterprise users

The main finding is that our current implementation has strong Azure Blob capability, but it is still too portal/API-handler driven. To align with the team architecture, we should move toward a single enforced runtime path:

Portal -> Data Agent Cell Orchestrator -> Guardian Decision -> Zero-Copy Accessor / Action Executor -> Provenance

I would like your review on the attached document, especially the decision points near the end:

1. Should the Orchestrator become the mandatory runtime entry point for every scan, read, AI recommendation, and action?
2. Should Workbench be repositioned as an advanced/expert tool rather than the primary customer journey?
3. What exact definition of zero-copy should Everest commit to?
4. Is orchestrator-managed durable execution state acceptable under the "no queues" principle?
5. Should every Guardian decision require policy bundle version/hash before execution?
6. Should AI be formally restricted to recommend, explain, classify, and prefill HITL only?
7. Should Azure Blob-specific UI be refactored behind a generic connector workbench contract?

Once we align on these points, the next step will be to create the final architecture visual as a dense PNG and an animated GIF sequence for team presentation.

Please let me know your feedback or any changes you would like before we finalize the architecture direction.

Thanks,  
Arun

