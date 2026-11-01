---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:27:41Z
identity: issue-triage-20261101T182741-jay-flowers
tier: draft
---

Migration-residue cleanup generates duplicate issues over time — intersect the new issue's file list with the older umbrella issue's list, re-verified against the current tree, because the intersection is the remaining scope. In unbound-force/gaze, the uf. namespace migration produced #244 (filed 2026-09-01, four files) and later #292 (filed 2026-09-29, two files, unaware of #244); by 2026-11-01 the only still-stale location named by either issue is .specify/memory/constitution.md:10,133 — identical in both — because unrelated later work (the divisor-gaze-testing doc sweep and a uf scaffold refresh) remediated every other enumerated file. A triage-time literal grep for '/review-council' (which is not a substring of '/uf.review-council') establishes both remaining scope and duplicate status in one step. When choosing the surviving tracker, compare acceptance-criteria quality, not just recency: #244 captured the constitution's PATCH-bump amendment protocol (1.3.0 → 1.3.1 plus a Sync Impact Report entry) that #292 omitted — older-and-complete beats newer-and-precise when consolidating duplicates.
