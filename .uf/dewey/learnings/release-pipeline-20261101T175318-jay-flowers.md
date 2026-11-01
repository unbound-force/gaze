---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:53:18Z
identity: release-pipeline-20261101T175318-jay-flowers
tier: draft
---

In gaze's release pipeline, the reusable preflight job (complytime/org-infra reusable_release_preflight.yml, invoked with contents: write) creates the annotated git tag via the GitHub API before the GoReleaser release, sign-macos, and cask-publication jobs run. A release-authorization gate must therefore be ordered before preflight, not merely before the release/signing/cask jobs — an unauthorized run blocked only at publication would still have created a tag. When designing fail-closed release authorization, enumerate every mutating step in the pipeline (tag creation, release creation, asset upload/clobber, signing, tap push) and gate the earliest one. Issue #267 (harden official macOS release authorization) specified exactly this ordering.
