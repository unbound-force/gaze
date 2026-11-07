---
tag: build-identity
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:06:57Z
identity: build-identity-20261107T170657-jay-flowers
tier: draft
---

Partial build-identity stamping loses provenance in downstream artifacts that read a different symbol than the one stamped. cmd/gaze/main.go declares three ldflags symbols (version/commit/date, defaults dev/none/unknown) and .goreleaser.yaml stamps all three, but gaze's CI source-install step stamped only -X main.commit: the job log's `gaze --version` named the commit (the step's stated goal), while `gaze report` metadata — ReportMetadata.GazeVersion, populated by buildReportMetadata's fallback to the unstamped main.version — published gaze_version "dev" into the GitHub Step Summary, so the published artifact could not be attributed to the commit that produced it. Rule: before deciding a partial -ldflags -X stamp suffices, enumerate every consumer of every build-identity symbol (version template, report metadata, scaffold/version-resolution gates); the release pipeline's ldflags set is the completeness baseline. Verified in gaze PR 317 (dogfood-ai-report-from-source): the fix stamps -X main.version alongside -X main.commit in the workflow's source-install step; sanitizeHeaderValue passes hex SHAs verbatim and no report-path consumer parses the value as semver, so non-semver stamp values are render-safe.
