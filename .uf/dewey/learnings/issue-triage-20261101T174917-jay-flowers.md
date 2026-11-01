---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:49:17Z
identity: issue-triage-20261101T174917-jay-flowers
tier: draft
---

Go statement-level coverprofile data cannot fully determine branch coverage: an else-less `if` has no statements in its false arm (so "condition never false" is indistinguishable from "condition sometimes false"), and short-circuit `&&`/`||` right operands are not tracked as separate basic blocks. For gaze issue #271 (branch coverage CI gate), this makes the prior triage comment's coverprofile-cross-reference approach an approximation of true branch coverage rather than an equivalent. The spec/clarify phase must explicitly choose between approximate coverprofile-derived branch coverage and real branch instrumentation (a Minimal Assumptions trade-off: instrumentation is invasive, derivation is incomplete), and the "performance impact benchmarked and acceptable" acceptance criterion needs a concrete threshold (e.g., max % slowdown vs. uninstrumented run) to be verifiable.
