---
tag: release-tooling
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:53Z
identity: release-tooling-20261101T171953-jay-flowers
tier: draft
---

GoReleaser changelog-mode gotcha (verified while triaging unbound-force/gaze#249): `changelog.use: github` is commit-based — it enumerates commits and applies `groups`/`filters` regexps per-commit — despite the name suggesting GitHub-native notes. PR-based "What's Changed" notes require `changelog.use: github-native`, which delegates to GitHub's release-notes engine and ignores `sort`/`groups`/`filters`; grouping is then controlled by a repo-local `.github/release.yml`, which GitHub does NOT inherit from the org-level `.github` repository, so each repo needs its own copy. Org-wide relevance: the commit-based pattern (Features/Bug Fixes/Documentation/Others + exclude `chore:`) was replicated from unbound-force/unbound-force into sibling repos (gaze, replicator's homebrew-distribution change), so several repos share this confusion surface. When triaging changelog/release-notes issues, read the active GoReleaser `changelog.use` mode before assuming the config's groups/filters shape the published notes.
