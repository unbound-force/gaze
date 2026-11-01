---
tag: test-coverage
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:18Z
identity: test-coverage-20261101T001218-jay-flowers
tier: draft
---

When a design decision documents a failure-ordering contract ("write X after Y so Y completes even when X fails"), the accompanying test matrix needs a forced-failure case for X, not only success paths for X and Y. Success-only coverage leaves two silent regression modes undetected: swallowing X's error (partial work reported as success — the CI-gate-integrity bug class, gaze issues #101/#108/#116) and reordering X before Y (breaking the documented contract). The forced-failure test is cheap when the package already has failure-injection precedent: gaze's internal/scaffold/ufconfig_test.go forces stat/MkdirAll failures by creating `.uf` as a regular file and makes files unreadable via chmod 0o444 with a deferred restore; the same pattern applies to any new write path (e.g. create `.github` as a regular file so the CI workflow's MkdirAll fails, then assert the error propagates AND the earlier .opencode/ writes are on disk). Observed on the init-ci-scaffold PR (#308): all 14 new ci_test.go functions covered success/skip/fail-closed-version paths, but the documented "core scaffolding completes even when the CI write fails" contract had zero coverage.
