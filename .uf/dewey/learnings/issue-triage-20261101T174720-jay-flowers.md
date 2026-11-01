---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:47:20Z
identity: issue-triage-20261101T174720-jay-flowers
tier: draft
---

Dependency-blocked issue labels go stale when the blocker closes. In gaze issue #269 (CRAP delta reporting, Group 2c), a September 2026 triage comment recorded "blocked by #268" (Pre-change CRAP gate / changed-function identification), but #268 closed on 2026-10-01 and its `--gate-on-change=origin/main|staged` capability shipped (documented in the README). Re-triage must verify the current state of declared blocking dependencies against the repository rather than treating a recorded blocking verdict as current — a closed blocker means the issue is actionable and the stale "blocked" label should be cleared as housekeeping.
