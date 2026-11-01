---
tag: branch-coverage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:49:15Z
identity: branch-coverage-20261101T174915-jay-flowers
tier: draft
---

Go branch coverage cannot be fully derived from `go test -coverprofile` data: profile ranges approximate if/else and switch/select case-arm execution (each arm is a statement block with its own range), but they cannot observe the implicit false arm of an else-less `if` (no statements carry a hit count) or short-circuit `&&`/`||` right-operand evaluation (sub-statement granularity). Exact coverage of those constructs requires custom instrumentation (source rewriting or build hooks). When triaging or specifying branch-coverage requests (gaze issue #271 pattern): (1) scope acceptance criteria by mechanism — statement-level branch arms are measurable via coverprofile cross-referencing, short-circuit arms only via instrumentation; (2) instrumentation-based measurement conflicts with the established `--coverprofile`-reuse CI pattern (gaze reuses a pre-generated profile and never runs the host tests itself, spec 020) and with `testing.Short()`-guarded suites, so the spec must state where instrumented runs happen and how the gate works in profile-reuse mode; (3) replace "performance impact acceptable" with a numeric bound before it becomes an acceptance criterion.
