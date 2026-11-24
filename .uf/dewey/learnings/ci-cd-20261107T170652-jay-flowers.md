---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:06:52Z
identity: ci-cd-20261107T170652-jay-flowers
tier: draft
---

When a CI source-build path stamps build-identity ldflags onto a binary, stamp every symbol the artifact pipeline consumes, matching the release pipeline's symbol set — partial stamping makes the log line look right while published artifact metadata keeps compile-time defaults. Verified on gaze PR 317 (dogfood-ai-report-from-source): cmd/gaze/main.go declares version/commit/date (defaults dev/none/unknown) and .goreleaser.yaml stamps all three (-X main.version/-X main.commit/-X main.date); the source-install step stamped only -X main.commit, so `gaze --version` named the commit in the job log while the published step-summary header read "Gaze Version: dev" — ReportMetadata.GazeVersion is the artifact's only build-identity field (no commit field) and reads the unstamped main.version. Any alternate build path for a binary must replicate the release pipeline's full stamping set.
