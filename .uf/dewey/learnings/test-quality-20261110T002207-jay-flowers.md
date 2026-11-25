---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:22:07Z
identity: test-quality-20261110T002207-jay-flowers
tier: draft
---

Environment-variable guards need paired direction canaries with a deterministic unset. When a guard refuses an operation on a non-empty env var (gaze's GAZE_COVERAGE_RUN nested-coverage guard, internal/provider/goprovider/coverage.go), the firing direction naturally gets a unit test, but the pass-through direction (variable unset → operation proceeds) often ends up with no fast coverage: every test exercising the real operation tends to be testing.Short()-guarded or E2E-only, so the -short unit suite never executes the guarded branch's fall-through. Verified on gaze PR #326 (fix-nested-coverage-recursion): the guard's negative direction ran only in Short-guarded integration tests and the 20-30 minute E2E self-check job. Pattern: mirror the refusal test — deterministically empty the variable with t.Setenv(name, "") rather than leaving it merely unset (nested coverage children inherit GAZE_COVERAGE_RUN=1, so an unset-based canary fails spuriously inside them; empty string exercises exactly the != "" guard semantics and t.Setenv restores the inherited value afterward), clear PATH so the spawned command fails instantly at exec lookup, and assert the error is non-nil AND NOT errors.Is(err, sentinel). This locks both directions of the guard's contract in milliseconds instead of waiting for an E2E suite.
