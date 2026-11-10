---
tag: resilience
author: jay-flowers
category: pattern
created_at: 2026-11-10T01:28:33Z
identity: resilience-20261110T012833-jay-flowers
tier: draft
---

Recursion guards for self-invoking tooling are most robust when (1) the guard sits at the single subprocess-spawning choke point rather than relying on every caller or test to check a convention — gaze PR #326 moved the GAZE_COVERAGE_RUN check into generateCoverProfile, the only production site that spawns `go test -coverprofile` (verified by exec.Command sweep), so the chain is bounded at one level regardless of which caller forgets to check; and (2) the trigger is fail-safe: checking os.Getenv(...) != "" (any non-empty value refuses) means the guard can only fire more often, never be used to disable the safety layer. Refusal before temp-file creation and before the spawn also guarantees no resource leak on the refusal path, and wrapping the sentinel with %w at the Coverage boundary keeps errors.Is intact for callers.
