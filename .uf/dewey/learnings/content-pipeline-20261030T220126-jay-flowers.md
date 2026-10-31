---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-10-30T22:01:26Z
identity: content-pipeline-20261030T220126-jay-flowers
tier: draft
---

Curator code-mode reviews close two verification loops on documentation gates. (1) A documentation advisory acknowledged in the PR body is awareness, not remediation: when a pre-PR review flags a docs-tree contract site as stale (gaze PR #305 — docs/architecture/extending.md runSubprocess/formatStderrSuffix helper bullets) and the PR ships the guide and AGENTS.md updates but not that site, the curator converts the acknowledged advisory into a formal MEDIUM finding so the council fix loop resolves it before merge; post-merge, an unfixed doc advisory is ordinary documentation drift. (2) The defeasible Website Documentation Gate pattern (spec records N/A plus an open question; a human may override by filing) closes only when verified: gh issue view the website issue to confirm it exists and accurately scopes the work (unbound-force/website#346 for redact-env-values-from-adapter-stderr — open and accurate), and grep the changed helper's name across docs/ to catch contract-description sites outside the Documentation Validation Gate's enumerated file list — in gaze the sweep isolated exactly one stale site, confirming it is cheap and precise.
