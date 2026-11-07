---
tag: documentation-traceability
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:56:45Z
identity: documentation-traceability-20261107T165645-jay-flowers
tier: draft
---

Deferral hygiene for spec'd changes (verified on gaze PR 317, dogfood-ai-report-from-source): when a PR defers work that an open tracking issue already covers, cite that issue at every deferral site — proposal Out of scope, design Non-Goals/decisions, the AGENTS.md Recent Changes entry, and the tasks.md continuation note — and when the PR invalidates part of that issue's scope, annotate the issue (comment or scope edit) in the same change. There, issue #313 tracked the deferred gaze-version bump, four deferral sites (proposal.md:46, design.md:24 and 61, AGENTS.md:536, tasks.md:38) omitted the citation while neighboring Recent Changes entries cite theirs, and #313's 'bump the dogfood caller input in test.yml' sub-item became moot the moment the dogfood job switched to build-from-source: true. Two cheap checks catch this class: grep the diff for the tracking issue number, and diff the issue's scope items against inputs/flags the PR removes from use.
