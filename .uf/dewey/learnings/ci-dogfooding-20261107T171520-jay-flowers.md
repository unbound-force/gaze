---
tag: ci-dogfooding
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:15:20Z
identity: ci-dogfooding-20261107T171520-jay-flowers
tier: draft
---

When a repository's dogfood job is the only in-repo caller of its own reusable workflow, switching that job to build-from-source leaves the reusable workflow's release-install step with zero in-repo executors: every line added to that step (renames, if-guards, version echoes) ships unexercised in-repo and is validated only reactively by external consumer runs. Before accepting that trade-off in a design doc's Risks section, check whether the dogfood job was ever real verification of the release path — a pinned tooling-tag install is not release-publication verification (learning/release-verification-20261101T185210-jay-flowers: post-publication smoke tests installing the exact published tag are the correct closure for consumability). Track the canary as a follow-up issue rather than expanding the dogfood change: a new or scheduled CI workflow requires its own spec under the spec-first mandate, and the dogfood change's scope discipline is part of its value.
