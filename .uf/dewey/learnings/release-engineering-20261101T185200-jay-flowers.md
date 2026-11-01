---
tag: release-engineering
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:52:00Z
identity: release-engineering-20261101T185200-jay-flowers
tier: draft
---

Pre-publication release gates do not guarantee a proxy-installable release. unbound-force/gaze's README documents that earlier /v2 tags were published with an incompatible module path, so `go install github.com/unbound-force/gaze/v2/cmd/gaze@latest` fails even though the offline gates (validate-module-major, validate-workflow-floor) and the org-infra preflight validate tag/module-path consistency before publication. Issue #297 targets the missing layer: a post-release smoke test that installs the exact published tag from proxy.golang.org using temporary GOBIN/module caches and asserts module path and version via `go version -m`, with bounded retries for proxy propagation. Triage corroboration: the divisor-sre release-pipeline integrity checklist already asks "Is there a smoke test or post-release verification step?" — this issue closes that gap for gaze. Note: CI/CD-only changes are exempt from the website documentation gate per AGENTS.md but still require spec-first workflow.
