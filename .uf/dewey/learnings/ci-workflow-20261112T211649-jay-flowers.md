---
tag: ci-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-12T21:16:49Z
identity: ci-workflow-20261112T211649-jay-flowers
tier: draft
---

A CI canary that mirrors a consumer step living in another workflow creates a false-assurance risk if nothing detects command-shape drift: a future change to the canonical step can leave the canary green against a shape no consumer runs. The correct guard is a command-shape invariant test (not byte-identical — the two steps are deliberately different in shape: one resolves the tag from GitHub Releases, the other takes a workflow input and is gated by a boolean), wired into test.yml in the style of .github/scripts/test-validate-workflow-floor.sh. Two important scoping rules learned from unbound-force/gaze#331: (1) with only two occurrences the repo's DRY MUST (default.md CS-004, >2 locations) does not trigger, and the relevant rule is the SHOULD-level drift-detection convention (go.md TC-007); (2) adding the test is itself a CI-configuration change, so under Spec-First it belongs in its own spec'd change, not bolted onto the canary PR. Prior learnings (learning/release-verification-20261101T185210, learning/ci-testing-20261101T175311) confirm static workflow-configuration tests are the established complement to end-to-end post-publication checks.
