---
tag: error-handling
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:31Z
identity: error-handling-20261110T012831-jay-flowers
tier: draft
---

A package-level sentinel error that names a CLI flag in its message silently assumes the flag surface of the entry point that authored it. Before shipping, enumerate every CLI command that can surface the error and verify the named remedy exists on each. In gaze PR #326, ErrNestedCoverageRun advises "pass a pre-generated --coverprofile instead" — valid for gaze crap and gaze report, but not for gaze self-check, which shares the same coverage pipeline yet registers no --coverprofile flag (verified in cmd/gaze/main.go newSelfCheckCmd). Prefer entry-point-neutral remedy text in shared internal sentinel errors, or wrap the sentinel with per-command context at the CLI layer.
