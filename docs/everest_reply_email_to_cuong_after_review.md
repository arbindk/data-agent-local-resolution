# Reply Email Draft To Cuong

Subject: Re: Everest Azure Blob Architecture Alignment

Hi Cuong,

Thank you for the detailed review and confirmation.

I agree with the direction and will treat the following as the accepted architecture baseline:

- The Data Agent Cell Orchestrator is the mandatory runtime entry point for scans, reads, AI recommendations, and actions.
- Guardian remains the non-bypassable policy enforcement gate before any mutation.
- Everest should operate autonomously by default within policy guardrails.
- Allowed actions can execute automatically when policy explicitly permits them.
- HITL tasks should be created only when Guardian requires review or obligations cannot be completed autonomously.
- Zero-copy will use the practical definition: no persistent full duplicate copies by default, with reference/handle/stream/range/bounded preview and ephemeral feature extraction allowed.
- DuckDB will be treated as transient batch working memory for normalized signals, scoring, and aggregation.
- Guardian owns decision provenance, including policy bundle version/hash, rules applied, reasoning, outcome, and obligations.
- Azure Blob should remain our first complete connector, but the design will move behind a generic connector contract.

I have updated the architecture alignment notes to reflect your feedback and will use them as the source for the final visual package.

Next, I will prepare:

1. A dense architecture PNG for review.
2. An animated sequence GIF showing the autonomous MVS1 flow.
3. A Phase 1 refactor plan focused on request envelopes, Guardian decisions, action receipts, provenance records, and enforcement tests.

The proposed final flow will be:

Portal / Intent -> Data Agent Cell Orchestrator -> Discovery & Analysis -> Guardian Decision -> Approved Action Execution or HITL Task -> Decision Provenance -> Elasticsearch Summary

Once the visual is ready, I will share it for review before we start the refactor.

Thanks,  
Arun

