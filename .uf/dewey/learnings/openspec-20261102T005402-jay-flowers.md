---
tag: openspec
author: jay-flowers
category: gotcha
created_at: 2026-11-02T00:54:02Z
identity: openspec-20261102T005402-jay-flowers
tier: draft
---

When an OpenSpec change carries ## MODIFIED Requirements that chain against requirements a still-active (unarchived) predecessor change introduced via ## ADDED Requirements, the archive order is load-bearing: the predecessor must archive first. Archiving the successor first would apply its MODIFIED deltas against a base requirement that does not yet exist in the main specs, and would then let the predecessor's superseded text sync afterward, silently reverting the modification. Record the required archive ordering explicitly in the successor proposal's Impact section. Gaze precedent: gaze-init-triage-advisor's MODIFIED deltas chain against divisor-gaze-testing's ADDED requirements, so divisor-gaze-testing must archive first — the constraint is documented in the proposal's Impact section (verified in code review, iteration 2, 2026-11-01).
