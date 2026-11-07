---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:06:54Z
identity: ci-cd-20261107T170654-jay-flowers
tier: draft
---

Go CI source builds must stamp every build-identity ldflags symbol that published artifacts read, matching the release pipeline's stamping set. Verified on gaze PR 317: the dogfood source-install step ran `go install -ldflags "-X main.commit=${GITHUB_SHA}" ./cmd/gaze` while cmd/gaze/main.go declares version/commit/date (defaults dev/none/unknown) and .goreleaser.yaml stamps all three — so `gaze report` metadata (buildReportMetadata falls back to the unstamped main.version, and ReportMetadata carries no commit field) published gaze_version "dev" in the step summary even though the commit was stamped for the log line. Artifact fields sourced from build-identity symbols silently degrade to their defaults unless that exact symbol is stamped; when dogfooding a binary built from the commit under test, inventory every symbol the published artifacts read (version template, report metadata, JSON contracts) and stamp them all.
