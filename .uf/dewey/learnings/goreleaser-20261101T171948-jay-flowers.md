---
tag: goreleaser
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:48Z
identity: goreleaser-20261101T171948-jay-flowers
tier: draft
---

GoReleaser's `changelog.use: github` mode enumerates individual commits and applies the `groups`/`filters` regexps per-commit — it does NOT produce GitHub's PR-based "What's Changed" release notes (PR-headed entries, contributor attribution, New Contributors). PR-based notes require `changelog.use: github-native`, with grouping controlled by `.github/release.yml` (label-based categories) living in the repository itself; `sort`, `groups`, and `filters` do not apply in github-native mode. Category grouping keys off PR labels, so repos whose org label set only defines bug/documentation/enhancement will bucket chore/refactor/test/ci PRs into a catch-all category until the org labels are enriched. Observed in unbound-force/gaze issue #249 (v1.7.0 noisy per-commit changelog root-caused to this exact config confusion); the same per-commit grouping pattern exists in sibling repos' GoReleaser configs, so the limitation is likely org-wide.
