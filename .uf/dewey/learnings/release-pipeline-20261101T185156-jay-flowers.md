---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:51:56Z
identity: release-pipeline-20261101T185156-jay-flowers
tier: draft
---

Pre-publication release gates do not guarantee post-publication installability. The gaze repo's release.yml validates tag/module-path consistency offline before publishing (validate-module-major, validate-workflow-floor, complytime preflight), yet earlier v2 tags were published with an incompatible module path and `go install github.com/unbound-force/gaze/v2/cmd/gaze@latest` fails against them — a failure class the pre-flight gates could not catch. The two check types are complementary, not substitutes: pre-publication checks validate repo/tag state; post-publication checks validate consumer-visible availability (public proxy propagation, module path resolvable by the Go toolchain). A lightweight post-release smoke test closes the loop: install the exact published tag from proxy.golang.org using temporary GOBIN and module caches, run `go version -m` on the installed binary to confirm both module path and exact version, and retry briefly for proxy propagation (proxy.golang.org is eventually consistent). Read-only permissions suffice because no private resources are touched, and confining the network call to a post-publication workflow preserves pre-merge CI hermeticity. Discovered while triaging unbound-force/gaze#297.
