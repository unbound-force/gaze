---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-16T19:35:12Z
identity: test-quality-20261116T193512-jay-flowers
tier: draft
as_of: "gaze@5f11936038eeb8a4537c3b74dbcce858972ae567"
---

Complementary-pin pattern for behavior flips (demonstrated in gaze PR #356, internal/scaffold): when a flag flip converts a test's positive assertion into a negative one ("the --force hint must NOT print on a plain re-run"), the negative assertion can pass vacuously if the underlying hint code is later removed entirely. Pair it with a positive pin of the behavior's only surviving trigger in a different scenario — here ci_test.go's TestRun_CI_SkipIfPresent pins that the CI-workflow skip still prints "1 file skipped (use --force to overwrite)". Both directions of the conditional stay locked: removal of the hint fails the positive pin; regression of the ownership classification fails the negative assertion.
