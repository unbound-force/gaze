---
tag: recursion-guard
author: jay-flowers
category: pattern
created_at: 2026-11-09T23:16:37Z
identity: recursion-guard-20261109T231637-jay-flowers
tier: draft
---

An env-var recursion guard delegated to consumer-side opt-in checks is only as reliable as the least careful consumer. In gaze, buildGoTestCmd set GAZE_COVERAGE_RUN=1 on the spawned `go test -coverprofile` subprocess and correctness relied on every self-analyzing test checking the variable and skipping itself; one unguarded test (TestSC005_AnalysisPerformance) turned a plain non-short `go test ./...` into unbounded subprocess recursion — the orphaned chain reached 23 processes and ~6.9 GB RSS in ~10 minutes
