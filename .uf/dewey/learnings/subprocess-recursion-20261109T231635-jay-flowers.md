---
tag: subprocess-recursion
author: jay-flowers
category: pattern
created_at: 2026-11-09T23:16:35Z
identity: subprocess-recursion-20261109T231635-jay-flowers
tier: draft
---

Recursion guards for tools that spawn subprocesses belong at the spawn seam, not at entry points. When a tool spawns `go test -coverprofile` and marks the child with an env var (gaze: GAZE_COVERAGE_RUN), refusing generation when the marker is already present — inside the single function that creates the temp profile and spawns — bounds recursion at one level for every caller (existing tests, future tests, user code) without relying on each entry point remembering to check. Test-side skip guards under the sa
