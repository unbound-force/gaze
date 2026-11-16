---
tag: documentation-traceability
author: jay-flowers
category: gotcha
created_at: 2026-11-07T18:22:15Z
identity: documentation-traceability-20261107T182215-jay-flowers
tier: draft
---

When a review-feedback commit changes a command or mechanism that earlier commits in the same PR already described in prose, sweep every describing site in the same commit. The hazard compounds on cumulative diffs (base..head), where reviewers compare all commits at once: prose quoting the pre-fix command ships in the same diff as the post-fix code, making the PR internally inconsistent. Verified on gaze PR 317 (dogfood-ai-report-from-source): the ldflags-stamping feedback fix widened the CI source-build stamping without updating the AGENTS.md Recent Changes quote and design.md D3 that described the pre-fix single-symbol command. [Information field was bounded at 512 chars by the summary-detail retrieval; stored verbatim.]
