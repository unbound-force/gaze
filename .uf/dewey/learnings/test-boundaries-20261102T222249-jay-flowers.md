---
tag: test-boundaries
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:22:49Z
identity: test-boundaries-20261102T222249-jay-flowers
tier: draft
---

Verifying a reviewer's "no test covers X" claim requires searching for incidental coverage, not just dedicated test cases. On gaze PR #314 feedback item 6, "a regression emitting a spurious recommendation for empty tallies would pass" was false: TestAssess_FixStrategyTemplates asserts an exact recommendation count of 1 on zero-tally payloads with the classification section present, so any empty-tally emission fails five subtests. The real gaps were narrower: an omitted-conditions test that returns at an earlier guard (section absent) leaves later guards unpinned, the equal-tally boundary (ambiguous == decided) had no pin at all, and the incidental pin was fragile — it silently disappears when the neighboring fixture changes. Accept test-gap feedback on the value of an explicit boundary case, but verify which regressions currently fail before endorsing the reviewer's specific failure-mode claim.
