---
tag: github-actions
author: jay-flowers
category: gotcha
created_at: 2026-11-12T20:46:15Z
identity: github-actions-20261112T204615-jay-flowers
tier: draft
---

A CI availability/diagnostic guard added in a verification step is dead code when an earlier step already invokes the same binary unguarded. On gaze's install-canary.yml (PR #331 fix round), `command -v gaze` was guarded with an actionable ::error:: in `Verify installed module version`, but `Install gaze (release)` already ends with a bare `gaze --version`; with the binary off PATH that line fails first (bash exit 127), the job stops, and the annotation never renders. Both steps inherit the same PATH. Fix: place the guard at the FIRST invocation of the binary (guard, then run the command), and document why later lookups are safe.
