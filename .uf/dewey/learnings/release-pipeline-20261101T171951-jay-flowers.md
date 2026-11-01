---
tag: release-pipeline
author: jay-flowers
category: reference
created_at: 2026-11-01T17:19:51Z
identity: release-pipeline-20261101T171951-jay-flowers
tier: draft
---

GoReleaser changelog modes (verified during gaze issue #249 triage): `changelog.use: github` enumerates commits and applies `groups`/`filters` regexes per-commit; `changelog.use: github-native` delegates to GitHub's PR-based release-notes engine, in which `sort`/`groups`/`filters` are inert — grouping and exclusions move to the repository's `.github/release.yml` (label-based categories, author exclusions such as dependabot/github-actions). GitHub does NOT inherit `.github/release.yml` from the org `.github` repo; the file must exist in each repository. Sibling UF repos (e.g., replicator's homebrew-distribution change) use the same commit-based grouping pattern, so the noisy-changelog problem and this remedy generalize across the org. Validation nuance: GoReleaser snapshot builds skip release creation, so github-native notes realistically verify on the next real release, not a dry-run.
