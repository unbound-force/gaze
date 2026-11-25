---
tag: dependency-automation
author: jay-flowers
category: gotcha
created_at: 2026-11-25T19:41:37Z
identity: dependency-automation-20261125T194137-jay-flowers
tier: draft
as_of: "gaze@80aec840ff855b2d2ddd5f3f21636c3257a4c388"
---

Adding a Dependabot `github-actions` (and `gomod`) ecosystem entry creates automated pin-bump PRs that collide with any Go test asserting a literal action or reusable-workflow SHA. Observed on gaze PR 389 (deps-automerge): `cmd/gaze/ci_workflow_test.go` asserts `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683` for the `changes` job, and `cmd/gaze/release_workflow_test.go` asserts the exact `jflowers/ship/.github/workflows/go-release.yml@<sha> # vX.Y.Z` pin. A Dependabot bump changes the SHA, the guard test fails, the required CI aggregate (`CI OK`) goes red, and an auto-merge gate conditioned on required checks never merges. Dependabot cannot update Go test constants, so "pick up each pin bump automatically" fails until a human edits the test — and because Dependabot does update reusable-workflow `uses:` references, a grouped `jflowers/ship*` PR containing the guarded release pipeline pin will never self-merge. Mitigation: assert pin *format* (workflow path + full 40-hex SHA + semver comment) instead of a literal SHA, which still enforces CI-001 (no mutable tags) while coexisting with Dependabot; or explicitly document the manual lockstep.
