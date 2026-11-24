---
tag: test-architecture
author: jay-flowers
category: gotcha
created_at: 2026-11-02T23:26:38Z
identity: test-architecture-20261102T232638-jay-flowers
tier: draft
---

When a testdata/golden path must be duplicated across Go packages (helpers in _test.go are not importable), every copy needs the documented-drift-guard treatment: a comment explaining the deliberate duplication plus a guard test that source-scans the canonical declaration. Observed in gaze PR 314 (spec 040): cmd/gaze/golden_paths_test.go guarded the second copy of internal/aireport's goldenDir with TestAireportGoldenDir_MatchesAireport, but a third copy added later in the same PR (internal/scaffold/scaffold_test.go, TestExampleReportMatchesGolden) shipped without the comment or guard. The copies added after the convention is established are the ones that miss it — sweep for all copies of a duplicated constant before closing a change that introduces a duplication guard.
