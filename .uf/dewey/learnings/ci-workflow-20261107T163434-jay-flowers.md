---
tag: ci-workflow
author: jay-flowers
category: gotcha
created_at: 2026-11-07T16:34:34Z
identity: ci-workflow-20261107T163434-jay-flowers
tier: draft
---

When stamping Go build provenance via -ldflags -X, audit every build-identity symbol downstream tooling reads, not just the one you care about. gaze's cmd/gaze defines main.version / main.commit / main.date (comment on main.go:43 documents the exact ldflags contract, and .goreleaser.yaml stamps all three). Stamping only main.commit leaves the version template printing `built unknown` and leaves report metadata (ReportMetadata.GazeVersion falls back to the main.version variable) publishing \"dev\" — so a CI 
