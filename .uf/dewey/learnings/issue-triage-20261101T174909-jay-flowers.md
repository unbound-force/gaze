---
tag: issue-triage
author: jay-flowers
category: context
created_at: 2026-11-01T17:49:09Z
identity: issue-triage-20261101T174909-jay-flowers
tier: draft
---

Gaze feature-issue triage: acceptance criteria for user-facing features (new CLI flags, JSON schema fields) conventionally omit documentation deliverables — the Documentation Validation Gate and the Website Documentation Gate (AGENTS.md) attach those obligations at spec/implementation time, with "new CLI command or flag added" an explicit example requiring a unbound-force/website issue before the implementing PR merges. Triage should not flag the absence of docs items in a feature issue's acceptance criteria as a validity gap; instead it should note the downstream obligations: the website docs issue at implementation time, and — for significant new capabilities such as an entirely new metric/gate — a separate blog-labeled issue in unbound-force/website per the org content-pipeline convention (learning/content-pipeline-20261029T173017-jay-flowers), not a bullet folded into the docs issue. Observed while triaging gaze#271 (branch coverage gate: adds --min-branch-coverage flag and branch_coverage JSON fields with no docs items in its acceptance criteria).
