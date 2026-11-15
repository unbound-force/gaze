---
tag: regression-protection
author: jay-flowers
category: pattern
created_at: 2026-11-15T06:48:12Z
identity: regression-protection-20261115T064812-jay-flowers
tier: draft
as_of: "gaze@7965494232385e920cd63a717a79469c8b7eac5e"
---

When a spec mandates fail-before-output ("MUST fail before any comparison output is written"), asserting only the returned error leaves the ordering clause unenforced: a refactor that writes the output before failing keeps every test green. Tests for such requirements must also assert the output channel stays clean on the error path — e.g. assert stdout contains no comparison-section markers ("Baseline Comparison", "New Functions") or is empty when the error returns. Observed in gaze PR #345 (baseline-absolute-path-compat): the incompatible-baseline CLI tests asserted the error and diagnostic substrings but never inspected stdout, leaving spec requirement R4's ordering clause with no regression protection.
