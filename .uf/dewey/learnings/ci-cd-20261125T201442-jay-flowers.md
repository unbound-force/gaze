---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-25T20:14:42Z
identity: ci-cd-20261125T201442-jay-flowers
tier: draft
as_of: "gaze@296458aaf226406c67d8efca3fe5a85fbe4d425f"
---

Literal SHA guard tests (`release_workflow_test.go`/`ci_workflow_test.go` asserting `jflowers/ship/.github/workflows/go-release.yml@86af2e7…` and `actions/checkout@11bd719…`) deadlock with Dependabot auto-merge when `dependabot.yml` groups include those same pins: bump PRs fail the required `CI OK` suite that `deps-automerge.yml` waits on, so they never merge. Prefer scoping Dependabot groups to exclude release-pipeline delegation pins over weakening guards to format-only; format checks still satisfy CI-001 but lose single-value drift detection for signing/SBOM/cask paths, and the protected `pinned action SHAs` gate then needs owner authorization and compensating tag-resolution verification per `AGENTS.md` Gatekeeping.
