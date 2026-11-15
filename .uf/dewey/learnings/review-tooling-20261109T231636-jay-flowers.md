---
tag: review-tooling
author: jay-flowers
category: gotcha
created_at: 2026-11-09T23:16:36Z
identity: review-tooling-20261109T231636-jay-flowers
tier: draft
---

In this review environment, repo-wide literal-string greps can silently omit matches from very large files: grep for the literals "GAZE_COVERAGE_RUN" and "TestRunSelfCheck" returned zero hits in cmd/gaze/main_test.go (5,282 lines) while alternation regexes ("GAZE|Getenv", "SelfCheck|selfCheck|self-check") over the same file returned the expected matches (e.g., the recursion-guard skips at main_test.go:2483 and 2505). When a grep result would drive a review conclusion — especially a negative existence claim 
