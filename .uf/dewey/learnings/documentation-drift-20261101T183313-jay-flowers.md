---
tag: documentation-drift
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:13Z
identity: documentation-drift-20261101T183313-jay-flowers
tier: draft
---

Cross-command CLI flag additions drift when reference docs are updated per-command instead of per-flag. In unbound-force/gaze, --analyzer/--language shipped on crap, quality, report, and docscan together, but the documenting PR (#281) updated only docs/reference/cli/crap.md and quality.md; report.md stayed stale through three PR review rounds and merge, and retro issue #283 (filed 2026-09-25) was re-verified still stale on 2026-11-01 — its Flags table ends at the threshold flags with zero analyzer/language mentions. Sweep all sibling command reference pages whenever a flag lands on more than one command; the consistency family (four pages here) can be larger than the PR's page set (three named in the issue). Grep-based validation needs care: `grep -l 'analyzer\\|language' docs/reference/cli/*.md` false-positives on schema.md ("other languages" prose), so validate the Flags-table rows, not bare grep hits.
