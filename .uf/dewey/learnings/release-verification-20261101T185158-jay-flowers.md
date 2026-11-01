---
tag: release-verification
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:51:58Z
identity: release-verification-20261101T185158-jay-flowers
tier: draft
---

Pre-publication release gates verify preconditions, not outcomes. Gaze's release pipeline validates tag format, tag/module-path consistency (validate-module-major), and the workflow floor (validate-workflow-floor) before publishing — yet the repo's documented v2 history shows a tag can pass every local gate and still be uninstallable via `go install` because it was cut from an incompatible module path. The architectural complement is post-release outcome verification: install the exact published tag from proxy.golang.org with a temporary GOBIN/module cache, run `go version -m` on the installed binary, assert both the module path and the exact version, and retry within a bounded window for proxy propagation. Precondition gates and post-publication consumption tests are orthogonal defense-in-depth layers, not duplicates — when triaging release-pipeline proposals, check for the outcome-verification layer (the SRE rubric's "smoke test or post-release verification step" question) before judging such a request redundant with the pre-flight gates (gaze issue #297).
