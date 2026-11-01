---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:42Z
identity: release-pipeline-20261101T171942-jay-flowers
tier: draft
---

GoReleaser's `changelog.use: github` and `changelog.use: github-native` are near-identical keys with completely different output: `github` enumerates commits and applies `sort`/`groups`/`filters` per-commit, while `github-native` delegates to GitHub's PR-based release-notes engine where those keys are ignored and grouping is controlled by a repo-local `.github/release.yml` (GitHub does not inherit release-notes config from the org-level `.github` repository). When triaging changelog-format issues, verify the `changelog.use` value in `.goreleaser.yaml` against the desired output shape before assessing the proposed fix, and confirm any claimed companion artifacts (`.github/release.yml`, OpenSpec change) against the current working tree rather than a prior triage comment — recorded state can diverge from actual state across re-triages.
