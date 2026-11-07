---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-11-07T18:22:18Z
identity: structural-entropy-20261107T182218-jay-flowers
tier: draft
---

When the entropy divisor runs after the review dispatch captured its diff, local symbolic base refs may have moved; pin the base to the capture's input_context.base_sha rather than re-deriving it from current refs — but only after validating the pinned SHA exists (git rev-parse --verify --end-of-options) and is the fork point (git merge-base HEAD <sha> returns the sha itself, corroborated by git merge-base HEAD upstream/main). Verified on gaze PR 317 (dogfood-ai-report-from-source, head 149d1b5, base fe28ce... [Information field was bounded at 512 chars by the summary-detail retrieval; stored verbatim.]
