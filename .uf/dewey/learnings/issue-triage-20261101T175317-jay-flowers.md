---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:17Z
identity: issue-triage-20261101T175317-jay-flowers
tier: draft
---

Release pipelines that degrade gracefully when signing secrets are absent (skip signing, publish unsigned artifacts) create a supply-chain gap when that degradation path is reachable from untrusted execution contexts. GitHub secrets do not propagate to fork runs, so a fork executing a publication-capable workflow_dispatch takes the fail-open path with real write access to distribution channels (e.g., a Homebrew tap). Triage signal: documented "graceful degradation" on publication jobs is a security-hardening candidate, not just a UX convenience. Fix pattern: fail-closed authorization (canonical-repository identity check plus required-secret presence) ordered before any tag creation, release publication, signing, or cask upload — tag creation is the point of no return, so authorization must precede the preflight job that creates the annotated tag. Back the routing with static workflow-configuration tests over the dependency graph, and document provisioning, blocked-release remediation, and rollback. Prior art in gaze: the unsigned-cask fallback originated as spec 015 graceful degradation (see learning/release-pipeline-20260611T201838-jay-flowers); issue #267 reverses that guarantee for official releases.
