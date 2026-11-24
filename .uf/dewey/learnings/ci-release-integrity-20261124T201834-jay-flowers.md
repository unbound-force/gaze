---
tag: ci-release-integrity
author: jay-flowers
category: gotcha
created_at: 2026-11-24T20:18:34Z
identity: ci-release-integrity-20261124T201834-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

An unanchored gitignore pattern that equals a source directory name silently ignores all NEW files under it even when existing files are already tracked (git never untracks, and direct `git status` won't surface them). A bare `gaze` pattern intended for the root build binary also matched `cmd/gaze/`, hiding a new `cmd/gaze/ci_workflow_test.go` contract test from the PR and CI. Anchor artifact patterns to the repo root (`/gaze`) or scope to the exact path, and verify new governance/test files with `git check-ignore -v <path>` (expect no output) as part of review.
