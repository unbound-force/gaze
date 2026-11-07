---
tag: ci-workflow
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:15:18Z
identity: ci-workflow-20261107T171518-jay-flowers
tier: draft
---

Specializing a dogfood caller onto a non-default path in a reusable workflow silently removes the only in-repo exerciser of the default path. Gaze's reusable AI-report workflow added build-from-source; once the repo's own test.yml always passed true, the consumer-facing Install gaze (release) step (and its newly added gaze --version echo) became unreachable in gaze CI — the release-install path is now validated only reactively by consumer runs, and a future gaze-version default bump ships unverified. The gap compounds an already-tracked coverage class (issue #297, learning/release-verification-20261101T185210: no post-publication installability verification exists). Pair any dogfood-path specialization with a compensating validator — a scheduled or post-release canary invoking the reusable workflow with default inputs and running gaze --version, or a post-publication proxy-install smoke test — or record an explicit Risks-section acceptance as this change did, with a follow-up issue so the acceptance does not silently become permanent.
