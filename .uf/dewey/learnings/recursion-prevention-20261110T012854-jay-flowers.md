---
tag: recursion-prevention
author: jay-flowers
category: pattern
created_at: 2026-11-10T01:28:54Z
identity: recursion-prevention-20261110T012854-jay-flowers
tier: draft
---

Recursion prevention that relies on every self-analyzing test remembering to check a guard variable fails open: one forgotten skip (TestSC005_AnalysisPerformance) made a plain `go test ./...` recurse without bound, gaining a process level roughly once per minute until manually killed. Enforcing the guard at the single choke point that spawns the subprocess (generateCoverProfile returning ErrNestedCoverageRun when GAZE_COVERAGE_RUN is non-empty) fails closed — the chain can never exceed one level regardless of which test or user code requests coverage — while per-test skip guards remain a fast-path optimization (keeping the nested run clean and cheap), not the safety mechanism.
