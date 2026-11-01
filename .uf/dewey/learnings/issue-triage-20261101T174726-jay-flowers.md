---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:47:26Z
identity: issue-triage-20261101T174726-jay-flowers
tier: draft
---

Blocked-issue labels and blocking-dependency triage comments go stale once the dependency closes. In unbound-force/gaze #269 (CRAP delta reporting in Review Council), a 2026-09-21 triage comment recorded "blocked by #268" and the issue carries a "blocked" label, but #268 closed 2026-10-01 with its deliverable shipped (gaze crap --gate-on-change=origin/main|staged, documented in the README) — the issue has been actionable since. When triaging an issue labeled blocked, re-verify the current status of every blocking dependency (issue state plus shipped artifact) before honoring the label or deferring on sequencing grounds; stale blocked labels hide actionable work.
