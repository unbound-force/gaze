---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-11-09T23:16:39Z
identity: structural-entropy-20261109T231639-jay-flowers
tier: draft
---

Refines the import-union entropy rule from learning/structural-entropy-20261030T201229-jay-flowers: the union that predicts ΔCe is the package's NON-TEST import set only. In PR #326 (fix-nested-coverage-recursion), `errors` already existed in internal/provider/goprovider via contract_internal_test.go, yet adding it to non-test coverage.go still produced ΔCe +1 (Ce 21→22, Instability 0.9130→0.9167, Distance 0.0870→0.0833 — an improvement — verdict COMMENT on materiality, all other 21 packages zero-delta). Te
