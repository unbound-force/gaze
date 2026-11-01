---
tag: ci-gate-integrity
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:49:06Z
identity: ci-gate-integrity-20261101T174906-jay-flowers
tier: draft
---

When adding a new threshold gate flag to gaze (e.g., issue #271's proposed --min-branch-coverage), the gate must fail closed from day one: an explicitly-set threshold (cmd.Flags().Changed() == true) must exit non-zero when the underlying metric is unavailable OR when zero functions/branches were analyzed. This extends the ci-gate-integrity precedent (issues #101/#108/#116; see learning/ci-gate-integrity-20260616T220950-jay-flowers for the *int nil-vs-zero propagation pattern) with the zero-result rule already enforced in runCrap and aireport.Run: "could not analyze" must never compare as a pass. New gate specs should carry this as an explicit acceptance criterion (mirroring EvaluateThresholds' nil-actual failure and the zero-result gates) so the unavailable/zero cases are designed in, not patched after a silent-pass bug ships.
