---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:06:50Z
identity: ci-cd-20261107T170650-jay-flowers
tier: draft
---

When a CI job builds a Go CLI from source to dogfood a commit, stamp every build-identity ldflags symbol the release pipeline stamps, not just the commit. Verified on gaze PR 317 (dogfood-ai-report-from-source): cmd/gaze/main.go declares version/commit/date with defaults dev/none/unknown; `gaze report` metadata falls back to main.version when no --gaze-version override is passed (buildReportMetadata, main.go:2427-2428), and ReportMetadata has no commit field, so the published step-summary header's only producer-identity value is main.version. A source build stamping only -X main.commit therefore publishes reports self-identifying as "dev" — commit provenance appears in the job log (`gaze --version`) but not in the published artifact. Mirror the release pipeline's stamping (.goreleaser.yaml: -X main.version -X main.commit -X main.date) in source-build CI jobs.
