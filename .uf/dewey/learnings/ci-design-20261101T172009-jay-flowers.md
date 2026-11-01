---
tag: ci-design
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:20:09Z
identity: ci-design-20261101T172009-jay-flowers
tier: draft
---

When adding a fast CI drift check that mirrors an invariant already enforced by a test suite, derive the file manifest from its single source of truth instead of hardcoding it a third time. In unbound-force/gaze, the 9-file .opencode/ ↔ internal/scaffold/assets/ manifest already lives in TestAssetPaths_Returns9Files and AGENTS.md; a CI job that restates the list adds a new sync surface. Repo precedent: .github/scripts/validate-workflow-floor.sh greps internal/scaffold/scaffold.go for the minWorkflowTag floor ("single source of truth, value never duplicated"). Shell-sketch gotchas to design around: '**' globs require globstar (off in the default GitHub Actions bash shell), assets/**/*.jsonc misses root-level assets/dcp.jsonc without it, a '[ -f "$src" ]' existence guard silently skips missing counterparts (turning manifest shrinkage into a false negative), and the internal/scaffold/assets/ci/ subtree must be excluded explicitly because it deliberately has no .opencode/ counterpart.
