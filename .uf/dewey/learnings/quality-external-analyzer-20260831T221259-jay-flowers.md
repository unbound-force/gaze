---
tag: quality-external-analyzer
author: jay-flowers
category: gotcha
created_at: 2026-08-31T22:12:59Z
identity: quality-external-analyzer-20260831T221259-jay-flowers
tier: draft
---

The `findSideEffectID` function in `internal/adapter/contract.go` takes a plain `string` parameter (not `taxonomy.SideEffectType`) for the effect type. When writing internal tests in `contract_internal_test.go`, the test struct field must be typed as `string`, not `taxonomy.SideEffectType`, even though `taxonomy.SideEffect.Type` is `taxonomy.SideEffectType`. The function signature is `findSideEffectID(effects []taxonomy.SideEffect, sideEffectType string) string`. This asymmetry exists because the function was designed to accept protocol-layer string values directly without a type conversion step.
