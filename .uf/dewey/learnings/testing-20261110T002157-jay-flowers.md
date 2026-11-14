---
tag: testing
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:21:57Z
identity: testing-20261110T002157-jay-flowers
tier: draft
---

When adding a fail-closed environment-variable guard (refuse an operation when the variable is set), unit-test both directions as a pair. The firing direction (variable set → assert the sentinel error) cannot catch an over-firing guard — a condition broadened to always-true still returns the sentinel when the variable is set. Add a canary for the pass-through: force the unset-equivalent state with t.Setenv(VAR, "") (a != "" guard treats empty identically to unset), clear PATH via t.Setenv("PATH", "") so the guarded operation fails fast without a real subprocess, and assert the error is non-nil and NOT the guard sentinel — any non-sentinel error proves execution passed the guard. Never rely on the ambient variable being unset: inside nested or self-referential test runs the variable is set by design, and an ambient-unset canary would fail exactly there. Verified on gaze PR #326 (goprovider GAZE_COVERAGE_RUN guard): the -short unit lane had zero coverage of the pass-through; only the 30-minute E2E self-check exercised it.
