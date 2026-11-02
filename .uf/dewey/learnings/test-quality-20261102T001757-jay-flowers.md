---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:57Z
identity: test-quality-20261102T001757-jay-flowers
tier: draft
---

Guard-clause input-class coverage (gaze-init-triage-advisor review, 2026-11-01): a YAML guard of the form `x == nil || x.Kind != SequenceNode` has two operands, and a table suite that stages only key-absent fixtures short-circuits on the nil operand forever — the Kind operand never executes under test. When a change extends such a guarded loop to a new member AND re-states the guard as a delta-spec MUST ("missing or is not a sequence"), derive a scenario and a table case for EACH input class, not just the newly-added member's missing-key case: a null-valued key (e.g. `triage:` left after commenting out the list) or a mapping value (`triage: {}`) is what exercises the Kind operand. In gaze's syncReviewMatrix mode loop (internal/scaffold/ufconfig.go), the suite had missing-key cases for two of four modes but zero non-sequence cases for any mode, leaving the spec'd non-sequence half of the skip contract unverified.
