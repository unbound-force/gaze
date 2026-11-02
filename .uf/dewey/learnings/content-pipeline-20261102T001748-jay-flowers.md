---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:48Z
identity: content-pipeline-20261102T001748-jay-flowers
tier: draft
---

gaze repo website-docs gate pattern: any change that alters what `gaze init` writes to consumer projects (scaffold output or `.uf` registration merges) is user-facing under the AGENTS.md Website Documentation Gate — a docs issue in unbound-force/website must exist before the PR merges (website docs issues are unlabeled, title prefixed "docs:"), and the AGENTS.md Recent Changes entry cites the issue number (precedent: unbound-force/website#345 for divisor-gaze-testing, #352 for init-ci-scaffold). Sequencing the filing after the review council (tasks.md 5.4 after 5.3, as gaze-init-triage-advisor did) leaves the mandatory artifact missing at council time: dispatched Curator reviewers are prohibited from creating issues, so the missing gate issue becomes a blocking HIGH finding carrying the full `gh issue create` command — a full council fix cycle for a five-minute task. File the website issue before or alongside the council dispatch. When checking for duplicates, verify candidates with `gh issue view`: an open issue in the same feature area (e.g., #345, which covers the prior three-mode registration) is not a match for a change that supersedes part of it.
