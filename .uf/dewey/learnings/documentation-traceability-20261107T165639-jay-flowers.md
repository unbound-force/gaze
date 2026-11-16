---
tag: documentation-traceability
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:56:39Z
identity: documentation-traceability-20261107T165639-jay-flowers
tier: draft
---

When a PR defers follow-up work, cite the tracking issue at every deferral site (OpenSpec proposal 'Out of scope', design 'Non-Goals'/decisions, AGENTS.md 'Recent Changes' — the project precedent cites issue numbers there), and when the PR supersedes part of an open issue's scope, annotate the issue in the same PR so it is not executed as written. Twin hazards observed on gaze PR 317 (dogfood-ai-report-from-source): the gaze-version-default-bump deferral appeared in three artifacts without citing the tracking issue #313, while build-from-source: true quietly mooted #313's dogfood-caller sub-item with no annotation. The citation half is a deterministic doc fix safely applied in-PR; the annotation half is an author/human action — review tooling must not mutate the issue tracker, so the assessment should scope it explicitly rather than drop it.
