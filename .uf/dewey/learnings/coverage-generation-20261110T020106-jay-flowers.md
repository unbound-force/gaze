---
tag: coverage-generation
author: jay-flowers
category: decision
created_at: 2026-11-10T02:01:06Z
identity: coverage-generation-20261110T020106-jay-flowers
tier: draft
---

Gaze's nested-coverage recursion guard (GAZE_COVERAGE_RUN) deliberately repeats the literal env-var name at test sites instead of importing a shared constant. The literal matches the canonical idiom documented in AGENTS.md (Test Suites section), and the external env-var contract stays independently pinned by literal assertions in TestBuildGoTestCmd_EnvVar and TestBuildGoTestCmd_EnvVarDeduplication (internal/provider/goprovider/coverage_test.go:162,181,192,198), so renaming the production coverageRunEnv constant fails loudly regardless of whether test guards reference literals or a shared constant. Decision recorded on PR #326 (divisor-architect feedback item 7, LOW dry finding): keep the literal at the current sites (cmd/gaze/main_test.go:2483,2505,3430; internal/aireport/runner_steps_test.go:30,110; goprovider guard tests); export goprovider.CoverageRunEnv or extract a shared skip helper only when a sixth guard site appears — at that point rename detection is still guaranteed by the buildGoTestCmd literal-pin tests, so centralization is safe.
