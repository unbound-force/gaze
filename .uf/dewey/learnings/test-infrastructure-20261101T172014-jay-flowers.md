---
tag: test-infrastructure
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:20:14Z
identity: test-infrastructure-20261101T172014-jay-flowers
tier: draft
---

Fast CI drift checks for byte-identical file mirrors: when a sync invariant (e.g., embed.FS assets mirrored from a source directory like .opencode/ → internal/scaffold/assets/) is enforced only by a test inside the full compiled suite, review agents that don't execute tests receive no signal until the slow suite finishes, and human reviewers substitute for the missing automated signal. Remedy: a sub-second CI shell-diff job placed before the full test suite. Implement it by mirroring the enumerated file manifest (e.g., the list asserted by TestAssetPaths_Returns9Files) rather than glob patterns — bash ** globstar is off by default in GitHub Actions run steps, so `assets/**/*.jsonc` collapses to a single directory level and misses root-level files like dcp.jsonc, and an existence-guarded diff (`[ -f "$src" ]`) silently passes embedded files that have no source counterpart, so new unmirrored assets go undetected. Validate the job with an intentional-drift test PR asserting a bounded failure time (e.g., fails within 30 seconds) and a remediation message naming the drifted pair and the cp fix, and keep the in-suite test as the authoritative redundant gate (defense in depth). Precedent: unbound-force/gaze issue #263 (TestEmbeddedAssetsMatchSource); script-plus-test wiring pattern: .github/scripts/validate-workflow-floor.sh + test-validate-workflow-floor.sh in test.yml.
