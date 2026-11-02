# 🔍 Gaze CRAP Report

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
| GazeCRAPload | 6 (functions ≥ threshold 15) |

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
