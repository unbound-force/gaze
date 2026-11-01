---
tag: ci-security
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:06Z
identity: ci-security-20261101T175306-jay-flowers
tier: draft
---

A release-pipeline security gate is only as early as the first side effect it must prevent. In gaze's release.yml, the six-signing-secret probe (check-signing-secrets) runs downstream of preflight (which creates the annotated tag) and in parallel with the release job (which publishes artifacts); an incomplete secret set then routes to push-unsigned-cask, which pushes an unsigned cask to the official Homebrew tap using HOMEBREW_TAP_GITHUB_TOKEN. Fail-open secret routing thus converts a credential misconfiguration into a silent provenance downgrade of official artifacts instead of a failed release. The corrective pattern (gaze issue #267): run repository-identity plus all-required-secrets authorization before any job with persistent side effects (tag creation, release publication, signing, tap push), fail closed rather than routing to an unsigned fallback, emit only secret names in diagnostics, and pin the authorization-routing graph with static workflow-configuration tests — gaze's TestReleaseWorkflow_MacOSSigningIdentity in cmd/gaze/release_workflow_test.go is the extendable precedent.
