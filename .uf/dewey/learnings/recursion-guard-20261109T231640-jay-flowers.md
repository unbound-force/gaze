---
tag: recursion-guard
author: jay-flowers
category: gotcha
created_at: 2026-11-09T23:16:40Z
identity: recursion-guard-20261109T231640-jay-flowers
tier: draft
---

An env-var recursion guard that depends on every spawning test opting in to a skip is a convention, not a constraint. fix-hardcoded-short-flag set GAZE_COVERAGE_RUN=1 on gaze's `go test -coverprofile` subprocesses and relied on each self-analyzing test checking the variable; the first test added without the check (TestSC005_AnalysisPerformance) made a non-short `go test ./...` recurse without bound (an orphaned chain reached 23 processes / ~7 GB RSS before manual kill). The durable fix (fix-nested-coverage-
