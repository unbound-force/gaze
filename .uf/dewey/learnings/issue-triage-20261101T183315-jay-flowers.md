---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:15Z
identity: issue-triage-20261101T183315-jay-flowers
tier: draft
---

Cross-command CLI flags drift documentation non-atomically: when one flag ships on multiple commands (gaze's --analyzer/--language are registered on crap, quality, docscan, and report), a PR that updates only a subset of the per-command reference pages leaves the rest silently stale — PR #281 (unbound-force/gaze) fixed crap.md and quality.md but missed report.md, and a medium-severity review finding repeated across 3 consecutive rounds still merged unaddressed, later surfacing as retro issue #283. Such docs-gap issues are objectively triageable with two checks: grep the command constructor for the flag registration (confirming the flag exists and is not MarkHidden) and grep the docs page for the flag name (confirming absence). The strongest issue format pairs a mirror-source (the sibling reference page whose wording to copy) with a machine-checkable validation command; note that open-ended grep criteria ("should return all three files") can be accidentally satisfied by unrelated matches, so criteria that name the target file explicitly are more robust.
