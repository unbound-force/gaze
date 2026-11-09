---
tag: subprocess-recursion
author: jay-flowers
category: pattern
created_at: 2026-11-09T23:16:34Z
identity: subprocess-recursion-20261109T231634-jay-flowers
tier: draft
---

Recursion guards for tool-spawning subprocesses must be enforced at the single spawn site, not delegated to convention (every potentially-recursive test remembering to check a guard variable). Gaze's GAZE_COVERAGE_RUN=1 marker originally relied on each self-analyzing test checking it and skipping; one test without the check (TestSC005_AnalysisPerformance) turned a local non-short `go test ./...` into an unbounded process chain (23 processes / ~6.9 GB RSS / ~15k orphaned go-build temp dirs observed). The fix
