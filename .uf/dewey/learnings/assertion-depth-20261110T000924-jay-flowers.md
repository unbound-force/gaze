---
tag: assertion-depth
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:09:24Z
identity: assertion-depth-20261110T000924-jay-flowers
tier: draft
---

In the gaze repo, goprovider Coverage() tests validate results with a canonical per-result loop (File non-empty, Percentage within [0,100]) established by TestGoLineCoverageProvider_PreGeneratedProfile and TestCoverage_ShortWithCoverProfile in internal/provider/goprovider/goprovider_test.go. When a new test reuses a sibling's fixture and setup, it must mirror the sibling's assertion depth, not just its length check: when a spec scenario demands parity with another code path ("as it does outside a coverage collection"), the sibling test's assertion set is the encoded definition of that parity, and stopping at len(results) > 0 under-verifies it (go.md TC-009: assertions must verify specific expected values, not just err == nil or length checks). Review heuristic: for any new test that duplicates a sibling's arrange/act steps, diff the assert steps too — a missing assert block is a spec-parity gap even when the test passes.
