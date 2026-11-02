# 🔍 Gaze Full Quality Report

Project: github.com/example/project · Branch: main
Gaze Version: v2.4.0 · Date: 2026-11-01

---

## 📊 CRAP Summary

| Metric | Value |
|--------|------:|
| Total functions analyzed | 12 |
| Average complexity | 7.3 |
| Average line coverage | 47.1% |
| Average CRAP score | 52.6 |
| CRAPload | 7 (functions ≥ threshold 15) |
| GazeCRAPload | ⚪ N/A |

### Top 5 Worst CRAP Scores

| Function | CRAP | Complexity | Coverage | Fix Strategy | Location |
|----------|-----:|-----------:|---------:|--------------|----------|
| `(*队列处理器).处理消息` | 342.0 | 18 | 0.0% | decompose_and_test | internal/队列/处理器.go:42 |
| `processGeneratedProtocolBufferMessageForTheExtremelyLongServiceNameprocessGeneratedProtocolBufferMessageForTheExtremelyLongServiceNameprocessGeneratedProtocolBufferMessageForTheExtremelyLongServiceNam` | 90.0 | 9 | 0.0% | add_tests | internal/codegen/generated.go:310 |
| ``parse`raw`Literal`` | 72.0 | 8 | 0.0% | add_tests | internal/lexer/literal.go:77 |
| `(*Parser).parse\|Pipe` | 48.0 | 16 | 50.0% | decompose | internal/parser/pipe.go:55 |
| `validateInput` | 22.5 | 10 | 50.0% | add_assertions | internal/api/validate.go:21 |

### GazeCRAP Quadrant Distribution

| Quadrant | Count | Meaning |
|----------|------:|---------|
| 🟢 Q1 — Safe | 4 | Low complexity, high contract coverage |
| 🟡 Q2 — Complex But Tested | 2 | High complexity, contracts verified |
| ⚪ Q3 — Needs Tests | 2 | Simple but underspecified |
| 🔴 Q4 — Dangerous | 4 | Complex and contracts not adequately verified |

### Remediation Breakdown

| Fix Strategy | Functions |
|--------------|----------:|
| decompose | 1 |
| add_tests | 4 |
| add_assertions | 1 |
| decompose_and_test | 1 |

---

## 🧪 Quality Summary

> ⚠️ Quality analysis unavailable: resolving packages for quality: go list ./...: exit status 1

---

## 🏷️ Classification Summary

| Classification | Count | % |
|----------------|------:|--:|
| Contractual | 4 | 26.7% |
| Ambiguous | 9 | 60.0% |
| Incidental | 2 | 13.3% |

---

## 🏥 Overall Health Assessment

### Summary Scorecard

| Dimension | Grade | Details |
|-----------|-------|---------|
| CRAPload | 🔴 F | 7/12 functions (58.3%) at or above threshold 15 |
| GazeCRAPload | ⚪ N/A | GazeCRAP unavailable |
| Avg Line Coverage | 🔴 D | 47.1% average line coverage |
| Contract Coverage | ⚪ N/A | analysis failed |
| Complexity | 🟡 C+ | 7.3 average cyclomatic complexity |

Documentation: 2 files scanned.

### Recommendations

1. 🔴 Add tests for `processGeneratedProtocolBufferMessageForTheExtremelyLongServiceNameprocessGeneratedProtocolBufferMessageForTheExtremelyLongServiceNameprocessGeneratedProtocolBufferMessageForTheExtremelyLongServiceNam` — 0.0% line coverage, complexity 9, CRAP 90.0 (internal/codegen/generated.go:310).
2. 🔴 Add tests for ``parse`raw`Literal`` — 0.0% line coverage, complexity 8, CRAP 72.0 (internal/lexer/literal.go:77).
3. 🔴 Add tests for `loadConfig` — 0.0% line coverage, complexity 4, CRAP 20.0 (internal/config/load.go:30).
4. 🟡 Add tests for `(*Handler).ServeHTTP` — 20.0% line coverage, complexity 5, CRAP 17.8 (internal/api/handler.go:163).
5. 🟡 Strengthen assertions for `validateInput` — tests execute it without verifying its contracts; GazeCRAP 110.0 (internal/api/validate.go:21).
6. 🟡 Resolve SSA degradation in 1 package(s) — quality and contract data are partial there.
7. 🟢 Map 2 skipped test(s) to targets with `gaze quality --target` — no target function could be inferred.
8. 🟢 Document contracts for ambiguous side effects — 9 of 15 effects (60.0%) are ambiguous.
