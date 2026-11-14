---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:39:19Z
identity: test-quality-20261102T213919-jay-flowers
tier: draft
---

Validating an "untested behavior" claim requires enumerating synthetic in-memory cases alongside fixture files, and naming the exact regression variant at risk. On gaze PR #314 feedback item 4, "no fixture exercises \r\n endings" was true at the testdata level, yet TestExtractFinalText_Synthetic's exact-match "crlf framing is tolerated" case already pinned CRLF framing — fixture-scoped verified facts can be technically true while the suite covers the behavior synthetically, and the accurate gap was only interleaved junk. Regression taxonomy for skip-on-error line parsers: an all-invalid fixture catches blanket hard-fail regressions (the error message changes), but a drop-neighbors regression (break instead of continue) reproduces the same zero-events error on all-invalid input and is invisible on all-valid input — only a mixed valid+junk input catches it. Put the mixed-input test at the unguarded unit layer when the subprocess replay layer skips under testing.Short, so the regression gate runs in short CI.
