---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-23T21:08:32Z
identity: test-quality-20261123T210832-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

When a repo delegates its pipeline to a shared reusable workflow (e.g., release.yml calling jflowers/ship's go-release.yml) and guards the delegation with a static workflow-configuration test, the guard's assertion surface must include both the secrets: mappings AND every with: input line. GitHub Actions resolves a dropped or mistyped with: key or ${{ inputs.* }} expression to an empty value or the callee's default instead of erroring, and dispatch-only workflows (release pipelines) have no scheduled CI execution to surface the drift — it appears only when the next real release fails (empty tag, preflight validating the wrong checks, cask pushed to the wrong tap). A table-driven exact-line strings.Contains loop over the with: block pins the whole contract at near-zero cost and matches the technique already used for secret mappings and needs ordering. Observed on gaze PR 362 (release-via-ship): TestReleaseWorkflow_CallsShip asserted the six MACOS_* mappings and HOMEBREW_TAP_TOKEN but none of the six forwarded inputs (tag, ci_checks, three skip_* flags, tap), leaving half the caller contract unguarded. Complements learning/coverage-strategy-20261123T205557-jay-flowers, which names inputs as part of the caller contract a delegating repo must document; this lesson specifies the guard-test mechanism that enforces it.
