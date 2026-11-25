---
tag: megalinter
author: jay-flowers
category: gotcha
created_at: 2026-11-16T03:56:25Z
identity: megalinter-20261116T035625-jay-flowers
tier: draft
as_of: "gaze@6d77d4d0f15cefa35fabf8d0de5eae2b5bffe5f1"
evidence:
  - ".mega-linter.yml"
  - "openspec/changes/reduce-markdownlint-noise-from-generated/design.md"
  - "openspec/changes/reduce-markdownlint-noise-from-generated/specs/lint-gating/spec.md"
  - "openspec/changes/reduce-markdownlint-noise-from-generated/tasks.md"
---

MegaLinter v9.3.0 registers markdownlint as a FORMATTER, so markdownlint errors default to NON-BLOCKING warnings — a green markdownlint check in MegaLinter v9.3.0 proves nothing about gate enforcement. The fix is one line in .mega-linter.yml: `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` (the generic DISABLE_ERRORS / DISABLE_ERRORS_IF_LESS_THAN knobs do not fix it; the linter-scoped variable is required). Verification methodology that proves the knob without a real violation: copy the repo (including .git and uncommitted changes) to a temp dir, append one non-auto-fixable violation (bare ``` fence = MD040, so APPLY_FIXES cannot mask it) to a linted file, and run scoped mega-linter pairs via `npx mega-linter-runner@9.3.0 --flavor go -e ENABLE_LINTERS=MARKDOWN_MARKDOWNLINT` — 0 findings must yield overall green, +1 MD040 must yield overall red. Scoped ENABLE_LINTERS runs also unconfound local probes from gitleaks, which otherwise fails on secret-redaction test fixtures found in fork-only git history (environmental, absent in CI). Discovered implementing gaze issue #295 (reduce-markdownlint-noise-from-generated).
