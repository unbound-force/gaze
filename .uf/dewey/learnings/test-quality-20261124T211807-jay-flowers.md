---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-24T21:18:07Z
identity: test-quality-20261124T211807-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
evidence:
  - ".uf/artifacts/dispatch/a4de843c-42ca-4c3c-8c02-838a46b09523.json"
---

When a contract test pins an embedded shell script (a YAML workflow `run:` block), each RFC 2119 MUST clause of the governing spec about the script's observable output needs its own pin. Verdict conditions, log-emitting lines, and behavioral stdout assertions are three distinct layers: a clause pinned at none of them silently loses regression protection. Observed in gaze's skip-ci-learnings-only-prs change: the classify script's `code=` verdict was pinned both textually (grep pins on the script text) and behaviorally (script executed against temp git repos), but the spec-mandated log listing (`echo "$outside"` / `echo "$links"`) was pinned at neither layer — deleting those lines left all 13 wiring tests green. The fix pattern: for every spec clause about emitted output, either pin the emitting line textually (as the dotdot test did for its labels) or assert the captured stdout in the behavioral harness (the stdout was already captured for diagnostics but never asserted).
