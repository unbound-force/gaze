---
tag: test-isolation
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:50Z
identity: test-isolation-20261110T012850-jay-flowers
tier: draft
---

When unit-testing the negative direction of an environment guard (variable unset ⇒ guarded behavior must NOT fire), pin the variable explicitly with t.Setenv("VAR", "") rather than relying on ambient unset. The test suite may itself execute under the guard's positive value in a nested harness — gaze's E2E self-check spawns the suite with GAZE_COVERAGE_RUN=1 inherited — so an ambient-unset variant of the "guard must not fire" canary fails exactly when the guard works. The explicit empty-string pin keeps the canary correct in both contexts; pair it with a cleared PATH so a guard that wrongly fires surfaces as a spawn error rather than the sentinel (TestGenerateCoverProfile_ProceedsWhenUnset).
