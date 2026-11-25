---
tag: ci-contract-testing
author: jay-flowers
category: pattern
created_at: 2026-11-24T23:27:51Z
identity: ci-contract-testing-20261124T232751-jay-flowers
tier: draft
as_of: "gaze@6675b56b89dea50273e85b513a60b545c5d2fac5"
---

Two-layer contract testing for GitHub Actions workflow YAML (precedent: cmd/gaze/ci_workflow_test.go, gaze PR #373, CIW-FR-006): (1) parse the workflow with minimal decode structs — a custom UnmarshalYAML on the needs field handles GitHub's scalar-vs-sequence duality — and pin wiring textually: gates, exact needs sets (sorted join comparison), SHA pins, timeouts, permissions, and the log-label strings the workflow MUST emit; (2) execute the embedded run script end-to-end by substituting the `${{ ... }}` expressions (failing the test on any residual unsubstituted `${{`), running it with bash against synthetic t.TempDir() git repositories, and parsing the GITHUB_OUTPUT file. Text pins catch wiring drift (including byte-identity between duplicated steps across workflows); execution pins the actual behavior — classification verdicts, fail-closed defaults, log listings — that text pins cannot. Executable scenario tables (learnings-only, mixed, renamed-into-path, symlink, empty diff, push event) plus a separately executed jq truth table for aggregate verdicts give spec-scenario-to-test traceability.
