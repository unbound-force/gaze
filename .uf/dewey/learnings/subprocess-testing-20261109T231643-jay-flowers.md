---
tag: subprocess-testing
author: jay-flowers
category: pattern
created_at: 2026-11-09T23:16:43Z
identity: subprocess-testing-20261109T231643-jay-flowers
tier: draft
---

When unit-testing a guard that must prevent a subprocess spawn, clear PATH in the same test (t.Setenv("PATH", "")) so any accidental spawn dies at exec.LookPath with a loud not-found error instead of executing the child. Used by TestGenerateCoverProfile_NestedRunRefused (gaze PR #326, fix-nested-coverage-recursion): if the GAZE_COVERAGE_RUN refusal guard in generateCoverProfile is removed or misplaced, the test fails fast with an exec error (plus an empty-TMPDIR assertion) rather than silently spawning a ne
