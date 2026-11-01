---
tag: goreleaser
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:46Z
identity: goreleaser-20261101T171946-jay-flowers
tier: draft
---

GoReleaser v2's two GitHub changelog modes produce fundamentally different release notes. `changelog.use: github` enumerates individual commits and applies `groups`/`filters` regexps per-commit — merge commits, review-fixup commits, and chores all appear as top-level entries. `changelog.use: github-native` delegates changelog generation to GitHub's PR-based release-notes engine ("What's Changed" with contributor attribution and New Contributors sections); in this mode `sort`, `groups`, and `filters` do not apply — categories and exclusions are instead controlled by `.github/release.yml` in the release repository. Critical gotcha: GitHub does NOT inherit `release.yml` from the org-level `.github` repo, so the file must exist in the repository that cuts releases. Also note `release.yml` categories key off PR labels, not conventional-commit prefixes, so repositories whose org label set is narrow (e.g., only bug/documentation/enhancement) will bucket unlabeled-prefix PRs (chore/refactor/test/ci) into a catch-all "Other Changes" category until the label set is enriched.
