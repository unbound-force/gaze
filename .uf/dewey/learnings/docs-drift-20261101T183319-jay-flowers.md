---
tag: docs-drift
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:19Z
identity: docs-drift-20261101T183319-jay-flowers
tier: draft
---

Per-command CLI reference pages drift when a flag shared across several commands is documented only on the pages a PR happens to touch. In unbound-force/gaze, --analyzer/--language are registered on the crap, quality, docscan, and report commands (cmd/gaze/main.go), but the PR #281 doc sweep updated only docs/reference/cli/crap.md and quality.md, leaving report.md the sole command page without the rows — even though a review agent flagged the gap in all 3 review rounds before the PR merged. Verification recipe when triaging or reviewing flag-documentation changes: grep the full flag tokens (e.g. '--analyzer', '--language') across every page in docs/reference/cli/ and diff the hit set against the flag-registration sites in the command constructors; bare-word greps (grep -l 'analyzer|language') can false-positive on prose mentions, so match the token including its leading dashes.
