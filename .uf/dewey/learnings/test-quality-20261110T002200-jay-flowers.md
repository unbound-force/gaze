---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:22:00Z
identity: test-quality-20261110T002200-jay-flowers
tier: draft
---

Environment-variable guards (e.g., `if os.Getenv(VAR) != "" { return ErrSentinel }`) need fast unit tests for BOTH directions, and the negative-direction canary must explicitly neutralize the variable with t.Setenv(VAR, "") instead of relying on the inherited environment. The nested/reentrant runs the guard exists to stop carry the variable SET — an E2E job that spawns the full test suite under the guard's variable will execute the canary there — so an untouched "leave it unset" canary fails exactly when the guard works correctly, turning the fix into a CI breakage. Empty string is the faithful unset simulation for any `!= ""` guard condition. Pair with t.Setenv("PATH", "") so the guarded subprocess spawn fails instantly with a distinguishable exec error: asserting err != nil AND !errors.Is(err, sentinel) proves execution proceeded past the guard without spawning anything (on gaze's proceed-then-fail path, recoverPartialProfile even removes the 0-byte temp profile, keeping the canary hermetic). Pattern location: internal/provider/goprovider/coverage_test.go — TestGenerateCoverProfile_NestedRunRefused (refusal) beside the recommended TestGenerateCoverProfile_ProceedsWhenUnset (canary).
