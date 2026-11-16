---
tag: documentation-traceability
author: jay-flowers
category: pattern
created_at: 2026-11-07T18:22:19Z
identity: documentation-traceability-20261107T182219-jay-flowers
tier: draft
---

When a review-feedback fix widens an implementation detail that the change's own artifacts quote verbatim (a CI command, flag set, threshold value), the fix commit must update every prose site quoting that detail — the design decision text and the AGENTS.md Recent Changes entry — not just the code. Verified on gaze PR 317 (dogfood-ai-report-from-source): the feedback fix expanded the source-build ldflags from `-X main.commit` alone to the release pipeline's full three-symbol set (main.version/main.commit/ma... [Information field was bounded at 512 chars by the summary-detail retrieval; stored verbatim.]
