---
tag: documentation
author: jay-flowers
category: decision
created_at: 2026-11-10T01:28:38Z
identity: documentation-20261110T012838-jay-flowers
tier: draft
---

In unbound-force/gaze, the AGENTS.md Website Documentation Gate (file an issue in unbound-force/website for user-facing behavior changes) is applied by repo precedent to feature/capability changes — new commands, flags, agent registrations, install flows, convention packs (website issues #345, #352, #363-366) — and not to behavior-affecting bug fixes whose documentation lands in-repo: redact-env-values-from-adapter-stderr, quality-empty-results-gate, fix-duplicate-scaffold-marker, and fix-nested-coverage-recursion (PR #326) all updated README/docs/reference in-repo with no website issue, and dogfood-ai-report-from-source explicitly reasoned "No website issue (consumer default unchanged)". A fail-fast fix for a pathological error path — no new command, flag, capability, or install-flow change, normal workflows unaffected — does not trigger the gate. Curator reviews of fix-typed PRs should instead verify in-repo coverage (a README mention plus a docs/reference/cli/<command>.md section for every command whose behavior changed) and cite this precedent rather than requesting a website issue.
