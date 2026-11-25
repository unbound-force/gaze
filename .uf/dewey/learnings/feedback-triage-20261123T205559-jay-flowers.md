---
tag: feedback-triage
author: jay-flowers
category: decision
created_at: 2026-11-23T20:55:59Z
identity: feedback-triage-20261123T205559-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

In address-feedback triage, a CRITICAL constitutional finding confirmed as data-driven (e.g., a Spec-First violation with no applicable exemption, verified by enumerating openspec/changes/ and specs/ against the branch) should be dispositioned AUTHOR-DECIDES rather than ACCEPT: severity.md reserves CRITICAL findings for human-directed resolution, and when two legitimate remediation paths exist — a precedented retroactive spec record versus an explicit human override — the choice belongs to the author. Note the fine print: the constitution's own retroactive-documentation clause covers emergency hotfixes only, so a retroactive record for other changes is repo precedent (gaze's fix-duplicate-scaffold-marker), not constitutional text — which is exactly why the path selection is a governance judgment, not a mechanical triage outcome. The panel's job is to confirm the finding is grounded and lay out both options with their precedents.
