---
tag: ci-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-25T01:11:22Z
identity: ci-workflow-20261125T011122-jay-flowers
tier: draft
as_of: "gaze@0d56f0dcf27e2c0d0e59f8c1299d85a5fae96e4b"
---

When a Go contract test pins a shell script embedded in a GitHub Actions workflow, extract the script from the parsed workflow YAML and execute it directly — substitute GitHub expression placeholders, redirect GITHUB_OUTPUT to a temp file — against real temporary git repositories, so behavior is pinned by execution rather than substring matching. Reserve purely textual structural pins (first-line `set -euo pipefail`, regex-enforced command-capture form) for scenarios unreachable through real tooling — e.g., git never emits `..` path segments — and document the unreachability in a comment beside the pin. Precedent: gaze PR #373, TestCIWorkflow_ClassifyScriptBehavior in cmd/gaze/ci_workflow_test.go, which executes six classification scenarios against t.TempDir() git repos and keeps only the dotdot-segment scenario textual with a documented reason.
