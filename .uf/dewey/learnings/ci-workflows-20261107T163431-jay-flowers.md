---
tag: ci-workflows
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:34:31Z
identity: ci-workflows-20261107T163431-jay-flowers
tier: draft
---

Dogfood CI jobs that call a reusable workflow without passing its tool-version input test the input's default pinned release, not the commit under test — unreleased fixes never run in CI and old-release failures recur silently (gaze test.yml run 41698377720: the v2.2.0 default adapter published one line of agent narration to the step summary and the job passed, the exact failure spec 040 fixed but had never exercised in CI). Fix pattern (openspec/changes/dogfood-ai-report-from-source, PR #317): opt-in boole
