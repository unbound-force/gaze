---
tag: go-testing
author: jay-flowers
category: gotcha
created_at: 2026-11-24T23:27:52Z
identity: go-testing-20261124T232752-jay-flowers
tier: draft
as_of: "gaze@6675b56b89dea50273e85b513a60b545c5d2fac5"
---

Go's os/exec dedups cmd.Env keeping the LAST entry per duplicate key, so `cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+outPath)` reliably overrides an inherited GITHUB_OUTPUT when a test suite runs inside a GitHub Actions runner (where GITHUB_OUTPUT is always set for the step). Without that dedup-last-wins guarantee, the runner's own GITHUB_OUTPUT could shadow the test's capture file and the assertions would read an empty result. Subprocess-based tests that override CI-runner-provided environment variables should rely on this append-last pattern (or filter explicitly) rather than assuming first-match semantics.
