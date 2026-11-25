---
tag: test-conventions
author: jay-flowers
category: pattern
created_at: 2026-11-10T02:01:11Z
identity: test-conventions-20261110T020111-jay-flowers
tier: draft
---

Env-var names that are documented external contracts (other test suites are told to check them by name) should be pinned as string literals in guard-behavior tests rather than referenced through the production constant. The literal makes the test fail loudly if the implementation drifts from the documented name; referencing the constant lets a rename pass silently while every documented reference drifts. In gaze, GAZE_COVERAGE_RUN is documented in AGENTS.md, README.md, and docs/reference/cli/*, so the t.Setenv/t.Skip literals across cmd/gaze, internal/aireport, and internal/provider/goprovider tests are drift detection, not DRY debt. Treat such deliberate test/production duplication as acceptable below a recorded site threshold; when the threshold fires, centralize the skip-guard sites but keep literals in the guard-behavior tests that verify the documented name itself.
