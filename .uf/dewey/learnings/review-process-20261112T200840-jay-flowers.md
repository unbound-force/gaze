---
tag: review-process
author: jay-flowers
category: pattern
created_at: 2026-11-12T20:08:40Z
identity: review-process-20261112T200840-jay-flowers
tier: draft
---

When a review advisory on a narrowly-scoped PR requests a fix that is itself a CI-configuration change (a new test wired into test.yml, a new script under .github/scripts/), the correct handling under this repo's spec-first governance is to spawn a follow-up OpenSpec change rather than expand the originating PR. Precedent: PR #317's canary advisory became the release-install-canary OpenSpec change itself. Deferring such an advisory with a tracked follow-up path is AUTHOR-DECIDES, not dismissal, and avoids out-of-spec scope creep in the reviewed PR.
