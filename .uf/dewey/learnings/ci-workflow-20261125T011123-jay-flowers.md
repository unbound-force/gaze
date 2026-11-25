---
tag: ci-workflow
author: jay-flowers
category: gotcha
created_at: 2026-11-25T01:11:23Z
identity: ci-workflow-20261125T011123-jay-flowers
tier: draft
as_of: "gaze@0d56f0dcf27e2c0d0e59f8c1299d85a5fae96e4b"
---

When a CI job's skip is transitive — job B declares `needs` on gated job A instead of carrying its own gate — pin the needs edge that provides the transitivity, not just the gate on A. An advisory-job allowlist that exempts B from gate assertions without also asserting B's `needs: A` edge lets a later edit remove the edge and silently un-skip B on gated PRs: B runs, consumes secrets, and fails on artifacts never produced by the skipped producer. Gate-only pins catch a broken `if:` condition; they do not catch a removed dependency edge upstream of it.
