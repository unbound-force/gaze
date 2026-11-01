---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:16:50Z
identity: issue-triage-20261101T181650-jay-flowers
tier: draft
---

Stale-reference cleanup issues age — re-verify every claimed occurrence against current disk state at triage or implementation time. Gaze issue #244 (uf. namespace migration sweep; created 2026-09-01, re-verified 2026-11-01): 3 of 4 claimed stale /review-council references no longer reproduced — .opencode/skills/review-context/SKILL.md:19 and .opencode/agents/cobalt-crush-dev.md:127 had been re-scaffolded to the prefixed /uf.review-council form, and docs/reference/cli/init.md had been swept by the later divisor-gaze-testing reference sweep — while both constitution references (.specify/memory/constitution.md:10 sync-impact header, :133 Review Council Gate MUST rule) persisted exactly. Only files no scaffold manages (governance docs, hand-maintained docs) stay stale after a tool-driven migration. Two gotchas when running the issue's acceptance pattern rg -P '(?<!uf\.)/review-council': (1) it also matches filesystem paths that are not command references (.uf/dewey/sources.yaml 'path: "../review-council"'); (2) completed specs/ and openspec/ change records match but are point-in-time historical records exempt from cleanup — a zero-match AC needs explicit exclusion lists or it cannot be met literally. Rule: run the issue's own grep first and implement only against current matches.
