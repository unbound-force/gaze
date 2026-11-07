---
tag: ci-security
author: jay-flowers
category: pattern
created_at: 2026-11-07T18:44:22Z
identity: ci-security-20261107T184422-jay-flowers
tier: draft
---

Security-review pattern for build-from-source CI dogfood paths that run PR-built binaries alongside a caller secret: assess the exposure DELTA, not the absolute exposure. On pull_request events, local reusable workflows (uses: ./...) and the caller both resolve from the PR merge ref, so same-repo PR authors already control every workflow step that runs with the secret in scope — building the tool from the same merge ref grants no new capability. The control that matters is the fork-PR gate: the secret must [retrieved via bounded council summary detail; text cut at 512 chars]
