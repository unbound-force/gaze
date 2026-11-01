---
tag: documentation-consistency
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:33:11Z
identity: documentation-consistency-20261101T183311-jay-flowers
tier: draft
---

Shared CLI flags require fan-out documentation updates across all sibling command reference pages. In gaze, --analyzer and --language are registered on the crap, quality, and report commands (cmd/gaze/main.go newReportCmd registers them at lines 2472-2473), and each command has a page under docs/reference/cli/. PR #281 updated crap.md and quality.md but missed report.md (gaze issue #283), and the gap survived 3 automated review rounds because nothing diffs a command's registered flags against its reference-page Flags table. Verification recipe for triage/review of docs-gap issues: list the cmd.Flags() registrations in new<Cmd>Cmd(), diff them against the Flags table in docs/reference/cli/<cmd>.md, and grep the flag names across docs/reference/cli/ — the diff catches the whole class of gap, not just the rows named in the issue (the same diff shows report.md also omits --test-short, registered main.go:2474). When writing the missing rows, mirror the canonical help strings from the flag registration; sibling pages' table descriptions are command-tailored, so substance (name/type/default/semantics) should be consistent while prose is adapted per command.
