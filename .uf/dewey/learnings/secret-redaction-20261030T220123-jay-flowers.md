---
tag: secret-redaction
author: jay-flowers
category: gotcha
created_at: 2026-10-30T22:01:23Z
identity: secret-redaction-20261030T220123-jay-flowers
tier: draft
---

PR-body acknowledgment of a review advisory does not discharge the requirement to record a security control's scope decisions in its spec artifacts while they are still amendable (pre-archive). The PR body is ephemeral text; the archived openspec/changes/<name>/design.md and the consumer-facing known-limits docs are the permanent record maintainers and auditors read. On gaze's redact-env-values-from-adapter-stderr PR (#305), the deliberate stdout-not-scrubbed exclusion — child stdout flows into the AI report appended to $GITHUB_STEP_SUMMARY, a channel covered by neither the stderr scrub nor GitHub's job-log-scoped masking, and verbatim-value scrubbing of stdout was rejected because it would corrupt legitimate report content (workspace paths and long identifiers routinely exceed the 32-byte threshold) — was acknowledged in the PR body per the pre-PR review's advisory, yet design.md's Non-Goals and ci-integration.md's known-limits paragraph still omit it, leaving the exclusion indistinguishable from an oversight in the artifact about to be archived. When a review advisory targets a spec artifact of an unarchived change, apply the artifact edit during the review phase (docs-only minor fixes are review-phase-legal) rather than carrying the advisory as PR-body text.
