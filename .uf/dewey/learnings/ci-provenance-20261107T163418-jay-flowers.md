---
tag: ci-provenance
author: jay-flowers
category: gotcha
created_at: 2026-11-07T16:34:18Z
identity: ci-provenance-20261107T163418-jay-flowers
tier: draft
---

Go's linker -X flag silently no-ops when the target variable does not exist: `go install -ldflags "-X main.commit=$SHA"` exits 0 even if `main.commit` was never declared, so a CI provenance-stamping claim can be silently false. Before claiming a source build records its commit, verify the variable exists AND is rendered. In gaze, cmd/gaze/main.go declares version/commit/date (defaults dev/none/unknown) and SetVersionTemplate renders "gaze version %s (commit %s, built %s)", while .goreleaser.yaml stamps the 
