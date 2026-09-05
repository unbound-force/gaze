---
tag: fix-assertion-detection-confidence
author: jay-flowers
category: gotcha
created_at: 2026-09-02T19:14:29Z
identity: fix-assertion-detection-confidence-20260902T191429-jay-flowers
tier: draft
---

When extending the fake analyzer's test_mapping response (internal/protocol/testdata/fake_analyzer/main.go) from 1 to 3 mappings, two existing tests broke: TestDiscoverAndTestMapping in client_test.go (expected count 1→3) and TestContractCoverageProvider_WithTestMapping in adapter_test.go (divide contract coverage changed from 0% to 100% because the new mappings now cover both ReturnValue and ErrorReturn effects). Always search for all tests that depend on fake analyzer responses before extending them — the fake analyzer is a shared fixture used across both protocol and adapter test packages.
