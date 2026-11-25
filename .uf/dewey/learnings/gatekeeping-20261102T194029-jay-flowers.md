---
tag: gatekeeping
author: jay-flowers
category: pattern
created_at: 2026-11-02T19:40:29Z
identity: gatekeeping-20261102T194029-jay-flowers
tier: draft
---

Spec-authorized gate modifications are distinguishable from gate tampering by three verifiable markers, and reviewers should check all three before flagging: (1) the authorizing FR names the exact gate being changed — spec 040 FR-024 names the File Sync Invariants row and TestEmbeddedPromptMatchesScaffold it removes, and FR-029/T049 scope the zero-result gate to CRAP-selected runs; (2) the change is recorded in the change log entry (AGENTS.md Recent Changes names every removal); (3) the remaining gates are explicitly affirmed unchanged (SC-011 'no gate value being changed', dogfood thresholds 38/6/8 restated, no .github/workflows files in the diff). PR 314 exercised this pattern cleanly: removing a CI-enforced invariant test was legitimate because the removal itself was the specified deliverable with a human-answered clarification behind it, while every untouched gate stayed byte-identical.
