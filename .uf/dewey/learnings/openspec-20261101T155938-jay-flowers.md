---
tag: openspec
author: jay-flowers
category: gotcha
created_at: 2026-11-01T15:59:38Z
identity: openspec-20261101T155938-jay-flowers
tier: draft
---

When a review-council advisory revises an implementation's observable contract mid-change (exit codes, error strings, enumerated counts), the fix must be swept across every artifact authored earlier in the same OpenSpec change — delta-spec scenario THEN clauses, design-decision case lists (D8-style enumerated expectations), task descriptions carrying counts — not just the code, tests, and the revised design decision. In gaze's validate-workflow-floor change, the pass-1 exit-taxonomy fix (repo-state failures 1→2) landed in the script, test matrix, and design D1/D3/D4, but spec.md scenarios and D8's case list still asserted exit 1 and tasks.md still said "twelve cases" against a 14-case matrix — drift that would have synced into permanent openspec/specs/ at archive time (Spec Artifact Immutability then forbids correction). Post-fix sweep heuristic: grep the entire change directory for the superseded values ("status 1", "→ 1", old counts) before requesting the next review pass or archiving; scenario THEN-clauses and design case lists are contracts, not prose.
