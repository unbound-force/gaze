---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:47:24Z
identity: issue-triage-20261101T174724-jay-flowers
tier: draft
---

Triage verdicts must verify the current state of cited blocking dependencies rather than trusting issue labels or prior triage comments — both are point-in-time records that outlive their premise. Gaze issue #269 (CRAP delta reporting in Review Council) carried a "blocked" label tied to dependency #268 (Pre-change CRAP gate), and its September 2026 automated triage comment concluded "purely a sequencing constraint," but #268 closed on 2026-10-01 with the capability shipped (`gaze crap --gate-on-change=origin/main` and `--gate-on-change=staged` documented in README). Neither the label nor the comment's premise was revisited, so the issue sat mislabeled as blocked while ready for work. A triage pass that re-checks dependency status (issue state + documented capability in README) before rendering a verdict both corrects the verdict inputs and unblocks ready-to-work issues.
