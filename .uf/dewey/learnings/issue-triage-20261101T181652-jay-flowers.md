---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:16:52Z
identity: issue-triage-20261101T181652-jay-flowers
tier: draft
---

Migration-cleanup issues go stale between filing and implementation — re-verify every claimed file against the current tree before acting on the issue's file list. In unbound-force/gaze #244 (filed 2026-09-01, triaged 2026-11-01), three of the four listed files (.opencode/skills/review-context/SKILL.md, docs/reference/cli/init.md, .opencode/agents/cobalt-crush-dev.md) had already been fixed by later scaffold runs and renames; only .specify/memory/constitution.md (lines 10, 133) still carried the pre-`uf.`-migration `/review-council` reference, and the exempted AGENTS.md changelog line had drifted from 531 to 586. The durable source of truth for rename-residue issues is the acceptance grep (e.g. `rg -P '(?<!uf\.)/review-council'` scoped to actionable docs, excluding historical changelog/spec entries), not the issue's enumerated file list or line numbers. Implementation should run the grep first and treat the enumeration as hints.
