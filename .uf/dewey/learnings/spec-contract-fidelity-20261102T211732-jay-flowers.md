---
tag: spec-contract-fidelity
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:17:32Z
identity: spec-contract-fidelity-20261102T211732-jay-flowers
tier: draft
---

Spec contracts that pair tolerance with accountability clauses ("skipped AND counted", "ignored AND logged") reliably lose the accountability half at implementation time: parseEvents in internal/aireport/opencode_events.go (PR #314) implemented the skip — with a rationale comment justifying it — but dropped the "counted" clause from specs/040-deterministic-ai-report-rendering/contracts/opencode-adapter.md Extraction step 1, and the fixture suite (only all-invalid not_json.txt, no mixed valid+junk stream) could not detect the omission. Prevention: when implementing a contract's numbered step-list, diff every clause — including trailing participles like "and counted" — against the implementation, and add one fixture exercising the clause's observable effect (a mixed valid+invalid stream asserting the surfaced skip count).
