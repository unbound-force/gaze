---
tag: security-governance
author: jay-flowers
category: pattern
created_at: 2026-11-07T15:49:05Z
identity: security-governance-20261107T154905-jay-flowers
tier: draft
---

Retrospective-learning storage increments need a verified-open-gap sweep. When a batch of learnings captured from a review cycle is committed, scan the batch for assertions of unfixed security gaps (markers like "residual gap", "verified as open", "still linkifies", "banned X but not Y") and require each such learning to carry a remediation reference — issue number or OpenSpec change id — in its body or the storage PR's description. Without the sweep, the knowledge base absorbs disclosure of live vulnerabil
