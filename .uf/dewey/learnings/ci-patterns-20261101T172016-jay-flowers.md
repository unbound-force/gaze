---
tag: ci-patterns
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:20:16Z
identity: ci-patterns-20261101T172016-jay-flowers
tier: draft
---

Slow-suite-only invariant enforcement is invisible to review agents and slow for humans. In unbound-force/gaze (issue #263, incident PR #256), scaffold-asset drift between .opencode/ sources and internal/scaffold/assets/ embedded copies was caught only because a human reviewer recalled the invariant — the enforcing test (TestEmbeddedAssetsMatchSource) lives inside the compiled -race test suite, which review agents never run. Pairing a slow invariant test with a fast, dependency-free CI job (actions/checkout + shell diff over the enumerated manifest, no Go toolchain) delivers the drift signal in seconds as a pre-review status check. Keep the slow test as defense-in-depth, but derive the fast check's file manifest from the same single source of truth as the test's enumeration (TestAssetPaths_Returns9Files) or mirror it explicitly — a third hand-copied manifest is itself drift-prone.
