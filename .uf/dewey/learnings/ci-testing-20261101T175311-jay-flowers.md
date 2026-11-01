---
tag: ci-testing
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:11Z
identity: ci-testing-20261101T175311-jay-flowers
tier: draft
---

Release-pipeline authorization gates (repository identity checks, required-secret enforcement) cannot be exercised end-to-end without publishing real artifacts, so the verifiable regression strategy is a static workflow-configuration test: parse the workflow YAML and assert (a) the authorization job exists and probes every required secret by name, (b) every downstream publishing job — preflight/tag creation, release creation, signing, cask publication — declares a dependency on it, (c) no path reaches tag creation without authorization, and (d) diagnostics reference secret names only, never values. Cover the full matrix: canonical repository with all secrets, canonical repository missing each secret individually, and fork execution. Precedent: gaze's test-validate-workflow-floor.sh (repository-invariant assertions plus a synthetic case matrix) and the TestReleaseWorkflow_AuthorizationGraph design proposed in unbound-force/gaze#267.
