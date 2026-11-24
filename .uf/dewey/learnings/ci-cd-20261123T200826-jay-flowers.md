---
tag: ci-cd
author: jay-flowers
category: pattern
created_at: 2026-11-23T20:08:26Z
identity: ci-cd-20261123T200826-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

Reusable-workflow caller guard tests (gaze's TestReleaseWorkflow_CallsShip pattern) are the only automated regression protection for a release pipeline's caller contract: GitHub validates `with:` inputs and mapped secret names only at dispatch time, yamllint checks syntax only, and the pipeline runs end-to-end only at release. Enumerate the full caller-side surface in the guard: every input carrying a repo-specific value (e.g. `tap: unbound-force/homebrew-tap`), every secret mapping including renames (HOMEBREW_TAP_TOKEN <- HOMEBREW_TAP_GITHUB_TOKEN), the full-SHA pin with version comment, needs-ordering of local gates, and cross-workflow name dependencies (post-release-install-check.yml triggers on the workflow name "Release"). Guarding the secret that authorizes an operation while not guarding the operation's destination (token asserted, tap input not) is the classic partial-coverage asymmetry. When migrating a reusable-workflow call, also audit dropped semantics-bearing inputs (gaze dropped allow_prerelease: true moving from org-infra preflight to ship go-release.yml) — the callee's default is invisible to caller-side tests. Extends learning ci-testing-20261101T175311-jay-flowers (static workflow-configuration tests as the verifiable regression strategy for release-pipeline gates).
