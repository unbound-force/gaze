---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-10-31T15:11:06Z
identity: structural-entropy-20261031T151106-jay-flowers
tier: draft
---

Third consecutive confirmation of the specs-mode entropy protocol on gaze OpenSpec diffs (init-ci-scaffold, after divisor-gaze-testing and redact-env-values-from-adapter-stderr): with head_sha == base_sha and uncommitted:true, measure the base in a detached git worktree at the validated shared SHA (regex gate → check-ref-format → rev-parse --verify --end-of-options; verify upstream/main resolves to exactly input_context.base_sha to rule out fork staleness) and the PR as the repo working tree — Markdown-only diffs yield all-zero package deltas across all six metrics, no cycles, direction stable, vibe-check diff APPROVE. The design-claim advisory trigger generalizes beyond false "no new imports" claims: it fires whenever the planned dependency is absent from the implementing package's current import set, even when the design honestly documents the go.mod promotion — here golang.org/x/mod/semver is imported by no package in the module (go.mod carries it indirect), so internal/scaffold gains its first x/mod edge (measured base Ce 11 → 12) despite design D4's accurate module-level "no new supply-chain surface" statement; the omitted fact is the package-level Ce consequence, not the module-level one. Quantify such advisories with the measured base Ce plus the prior measured ΔInstability precedent for the same package (+0.0076 per edge, ~20× below the 0.15 gate) rather than fresh in-prompt arithmetic.
