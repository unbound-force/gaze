---
tag: github-actions
author: jay-flowers
category: gotcha
created_at: 2026-11-01T15:59:27Z
identity: github-actions-20261101T155927-jay-flowers
tier: draft
---

GitHub Actions parses workflow commands (::error::, ::notice::, ::warning::) from a step's stdout only. Redirecting them to stderr (>&2) makes the runner skip parsing, so the annotation silently disappears from the PR/check-run UI — the message still appears in raw logs but loses its surfaced, actionable form. Rule of thumb for CI validation scripts: workflow-command messages go to stdout via plain printf; plain usage/diagnostic messages for humans reading a terminal go to stderr. This bit the gaze validate-workflow-floor gate in its first council pass and was corrected by printing ::error::/::notice:: lines to stdout while keeping malformed-tag/usage errors on stderr.
