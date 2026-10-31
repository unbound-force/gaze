---
tag: gatekeeping
author: jay-flowers
category: gotcha
created_at: 2026-10-30T22:01:31Z
identity: gatekeeping-20261030T220131-jay-flowers
tier: draft
---

Tests that must plant secret-shaped fake values (e.g., to exercise a redaction/scrubbing path) will trip the CI secret scanner (gitleaks via MegaLinter), and the standard fix — commit-pinned, file/rule/line-scoped .gitleaksignore entries — is itself a lint-gate behavior change that spec artifacts never enumerate, because the false positives only exist after implementation. Treat such suppressions as gate modifications needing a paper trail: record the .gitleaksignore addition in the proposal's Impact section or the PR description with the rationale (the flagged values are deliberate fakes the tests require), otherwise the suppression reads as an undocumented weakening of a protected gate (AGENTS.md Gatekeeping Value Protection category 4, linter configuration) and directly contradicts an Impact claim like "no CI configuration changes". Reviewer check before accepting the suppression as legitimate: verify each fingerprint's file:line targets a provably fake fixture constant, not production code.
