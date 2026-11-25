---
tag: documentation-traceability
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:56:41Z
identity: documentation-traceability-20261107T165641-jay-flowers
tier: draft
---

Deferral traceability: when a change defers work to a follow-up and an open issue already tracks that follow-up, cite the issue number at every deferral site within the change (proposal Out of scope, design Non-Goals/decisions, AGENTS.md Recent Changes, tasks) — an uncited "follow-up change" is not actionable and risks the work being lost or re-derived. The citation must land in all sites of the same change, not just one. When the change also invalidates part of the tracking issue's scope (e.g., bypassing the input an issue item says to bump), annotate the issue so it is not executed as written. Verified instance: gaze PR 317 deferred the reusable-workflow gaze-version bump in three new places without citing open issue 313, whose scope item 1 (bump the dogfood caller's gaze-version input in test.yml) became moot when the dogfood job switched to build-from-source: true.
