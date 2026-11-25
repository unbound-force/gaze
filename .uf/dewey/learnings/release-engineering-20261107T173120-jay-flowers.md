---
tag: release-engineering
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:31:20Z
identity: release-engineering-20261107T173120-jay-flowers
tier: draft
---

When stamping build identity in CI outside GoReleaser, mirror GoReleaser's ldflags symbols exactly and completely — partial stamping produces misleading identity. Gaze's .goreleaser.yaml stamps -X main.version={{.Tag}}, -X main.commit={{.Commit}}, -X main.date={{.CommitDate}}; a CI source build that stamped only main.commit produced a binary reporting version "dev", so the published report header failed to name the producing commit (caught in review of gaze PR #317, fixed in the stamp-full-build-identity co
