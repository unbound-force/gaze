---
tag: ci-documentation
author: jay-flowers
category: gotcha
created_at: 2026-11-13T18:21:05Z
identity: ci-documentation-20261113T182105-jay-flowers
tier: draft
---

Adding a release gate to a workflow is a documentation sweep, not a one-line workflow edit. In the gaze repo the in-repo obligation for CI/CD-only changes is two AGENTS.md surfaces — the CI/CD section that enumerates release gates and preflight dependencies, and a Recent Changes entry — and satisfying only the Recent Changes entry leaves the release runbook misstating which unskippable gates exist. Gaze PR #335 added a third preflight gate and updated the release.yml header comment to "Three local gates" while AGENTS.md still said "Two local gates", producing a document pair that contradicts itself exactly when an operator is diagnosing a blocked release.
