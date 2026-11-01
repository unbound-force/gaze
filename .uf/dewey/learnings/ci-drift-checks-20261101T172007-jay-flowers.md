---
tag: ci-drift-checks
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:20:07Z
identity: ci-drift-checks-20261101T172007-jay-flowers
tier: draft
---

Glob-based file-sync drift checks pass silently on deletion: a loop over `internal/scaffold/assets/**/*.md` only compares pairs where the embedded file still exists, so deleting an embedded asset (or its source counterpart) escapes detection entirely — and bash's `**` requires `shopt -s globstar` to recurse at all. Deletion-safe drift checks iterate an explicit manifest of expected pairs (gaze: the 9-file .opencode/ ↔ internal/scaffold/assets/ list enumerated by TestAssetPaths_Returns9Files, plus the internal/aireport gaze-reporter copy) and fail when any listed file is absent on either side. General principle: when triaging or reviewing a proposed fast CI check that front-runs a test-suite invariant, verify it covers the same failure surface as the test it accelerates — drift in both directions, including deletion — not just the subset the example script happens to compare.
