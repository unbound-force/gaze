---
tag: recursion-prevention
author: jay-flowers
category: pattern
created_at: 2026-11-10T01:28:48Z
identity: recursion-prevention-20261110T012848-jay-flowers
tier: draft
---

Recursion prevention for self-analyzing tooling is reliable only when enforced at the single choke point that spawns the nested subprocess, not when distributed as a skip convention every caller must remember. In gaze, GAZE_COVERAGE_RUN=1 was set on spawned `go test -coverprofile` processes, but protection depended on each self-analyzing test checking the variable; one test without the check (TestSC005_AnalysisPerformance) caused unbounded local `go test ./...` recursion. Moving the refusal into generateCoverProfile (returning ErrNestedCoverageRun before temp-profile creation or spawn) makes the one-level-deep invariant structural: forgotten skips, future tests, and user code invoking gaze from its own tests are all covered, and a forgotten skip now fails loudly inside the nested run instead of recursing silently. Precedent: openspec/changes/fix-nested-coverage-recursion design D1.
