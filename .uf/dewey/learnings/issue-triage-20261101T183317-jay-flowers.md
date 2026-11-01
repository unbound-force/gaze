---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:17Z
identity: issue-triage-20261101T183317-jay-flowers
tier: draft
---

Cross-command flag documentation drift pattern (gaze issue #283): when a CLI flag is registered on multiple Cobra commands (--analyzer/--language on crap, quality, docscan, and report), docs PRs tend to update only the reference pages they touched — PR #281 fixed crap.md and quality.md, missed report.md, and the medium-severity stale-docs finding survived three review rounds to merge. Triage verification recipe: grep the docs reference directory for the flag name AND grep cmd/gaze/main.go for its StringVar registration; in #283 the mismatch was exact (the report command registers both flags at main.go:2472-2473 while report.md had zero 'analyzer'/'language' matches, making it the sole outlier among all command reference pages). Also: fullsend retro-bot issues sourced from unresolved PR review findings are high-confidence, but their triage-agent comments can be topically mismatched (comment 2 discussed HTML formatter issues #8/#260 on a CLI-docs issue) — weight the issue body and repository evidence over bot comments.
