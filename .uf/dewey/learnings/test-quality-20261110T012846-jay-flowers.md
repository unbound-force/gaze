---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T01:28:46Z
identity: test-quality-20261110T012846-jay-flowers
tier: draft
---

To verify "no subprocess was started" in a guard test without observing process creation, clear PATH (t.Setenv("PATH", "")) so that removing the guard produces an immediate exec.LookPath failure instead of the sentinel error. The test then differentiates "guard fired" from "spawn failed" purely by error identity (errors.Is), with no mocks, no process inspection, and no timing. Used by gaze's TestGenerateCoverProfile_NestedRunRefused and TestGenerateCoverProfile_ProceedsWhenUnset to cover both directions of the GAZE_COVERAGE_RUN guard.
