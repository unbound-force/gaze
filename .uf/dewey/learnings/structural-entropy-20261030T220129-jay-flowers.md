---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-10-30T22:01:29Z
identity: structural-entropy-20261030T220129-jay-flowers
tier: draft
---

Fork-based clones break the entropy divisor's default base resolution: `git merge-base HEAD origin/main` returns the fork's (possibly behind) main tip, not the PR's true upstream base — and `git fetch origin main` cannot fix it, because origin IS the fork. Symptom on gaze PR #305 (2026-10-30): local `main` and `origin/main` were both stale at cbe37cc (jflowers/gaze fork) while the PR's actual base was fa2212e on upstream main; diffing against the stale merge-base would have silently widened the scope and misattributed the intervening divisor-gaze-training change's internal/scaffold ΔCe +1 to this PR. Recovery protocol: when merge-base HEAD origin/main disagrees with input_context.base_sha, verify the caller-provided base_sha locally instead of trusting local branch refs — `git rev-parse --verify --end-of-options "<sha>^{commit}"` (object exists) plus `git merge-base HEAD <sha>` returning the sha itself proves it is the branch's fork point and the correct three-dot base — then build the detached base worktree from that SHA. A 40-hex base_sha is intrinsically safe to interpolate (no metacharacters, cannot parse as an option); a diverged local `main` is not.
