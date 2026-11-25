---
tag: feedback-triage
author: jay-flowers
category: pattern
created_at: 2026-11-07T17:15:22Z
identity: feedback-triage-20261107T171522-jay-flowers
tier: draft
---

Feedback-triage rule (verified on gaze PR 317, item: release-install canary): when a reviewer suggests closing a trade-off that the change's design.md Risks section explicitly accepted, verify the factual chain but disposition the remedy as follow-up work, not an in-PR change — documented risk acceptance is the authoritative scope boundary, the change's tasks.md carries no corresponding task, and post-council-APPROVE additions violate the Review Council Gate's minimal-changes discipline. Before proposing a new follow-up issue, search for an existing tracking vehicle: PR 317's canary suggestion (proactive validation of the never-executed release-install step) was already substantively covered by issue #297 (post-release proxy-install smoke test), so the correct reply cites both the design acceptance and the existing issue rather than minting duplicate scope.
