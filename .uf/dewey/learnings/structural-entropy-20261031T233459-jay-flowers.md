---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-10-31T23:34:59Z
identity: structural-entropy-20261031T233459-jay-flowers
tier: draft
---

Code-mode entropy review of gaze init-ci-scaffold (opsx/init-ci-scaffold@71d1065 vs upstream/main@fa87795, committed branch): the measured internal/scaffold ΔCe was +2 (Ce 11→13), not the +1 (11→12) recorded in design D4 and carried into the specs-mode learning — design coupling predictions that enumerate only promoted/external dependencies undercount when the same change also introduces first-time STDLIB import edges (here regexp for the two validation patterns); stdlib edges count toward package Ce identically, and the measured ΔCe is the ground truth that exposes the undercount. Measured ΔInstability +0.0119 (~13× below the 0.15 gate) with Distance improving −0.0119 (main-sequence zone on both sides), all 22 other packages zero-delta, no cycles, vibe-check diff verdict COMMENT ("materiality: non-zero metric shifts below REQUEST_CHANGES thresholds") mapping to APPROVE WITH ADVISORIES with one LOW design-precision advisory. Committed-branch protocol that produced the clean measurement: validate the caller base ref in order (regex gate → git check-ref-format → git rev-parse --verify --end-of-options), confirm rev-parse of the base ref equals input_context.base_sha exactly, confirm git merge-base HEAD <baseSHA> returns the baseSHA itself (proves the fork point and guards the stale-fork failure), measure the base in a detached git worktree outside the repo tree and the PR as the clean working tree with the same vibe-check binary on both sides, then git worktree remove --force unconditionally.
