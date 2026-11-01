---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:33:09Z
identity: issue-triage-20261101T183309-jay-flowers
tier: draft
---

Issue acceptance criteria built on unanchored directory-wide greps can false-pass before the fix lands — at triage, run the issue's own validation command against the current tree and confirm it FAILS before trusting it as the done-check. Gaze issue #283 (add --analyzer/--language rows to docs/reference/cli/report.md) validates with `grep -l 'analyzer\\|language' docs/reference/cli/*.md` "should return all three files", but that grep already returns four files today (crap.md, quality.md, docscan.md, and schema.md via the prose word "languages") — report.md is not among them — so the criterion passes without any fix. The issue also undercounts the flag surface ("all three commands"): four commands register --analyzer (newCrapCmd main.go:1186, newDocscanCmd :1347, newQualityCmd :2017, newReportCmd :2472). Rule: anchor doc-gap acceptance criteria to the specific artifact — scope the grep to the target file or assert the exact table rows — and derive command-flag inventories from flag-registration sites in code, not from the reference pages that happen to document them.
