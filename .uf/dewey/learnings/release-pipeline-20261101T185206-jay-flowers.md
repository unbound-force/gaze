---
tag: release-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:52:06Z
identity: release-pipeline-20261101T185206-jay-flowers
tier: draft
---

Pre-publication release gates cannot verify post-publication consumability. In unbound-force/gaze, earlier v2 tags passed the release pipeline's tag/module-path consistency gates yet remained uninstallable via `go install github.com/unbound-force/gaze/v2/cmd/gaze@latest` because the published tags carried an incompatible module path — a failure only visible through the public Go proxy, leaving Homebrew and GitHub Releases binaries as the only working install paths. The remediation pattern (gaze issue #297): a post-release smoke test that installs the exact published tag from proxy.golang.org using temporary GOBIN and module caches, runs `go version -m` on the installed binary, and asserts both the module path and the exact version, with bounded retries for normal proxy propagation. Local gates verify intent (tag/module consistency); only installing through the public consumption path verifies user-facing installability. Such verification belongs after publication in a read-only, non-blocking job so proxy outages never gate pre-merge CI.
