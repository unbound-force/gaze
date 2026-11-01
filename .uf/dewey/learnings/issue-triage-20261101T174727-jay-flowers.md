---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:47:27Z
identity: issue-triage-20261101T174727-jay-flowers
tier: draft
---

Triaging dependency-blocked issues must verify the blocking dependency's current state rather than honoring a 'blocked' label or repeating a prior triage's sequencing verdict. Labels and triage comments are point-in-time records, not live state: in unbound-force/gaze, issue #269 ("CRAP delta reporting in Review Council") remained labeled 'blocked' after its dependency #268 (the --gate-on-change pre-change CRAP gate) closed on 2026-10-01, leaving a September triage comment's blocking premise stale two months later. Before concluding a dependency-blocked issue is unactionable, check the linked issue's current status and, where repository context allows, confirm the dependency capability actually shipped (e.g., the flag appearing in README's command documentation).
