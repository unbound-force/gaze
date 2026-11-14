---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:22:03Z
identity: test-quality-20261110T002203-jay-flowers
tier: draft
---

Environment-variable guards need both truth-table directions in the fast test lane, and the negative-direction canary must pin the variable explicitly. A refusal test (var set → sentinel error) cannot detect a guard that fires unconditionally or reads the wrong variable — both regressions still return the sentinel when the test sets the variable, so the fast suite stays green and the break surfaces only in the slow E2E lane. Fast canary pattern: t.Setenv(var, "") + t.Setenv("PATH", "") → call the guarded function → assert the error is non-nil AND NOT errors.Is(err, sentinel); the cleared PATH makes the call fail in milliseconds without spawning a subprocess. The empty-string pin (not "leave unset") is required because a nested run spawned by an E2E self-check re-executes the same tests with the variable inherited as set — a canary relying on inheritance fails inside that nested run and breaks the self-check that spawned it. Applied in gaze PR #326, fix-nested-coverage-recursion (goprovider ErrNestedCoverageRun guard).
