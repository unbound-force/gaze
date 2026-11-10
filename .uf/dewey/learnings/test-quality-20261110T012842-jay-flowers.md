---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:42Z
identity: test-quality-20261110T012842-jay-flowers
tier: draft
---

When unit-testing the negative direction of an environment-variable guard in a suite that may itself be spawned WITH that variable set, pin the variable to the empty string with t.Setenv(VAR, "") rather than relying on ambient unset. os.Getenv returns "" for both unset and empty, so the code path is identical — but an ambient-unset test breaks inside a nested run exactly when the guard works. In gaze (PR #326), the E2E self-check spawns the test suite with GAZE_COVERAGE_RUN=1 inherited, so TestGenerateCoverProfile_ProceedsWhenSet pins GAZE_COVERAGE_RUN="" explicitly and documents why in a comment.
