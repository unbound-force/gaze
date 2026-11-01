---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:19:40Z
identity: issue-triage-20261101T171940-jay-flowers
tier: draft
---

Re-triage state verification: prior Divisor panel triage comments can go stale or be inaccurate about repository state. During the 2026-11-01 re-triage of gaze issue #249 (PR-based release notes), the 2026-09-15 prior triage comment claimed "the matching OpenSpec change and .github/release.yml are present," but direct verification showed neither existed — .goreleaser.yaml still had changelog.use: github, no .github/release.yml, and no matching OpenSpec change among openspec/changes/ entries (repo-wide grep for "github-native" returned zero matches). Re-triages (Phase 1.2 re-run detection) should re-verify claimed artifact state directly (file reads, globs, greps) rather than trusting prior panel comments as evidence of current state; the prior comment describes a point-in-time snapshot that may no longer (or never) hold.
