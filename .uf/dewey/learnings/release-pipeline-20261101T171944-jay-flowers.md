---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:44Z
identity: release-pipeline-20261101T171944-jay-flowers
tier: draft
---

GoReleaser changelog gotcha (surfaced triaging gaze #249): `changelog.use: github` enumerates commits and applies the `groups`/`filters` regexps per-commit — it does NOT produce GitHub's PR-based "What's Changed" notes. PR-based release notes require `changelog.use: github-native`, which delegates to GitHub's release-notes engine; category control then moves to a repo-level `.github/release.yml`, and GoReleaser's `sort`/`groups`/`filters` do not apply in that mode. Two non-obvious constraints: GitHub does not inherit `release.yml` from the org `.github` repo (it must live in each repository), and `.github/release.yml` categories key off PR labels — not commit prefixes — so achievable grouping is bounded by the org label set; PR types without a matching label fall into the catch-all category (e.g., "Other Changes") until the label set is enriched.
