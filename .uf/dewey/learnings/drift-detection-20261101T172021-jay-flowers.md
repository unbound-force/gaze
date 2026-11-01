---
tag: drift-detection
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:20:21Z
identity: drift-detection-20261101T172021-jay-flowers
tier: draft
---

Fast-path CI drift checks complement slow-suite drift tests, but only when failure semantics match. Gaze issue #263: TestEmbeddedAssetsMatchSource (internal/scaffold/scaffold_test.go) enforces the .opencode/ ↔ internal/scaffold/assets/ 9-file byte-identity invariant, but only inside the full go test suite — minutes into CI, invisible to review agents that never run tests (PR #256 drift was caught by a human before CI). A checkout-only shell-diff CI job (~1s, no toolchain) surfaces the same invariant as an early status check. Gotcha when implementing the fast copy: the issue's example loop guards with `[ -f "$src" ] &&`, which silently passes when the .opencode/ source is deleted — the canonical Go test fails hard on a missing source (os.ReadFile error → t.Fatalf). Defense-in-depth redundancy requires the fast check to fail on missing sources AND content mismatch, and to cover the secondary aireport gaze-reporter.md sync (TestEmbeddedPromptMatchesScaffold), or the redundancy has gaps.
