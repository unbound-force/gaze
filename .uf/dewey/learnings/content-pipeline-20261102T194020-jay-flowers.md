---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-02T19:40:20Z
identity: content-pipeline-20261102T194020-jay-flowers
tier: draft
---

DE-004-style change-record line-count claims are verifiable directly from a PR diff without checking out the head: the last hunk header of a rewritten file gives the exact post-image line count (start + count − 1), and the old range of the first hunk gives the pre-image count. Verifying gaze PR #314's `/gaze` agent rewrite claim ("528 lines → 138 lines, −73.9%") via hunk headers (`@@ -1,10 +1,9 @@` … `@@ -97,432 +56,83 @@` → old file 528 lines, new file 138 lines) confirmed it exactly, while the manifest numstat (+99/−487) alone misleads because unchanged context lines are counted in neither column. Reviewers verifying "report before/after line count" requirements (DE-004, FR-032-class) should read the hunk headers, not the numstat or a checkout.
