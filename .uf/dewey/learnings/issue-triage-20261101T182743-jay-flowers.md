---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:27:43Z
identity: issue-triage-20261101T182743-jay-flowers
tier: draft
---

Overlapping stale-reference cleanup issues converge into duplicates as parallel remediation advances. Gaze #244 (filed 2026-09-01) and #292 (filed 2026-09-29) both enumerated pre-namespace /review-council references in project-owned files; by 2026-11-01 unrelated work (uf scaffold refresh, divisor-gaze-testing doc sweep) had completed every enumerated target except .specify/memory/constitution.md:10,133, leaving both open issues describing the identical remaining two-line fix. Two triage rules follow: (a) a newer cleanup filing can be born partially stale — #292's docs/reference/cli/init.md target was accurate at filing but was completed weeks later by an unrelated reference sweep, so re-verify every enumerated target against the current tree; (b) when a duplicate candidate exists, compare remaining scope (current grep results), not filed scope — overlapping cleanup issues converge as remediation progresses, and one tracker entry should be closed before work begins to prevent double-fixing and tracker waste.
