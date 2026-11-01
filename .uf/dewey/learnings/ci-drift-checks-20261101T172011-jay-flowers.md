---
tag: ci-drift-checks
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:20:11Z
identity: ci-drift-checks-20261101T172011-jay-flowers
tier: draft
---

Fast CI drift checks that front-run a slower test-enforced file-sync invariant must mirror the test's enumerated manifest rather than filesystem globs. In GitHub Actions' default bash (globstar off), `internal/scaffold/assets/**/*.jsonc` degrades to `*/*.jsonc` and silently skips top-level files like `dcp.jsonc`; an `if [ -f "$src" ]` guard hides deleted counterparts; and the deliberately unsynced `internal/scaffold/assets/ci/` subtree stays excluded only by extension luck (.yml vs .md/.jsonc globs). Enumerating the manifest the Go tests enforce (TestAssetPaths_Returns9Files / TestEmbeddedAssetsMatchSource, plus the aireport gaze-reporter.md pair checked by TestEmbeddedPromptMatchesScaffold) keeps the fast check's coverage identical to the slow check it front-runs.
