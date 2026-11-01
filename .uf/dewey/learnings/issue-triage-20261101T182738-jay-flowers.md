---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:27:38Z
identity: issue-triage-20261101T182738-jay-flowers
tier: draft
---

Duplicate detection for stale-reference cleanup issues must compare current actionable scope, not filed scope. In unbound-force/gaze, #292 (filed 2026-09-29) narrowed the older sweep issue #244 (filed 2026-09-01) to the two project-owned files after excluding uf-managed ones, claiming fixes in .specify/memory/constitution.md:10,133 and docs/reference/cli/init.md:31. By triage on 2026-11-01, an unrelated reference sweep (divisor-gaze-testing) had already removed the init.md reference, leaving both open issues with identical remaining scope (constitution:10,133) — they had silently converged on the same two-line edit. Overlapping cleanup issues decay at different rates because parallel remediation fixes enumerated targets incidentally, so a filed-scope comparison misstates overlap. Rule: at triage, re-verify every enumerated target of both issues against HEAD (the literal old-form grep is a complete discriminator when the new form does not contain the old as a substring), classify duplication on what remains actionable, and recommend consolidation — close one issue and port the survivor's precise line references and scope boundaries (e.g., #292's exclusion of uf-managed .opencode/ paths, which matches scaffold-ownership conventions).
