---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:34:24Z
identity: content-pipeline-20261107T163424-jay-flowers
tier: draft
---

Curator code-mode reviews should run a deferral-traceability check alongside the website-gate verification: when a PR's change record (proposal out-of-scope, design decisions, AGENTS.md Recent Changes) defers work to an unnamed "follow-up change", search the repo's open issues for the deferred work before treating the deferral as tracked. Precedent: gaze PR #317 (dogfood-ai-report-from-source) deferred the reusable workflow's gaze-version bump in three places while open issue #313 — filed earlier by spec 04
