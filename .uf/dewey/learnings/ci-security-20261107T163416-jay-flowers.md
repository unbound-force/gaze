---
tag: ci-security
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:34:16Z
identity: ci-security-20261107T163416-jay-flowers
tier: draft
---

When a reusable workflow gains a build-from-source input that compiles and runs the caller's checked-out tree, evaluate the trust boundary against what a same-repo PR author could ALREADY do — not against the previously-installed binary. For callers triggered by plain `pull_request` (not `pull_request_target`), workflow files resolve from the PR merge ref with secrets available on same-repo PRs, so the PR author already controls the job's executed code (they can edit the workflow itself in the same PR). Com
