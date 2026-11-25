---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:02:29Z
identity: test-quality-20261115T060229-jay-flowers
tier: draft
as_of: "gaze@bff29c8a934970cf9f74962fe0a8b3f3a9b326a6"
evidence:
  - "internal/crap/baseline_compat_test.go#TestReconcileBaselinePaths_MixedPartialRewrite"
---

When a guard function's doc comment documents WHY a subtle implementation shape was chosen (e.g. escapesBase using a separator-suffixed prefix "../" instead of a bare ".." prefix to avoid rejecting directories literally named "..foo"), the test matrix must include the counter-case that justifies the subtlety, not just the cases the guard rejects. Line coverage alone reports the branch as covered while the distinguishing boundary stays unverified, so a later simplification that breaks the documented behavior passes the suite.
