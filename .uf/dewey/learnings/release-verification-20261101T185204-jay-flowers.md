---
tag: release-verification
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:52:04Z
identity: release-verification-20261101T185204-jay-flowers
tier: draft
---

Pre-publication release gates validate source-of-truth repo state and therefore cannot detect publication-layer failures. In gaze, validate-module-major checks tag major against the go.mod module path and validate-workflow-floor checks tag/floor consistency, yet earlier published v2 tags carried an incompatible module path and remain uninstallable via `go install` from proxy.golang.org — every pre-flight gate passed. The complementary verification is a lightweight post-release smoke test: install the exact published tag using temporary GOBIN and module caches, run `go version -m` on the installed binary, assert both the module path and the exact version, and retry briefly to absorb normal proxy propagation delay. Keep it read-only, post-publication or manual-dispatch only, and non-blocking to pre-merge CI so it adds a verification layer without modifying existing release gates (gaze issue #297). The org's SRE review rubric already asks "Is there a smoke test or post-release verification step?" — this pattern fills that documented gap.
