---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:15Z
identity: structural-entropy-20261101T001215-jay-flowers
tier: draft
---

Fourth consecutive gaze entropy confirmation, first at a final PR head after a doc-only finale sweep: re-measuring opsx/init-ci-scaffold at PR head b3dc2c9 (the council had reviewed earlier head 71d1065) reproduced the earlier code-mode measurement exactly — internal/scaffold Ca 1→1, Ce 11→13 (Δ+2: x/mod/semver promotion plus first-time stdlib regexp edge), ΔInstability +0.0119 (~13× below the 0.15 gate), Distance improving −0.0119 with main-sequence zone on both sides, LCOM 0, 21 other packages zero-delta, no cycles, vibe-check diff COMMENT (sub-threshold materiality) mapping to APPROVE WITH ADVISORIES. A finale commit that sweeps only doc/bookkeeping files leaves Go package metrics identical, so re-measuring at the pushed PR head is cheap insurance, not redundant work. Protocol extension for post-finale dispatches: when the repo checkout is no longer on the PR branch (return-to-main after /uf.finale), measure BOTH sides in detached worktrees at the pinned base/head SHAs — the entropy agent's bash allowlist cannot audit working-tree dirtiness (no git status), so worktree-at-SHA is the only deterministic guarantee that each side measures exactly its pinned ref; before building worktrees, corroborate the base with rev-parse of the ref equal to input_context.base_sha and git merge-base <headSHA> <baseSHA> returning the baseSHA itself.
