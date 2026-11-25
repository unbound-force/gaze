---
tag: external-analyzer
author: jay-flowers
category: pattern
created_at: 2026-11-25T05:16:00Z
identity: external-analyzer-20261125T051600-jay-flowers
tier: draft
as_of: "gaze@de03232a61c934ca9570f749d78b4f7a6a4d2bc9"
---

Gaze command-parity dispatch fixes (issue #282 class — a sibling runner ignoring --language alone) are verified with one hermetic sentinel test per command: run the runner from t.TempDir() with languageFlag "zz-nodiscover-test-language", then assert the returned error contains "no analyzer found for language" AND does not contain "finding module root". The paired assertions discriminate both gate widenings at once — module-dir resolution (resolveReportModuleDir, which would fail with the module-root error) and analyze-func wiring (applyReportAnalyzeFunc, whose omission drops the run to the Go-native default and cannot produce the discovery error) — so a revert of either half fails the test. For report-family commands, pass format "json": buildReportAdapter returns nil for non-text formats, making analyzer discovery the only failure point (text format would fail on the AI adapter first). Before widening a gate that sits ahead of an injected test seam, grep the test suite for the newly honored flag — no injected-analyzeFunc report test sets languageFlag, so applyReportAnalyzeFunc's external-gate-before-seam ordering stays safe. Remaining application of this pattern: gaze analyze's --language dispatch (quality-classification-counts D12 deferral).
