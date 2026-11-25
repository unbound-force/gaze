---
tag: ci-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-24T16:35:22Z
identity: ci-workflow-20261124T163522-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

Rolling out a required-check migration and the job gating that skips those checks in one PR creates a window where the old required names never report (skipped matrix legs don't emit per-leg checks). Safest sequence: add the aggregate job ungated, switch branch protection to require only the aggregate, then add the gating — never require a check the gating will skip.
