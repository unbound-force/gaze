---
tag: ci-cd
author: jay-flowers
category: reference
created_at: 2026-11-25T23:30:01Z
identity: ci-cd-20261125T233001-jay-flowers
tier: draft
as_of: "gaze@a286142e8838a0c84a4780508168eb48c1fbcf04"
---

actions/checkout v4 → v7 needs no workflow-side migration for the simple checkout patterns used in gaze's workflows when running on GitHub-hosted runners: v5 moves the action to Node 24 (runner support is the only requirement; ubuntu-latest is fine), v6.0.0 changes credential persistence to a separate file while keeping `persist-credentials: false` and later-step `git push` working, and v7.0.0 only newly blocks fork-PR checkout under `pull_request_target`/`workflow_run` (plain `pull_request`, `workflow_dispatch`, `workflow_call` are unaffected) and switches the action to ESM internally. Verification for a bump PR: confirm no changed workflow checks out under pull_request_target/workflow_run, confirm all runners are GitHub-hosted (node24), and confirm any job that pushes (e.g. a git-auto-commit step) on the next lint-fix PR. The org's current pinned target is v7.0.1 SHA 3d3c42e5aac5ba805825da76410c181273ba90b1.
