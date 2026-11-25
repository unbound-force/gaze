---
tag: coverage-strategy
author: jay-flowers
category: gotcha
created_at: 2026-11-23T20:55:57Z
identity: coverage-strategy-20261123T205557-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

Delegating pipeline behavior to an external reusable workflow (e.g., a repo's release.yml calling a shared go-release.yml) splits its test coverage: the delegating repo can only guard the caller contract (workflow name, pins, inputs, secret mappings, job ordering), while the delegated behavior is verifiable only in the callee's repository. The spec record for such a migration — including a retroactive one written after a Spec-First violation — must document this coverage boundary by name: which guard test asserts which caller invariants, and which behavior the callee owns. A retroactive record without the coverage strategy repeats Constitution IV's "missing coverage strategy is CRITICAL" violation at the documentation layer. Observed on gaze PR 362 (release-via-ship migration): TestReleaseWorkflow_CallsShip guards the caller side, the partial-signing-secrets-fail behavior lives only in jflowers/ship, and no artifact records that split as intentional.
