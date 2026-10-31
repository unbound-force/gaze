---
tag: init-ci-scaffold
author: jay-flowers
category: gotcha
created_at: 2026-10-31T19:21:35Z
identity: init-ci-scaffold-20261031T192135-jay-flowers
tier: draft
---

OpenSpec change init-ci-scaffold (gaze, issue #306): task 1.1 adds internal/scaffold/assets/ci/gaze-ai-report.yml into the embed.FS tree that Run's fs.WalkDir and assetPaths() enumerate, but the `ci/` prefix skip that excludes it from the .opencode/ mirror is task 1.4's first clause. Between 1.1 and 1.4 the internal/scaffold suite is therefore red by design: 10 tests fail with "expected 9 ... got 10" counts (.opencode/ci/gaze-ai-report.yml appears in every list), including TestEmbeddedAssetsMatchSource and TestAssetPaths_Returns9Files. When implementing mid-sequence tasks (1.2, 1.3), prove no NEW regressions with a single-file git-stash experiment (git stash push -- internal/scaffold/scaffold.go; run suite; git stash pop) rather than assuming the tree was green — the failure sets are byte-identical with and without 1.3's changes. The tree goes green when task 1.4 lands the walk skip.
