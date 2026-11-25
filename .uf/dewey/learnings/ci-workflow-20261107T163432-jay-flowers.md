---
tag: ci-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-07T16:34:32Z
identity: ci-workflow-20261107T163432-jay-flowers
tier: draft
---

Dogfooding a Go tool through its own reusable GitHub Actions workflow should build from the checked-out tree behind an opt-in boolean input guarded by a module-path check (`go list -m` must equal the tool's module, else fail with an ::error:: pointing at the release-version input), never via `go install <module>@${{ github.sha }}`: PR merge-commit SHAs are ephemeral and resolve unreliably through the module proxy, and every push would publish a pseudo-version to the public module proxy and checksum DB. Stam
