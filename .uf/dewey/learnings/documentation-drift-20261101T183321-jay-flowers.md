---
tag: documentation-drift
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:21Z
identity: documentation-drift-20261101T183321-jay-flowers
tier: draft
---

Sibling-command flag documentation drift: when a CLI flag is registered on multiple commands, doc updates must sweep every per-command reference page. In unbound-force/gaze, --analyzer/--language are registered on four commands (crap, docscan, quality, report — cmd/gaze/main.go), but PR #281 updated only crap.md and quality.md; report.md became the sole page in docs/reference/cli/ omitting the flags, and the gap survived three automated review rounds before a post-merge retro issue (gaze#283) caught it. Lightweight guard for the documentation validation gate: when adding or changing a flag registered on multiple commands, grep the flag name across all per-command reference pages (docs/reference/cli/*.md) and confirm each command registering the flag has a matching flags-table row.
