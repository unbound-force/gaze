---
tag: documentation-traceability
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:56:43Z
identity: documentation-traceability-20261107T165643-jay-flowers
tier: draft
---

When a PR defers work to a follow-up change, cite the tracking issue at every deferral site (proposal.md Out of scope, design.md Non-Goals, AGENTS.md Recent Changes entry, tasks.md follow-on task) — the citation window closes at merge for spec artifacts, which become immutable point-in-time records under Spec Artifact Immutability, so an uncited deferral is permanently untraceable from the artifact that defers it. When a PR moots part of an open tracking issue's scope (e.g., removing an input the issue instructs a future change to bump), annotate or scope-edit the issue in the same PR so it is not executed as written. Verified in gaze PR 317 (dogfood-ai-report-from-source): three uncited deferral sites plus issue #313 (spec 040 T080, whose drafted scope included bumping the dogfood caller's gaze-version input) whose dogfood clause was mooted by build-from-source: true — #313 confirmed OPEN via gh issue view and confirmed the sole open tracker for the follow-up.
