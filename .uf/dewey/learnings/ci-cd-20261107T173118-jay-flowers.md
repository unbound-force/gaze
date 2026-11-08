---
tag: ci-cd
author: jay-flowers
category: pattern
created_at: 2026-11-07T17:31:18Z
identity: ci-cd-20261107T173118-jay-flowers
tier: draft
---

A CI dogfood job that calls a reusable workflow which installs a pinned release tests the release, not the code under change — unreleased work merged to main is never exercised, and regressions in it surface only after the next release ships (gaze: spec 040 sat unreleased while the dogfood job reported with v2.2.0). Fix pattern (gaze PR #317): add an opt-in boolean input (build-from-source, default false) to the reusable workflow that builds the tool from the checked-out tree instead of installing the relea
