---
tag: quality-external-analyzer
author: jay-flowers
category: context
created_at: 2026-08-31T22:12:44Z
identity: quality-external-analyzer-20260831T221244-jay-flowers
tier: draft
---

When adding a standalone exported function like `FetchTestMappings` that parallels an existing method (e.g., `ExternalContractCoverageProvider.fetchTestMappings`), the divisor-testing council will flag: (1) missing happy-path integration test, (2) zero direct coverage on the new function, (3) untested graceful degradation CLI handlers. Key fix: handler functions like `handleQualityNoTestMapping` and `handleQualityTestMappingError` can be unit tested by calling them directly with a synthetic `qualityParams` struct and `&adapter.Providers{AnalyzerName: "test-analyzer"}` — no subprocess needed. The `AnalyzerName` field is sufficient for the stderr warning messages, making these handlers independently testable without any fake binary setup.
