---
tag: quality-external-analyzer
author: jay-flowers
category: pattern
created_at: 2026-08-31T22:12:30Z
identity: quality-external-analyzer-20260831T221230-jay-flowers
tier: draft
---

When using a fake analyzer binary in integration tests via the JSON-RPC protocol, the happy-path test for external quality analysis must go through `runQuality` with `format: "json"` and parse the resulting JSON. The output key is `quality_summary` (not `summary`) as defined by `qualityOutput` in `internal/quality/report.go`. The fake analyzer at `internal/protocol/testdata/fake_analyzer/main.go` returns exactly one test_mapping record (test_multiply → multiply:ReturnValue, confidence 80, target_package math_utils), which produces a quality report with 100% contract coverage when multiply has one contractual ReturnValue effect. The `moduleDir` temp dir only needs a `go.mod` file — it does not need real Go source files since the external analyzer provides the analysis data.
