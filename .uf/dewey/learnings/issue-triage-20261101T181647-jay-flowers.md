---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:16:47Z
identity: issue-triage-20261101T181647-jay-flowers
tier: draft
---

Rename-cleanup issue triage: file lists rot, executable acceptance criteria self-correct. Triage of gaze issue #244 (filed 2026-09-01, assessed 2026-11-01) found that 3 of 4 enumerated files already carried the replacement text two months later — fixed independently by scaffold upgrades and doc sweeps — while the issue's acceptance criterion (a project-wide negative-lookbehind grep for the old name) still correctly isolated the one genuinely stale file (the constitution). Lesson: when triaging rename-migration cleanup issues, verify current state by running the acceptance grep yourself rather than trusting the enumerated file list, and treat the executable criterion as the scope contract. Two recurring gaps to flag in grep-based "zero matches" criteria: (1) fuzzy exclusion classes ("actionable docs", "excluding historical entries") are not mechanically enumerable — literal matches include non-command strings (e.g., a Dewey source path "../review-council") and historical task/spec records that need judgment to exempt; (2) the fix's own paper trail is self-referential — a Sync Impact Report or changelog entry describing the rename matches the old-name grep unless worded using only the new name, so prescribe new-form-only wording for migration records or enumerate exclusions by file.
