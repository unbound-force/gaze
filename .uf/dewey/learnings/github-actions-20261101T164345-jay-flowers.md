---
tag: github-actions
author: jay-flowers
category: pattern
created_at: 2026-11-01T16:43:45Z
identity: github-actions-20261101T164345-jay-flowers
tier: draft
---

GitHub Actions parses workflow commands (::error::, ::stop-commands::, ::add-mask::) from a step's STDOUT only — a property that gates both annotation rendering AND workflow-command injection safety. In bash CI gate scripts that echo caller-supplied values before validating them (e.g., a malformed release tag rejected by a regex), route those pre-validation echoes to stderr, and reserve stdout ::command:: messages for values already constrained by a strict grammar regex or inherently single-line-derived (e.g., a constant grepped from one source line — grep is line-oriented, so the extracted value cannot contain a newline). gaze's validate-workflow-floor.sh demonstrates the complete pattern: the unvalidated release tag appears only in the stderr message ('invalid release tag: %s' >&2) where the runner cannot parse it as commands, while every stdout ::error:: interpolates values already validated against a strict semver character class — making workflow-command injection impossible by construction without any escaping machinery, while annotations still render.
