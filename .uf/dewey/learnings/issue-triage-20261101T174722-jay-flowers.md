---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:47:22Z
identity: issue-triage-20261101T174722-jay-flowers
tier: draft
---

Triage verdicts have a shelf life: dependency states recorded in issue labels and comments go stale. Gaze issue #269 (CRAP delta reporting in Review Council) carried a `blocked` label and a 2026-09-21 triage comment citing open dependency #268 (Pre-change CRAP gate / --gate-on-change). By 2026-11-01, #268 had closed (2026-10-01) and its `--gate-on-change=origin/main` / `--gate-on-change=staged` capability shipped and was README-documented — leaving both the label and the recorded blocking premise stale while the issue remained actionable. When picking up or re-triaging an issue labeled `blocked`, verify the dependency's current state (issue open/closed, capability present in README/docs) instead of trusting the label or prior triage comments; labels and comment conclusions are point-in-time snapshots, not live state.
