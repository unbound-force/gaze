---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:30:25Z
identity: issue-triage-20261101T183025-jay-flowers
tier: draft
---

When triaging lint-noise issues (unbound-force/gaze #295: "reduce markdownlint noise from generated and tool-owned artifacts"), do not assess staleness by issue-link archaeology alone — adjacent CI-fixup commits can implement most of the ask without referencing the issue. In gaze, the markdownlint exclusions for .uf/, openspec/changes/, and embedded asset mirrors were last adjusted by a human commit "chore(#306): fix gitleaksignore SHAs and markdownlint exclusions for CI" (per the local reflog, 2026-10-31), while a repo-wide grep for "#295" returns zero matches and no OpenSpec change exists for it. Verify each acceptance criterion against current config state instead. Two verification traps: (1) MegaLinter FILTER_REGEX_EXCLUDE values are unanchored substring regexes — "specs/" matches "openspec/specs/" but NOT "openspec/schemas/", so per-path coverage must be checked substring-by-substring; that schemas/templates gap is exactly the remaining scope that keeps such an issue actionable after the exclusions land. (2) Lint-noise issues bundle three separately-landing deliverables — path exclusions, a violation baseline/tracking mechanism, and policy documentation — so exclusions landing first does not satisfy the remaining acceptance criteria. Implementation-safety rule for the fix: noise reductions belong in the linter-scoped variable (MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE), never the global FILTER_REGEX_EXCLUDE, because the global variable applies to all enabled linters (yamllint and potentially gitleaks secret scanning on matched paths) — routing a style-lint cleanup through it widens into coverage loss on other checks.
