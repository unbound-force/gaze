---
tag: lint-governance
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:30:28Z
identity: lint-governance-20261101T183028-jay-flowers
tier: draft
---

When triaging markdownlint-noise issues in repos using MegaLinter, diff the issue's named noise paths against the live MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE before assessing remaining scope: the exclusion regex is unanchored, so alternatives have non-obvious substring coverage. In unbound-force/gaze, `openspec/changes/` silently covers `openspec/changes/archive/`, and the bare `specs/` alternative covers Speckit `specs/`, OpenSpec main specs `openspec/specs/`, and per-change delta specs — while sibling `openspec/schemas/unbound-force/templates/` (tool-owned templates that intentionally conflict with MD041/MD022/MD032) remains linted. The repo precedent is path exclusion with per-path rationale documented as comments directly above the regex in .mega-linter.yml (byte-identical mirror assets under sync-test invariants, regenerating Dewey learnings, template-text-bearing artifacts) — extend that pattern rather than reformatting generated content.
