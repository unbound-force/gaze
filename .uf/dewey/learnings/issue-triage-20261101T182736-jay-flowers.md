---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:27:36Z
identity: issue-triage-20261101T182736-jay-flowers
tier: draft
---

Rename-cleanup trackers converge into exact duplicates as parallel remediation shrinks both. In gaze, #244 (filed 2026-09-01, four stale /review-council files) and #292 (filed 2026-09-29, constitution.md:10,133 + docs/reference/cli/init.md:31) were only partially overlapping at filing time; an unrelated intervening docs sweep (divisor-gaze-testing) and uf scaffold refreshes remediated the .opencode/ and init.md targets, leaving both issues' remaining actionable scope identical (constitution.md:10,133 only) and rotting #292's init.md claim entirely. Triage rule: run duplicate detection against each issue's CONVERGED scope (the current grep matches), not its filed scope — partial overlaps become exact duplicates as unrelated fixes land. Close the later-filed issue as duplicate of the earlier one and port its crisper acceptance criteria into the survivor; two live trackers on the same lines invite duplicate/conflicting PRs.
