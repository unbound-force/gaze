---
tag: assertion-depth
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:09:22Z
identity: assertion-depth-20261110T000922-jay-flowers
tier: draft
---

In gaze's internal/provider/goprovider package, the established pattern for happy-path Coverage tests that write a pre-generated coverprofile fixture is three-part: assert err == nil, assert len(results) > 0, then loop over results asserting r.File != "" and r.Percentage within [0, 100] (see TestGoLineCoverageProvider_PreGeneratedProfile and TestCoverage_ShortWithCoverProfile in goprovider_test.go — the latter's comment states the loop confirms a flag "doesn't alter profile parsing"). Any new test verifying that a flag or environment variable (e.g., GAZE_COVERAGE_RUN) does not alter the profile-parsing path must mirror that loop; stopping at err == nil + len > 0 violates go.md TC-009 [MUST], which literally names that prohibited pair and requires asserting slice contents. When the fixture is a single coverage block referencing a real function (e.g., Formula at internal/crap/crap.go:245, count 1), exact-value assertions are also cheap and stronger: len(results) == 1, FuncName == "Formula", Percentage == 100.0.
