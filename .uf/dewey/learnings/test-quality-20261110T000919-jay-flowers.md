---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:09:19Z
identity: test-quality-20261110T000919-jay-flowers
tier: draft
---

In internal/provider/goprovider tests, any test that exercises the pre-generated-profile path through Coverage() should mirror the established per-result validation loop used by TestGoLineCoverageProvider_PreGeneratedProfile and TestCoverage_ShortWithCoverProfile: for each result, assert r.File is non-empty and r.Percentage is within [0,100]. A bare len(results) > 0 check violates go.md TC-009 [MUST] and under-verifies the recovery path that ErrNestedCoverageRun directs users to (pass a pre-generated --coverprofile). When adding a sibling test with an identical fixture, copy the full assertion loop, not just the setup.
