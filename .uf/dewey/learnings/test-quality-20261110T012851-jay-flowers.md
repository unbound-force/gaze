---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:51Z
identity: test-quality-20261110T012851-jay-flowers
tier: draft
---

When unit-testing the negative direction of an environment-variable guard, pin the variable to the empty string with t.Setenv rather than relying on ambient unset. The test binary may itself execute under the variable — e.g., inside a nested `go test -coverprofile` collection spawned by an E2E self-check — and an ambient-unset variant fails exactly when the guard works. The explicit empty-string pin keeps the canary correct in both contexts (verified in internal/provider/goprovider/coverage_test.go TestGenerateCoverProfile_ProceedsWhenUnset).
