---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:21Z
identity: issue-triage-20261101T175321-jay-flowers
tier: draft
---

Triage pattern for release-pipeline hardening issues (from gaze #267): (1) A documented graceful-degradation fallback is objective evidence that a hardening issue targets real current behavior — AGENTS.md records "when macOS signing secrets are unavailable (e.g., in forks), the unsigned cask is pushed directly to the Homebrew tap", and learning release-pipeline-20260611T201838-jay-flowers confirms that fallback was deliberately added to preserve spec 015-native-macos-signing's graceful-degradation guarantee. Closing such a fallback is a conscious policy reversal of a prior design guarantee — valid, but the implementing spec must explicitly supersede the prior guarantee so the design record stays coherent. (2) Subsumption is not duplication: gaze #99 (move hardcoded signing identity to a secret) is a strict subset of #267 (require all six signing secrets before publication); duplicate detection correctly returns none because one issue subsumes rather than duplicates the other, and the superset implementation should close the subset by reference on merge. (3) Static workflow-configuration test requirements (parse release.yml; assert authorize-job existence, six-secret probing, job dependency ordering) are precedented via the validate-workflow-floor.sh + test-validate-workflow-floor.sh pattern (repo invariants + synthetic matrix + canary invocation, learning release-gates-20261101T164347-jay-flowers) — cite that precedent when judging whether such acceptance criteria are implementable.
