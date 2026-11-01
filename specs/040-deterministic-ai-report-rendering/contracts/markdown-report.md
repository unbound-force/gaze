# Contract: Gaze-rendered markdown report

Covers FR-001 through FR-006, FR-017, FR-020, FR-025, and the
edge cases for wide tables and partial failures. The **golden files
are the normative byte-level contract**. This document describes
their structure.

## Golden files (shared by renderer and CLI tests)

Location: `internal/aireport/testdata/golden/`

| File | Payload | AI outcome | Sections |
|------|---------|-----------|----------|
| `payload-full.json` | full fixture (all steps succeed, all metadata fixed) | — | input |
| `payload-partial.json` | quality step error, GazeCRAPload absent | — | input |
| `report-full.md` | full | none (markdown format) | all |
| `report-full-ai.md` | full | valid answer A | all |
| `report-full-rejected.md` | full | rejection (`summary_length`) | all |
| `report-crap.md` | full | none | crap |
| `report-quality.md` | full | none | quality |
| `report-partial.md` | partial | none | all |

Tests regenerate the files with `go test ./internal/aireport -run Golden -update`.
The `-update` flag is test-only. CI never passes it.

## Layout (all sections, AI present)

```markdown
# 🔍 Gaze Full Quality Report

Project: github.com/example/project · Branch: main
Gaze Version: v2.4.0 · Date: 2026-11-01

<!-- gaze:ai-summary -->
### Summary

<validated AI summary paragraphs>
<!-- /gaze:ai-summary -->

---

## 📊 CRAP Summary

| Metric | Value |
|--------|------:|
| Total functions analyzed | 216 |
| Average complexity | 6.2 |
| Average line coverage | 79.0% |
| Average CRAP score | 7.7 |
| CRAPload | 24 (functions ≥ threshold 15) |
| GazeCRAPload | 4 (functions ≥ threshold 15) |

### Top 5 Worst CRAP Scores

| Function | CRAP | Complexity | Coverage | Fix Strategy | Location |
|----------|-----:|-----------:|---------:|--------------|----------|
| `(*Handler).ServeHTTP` | 42.0 | 6 | 0.0% | add_tests | internal/api/handler.go:163 |

### GazeCRAP Quadrant Distribution

| Quadrant | Count | Meaning |
|----------|------:|---------|
| 🟢 Q1 — Safe | 29 | Low complexity, high contract coverage |
| 🟡 Q2 — Complex But Tested | 1 | High complexity, contracts verified |
| ⚪ Q3 — Needs Tests | 0 | Simple but underspecified |
| 🔴 Q4 — Dangerous | 4 | Complex and contracts not adequately verified |

### Remediation Breakdown

| Fix Strategy | Functions |
|--------------|----------:|
| decompose | 4 |
| add_tests | 12 |
| add_assertions | 6 |
| decompose_and_test | 2 |

---

## 🧪 Quality Summary

| Metric | Value |
|--------|------:|
| Tests analyzed | 140 |
| Average contract coverage | 31% |
| Over-specifications | 3 |
| Assertion detection confidence | 92% |
| Skipped tests | 0 |

### Worst Tests by Contract Coverage

| Test | Target | Contract Coverage | Gaps |
|------|--------|------------------:|-----:|
| `TestServe` | `(*Handler).ServeHTTP` | 0.0% | 3 |

### Coverage Gaps in Worst Tests

| Target | Effect | Description | Location |
|--------|--------|-------------|----------|
| `(*Handler).ServeHTTP` | ReturnValue | returns error | internal/api/handler.go:170 |

---

## 🏷️ Classification Summary

| Classification | Count | % |
|----------------|------:|--:|
| Contractual | 73 | 31.3% |
| Ambiguous | 155 | 66.5% |
| Incidental | 8 | 3.4% |

---

## 🏥 Overall Health Assessment

### Summary Scorecard

| Dimension | Grade | Details |
|-----------|-------|---------|
| CRAPload | 🟡 C+ | 24/216 functions (11.1%) at or above threshold 15 |
| GazeCRAPload | 🟢 A | 4/216 functions (1.9%) at or above threshold 15 |
| Avg Line Coverage | 🟡 B | 79.0% average line coverage |
| Contract Coverage | 🟡 C+ | 31% average contract coverage across 140 tests |
| Complexity | 🟡 B- | 6.2 average cyclomatic complexity |

Documentation: 12 files scanned.

### Recommendations

1. 🔴 Add tests for `processQueue` — 0.0% line coverage, complexity 8, CRAP 72.0 (internal/worker/queue.go:89).
2. 🟡 Decompose `validateInput` — complexity 16 keeps CRAP at 22.4 even at full coverage (internal/api/validate.go:55).

<!-- gaze:ai-recommendations -->
### AI-Prioritized Recommendations

1. <validated AI recommendation>
<!-- /gaze:ai-recommendations -->
```

The values above are illustrative. The golden files fix the actual
bytes.

## Rules

1. **Title** depends on the section set. All four sections: `🔍 Gaze
   Full Quality Report`. `{crap}` alone: `🔍 Gaze CRAP Report`.
   `{quality}` alone: `🔍 Gaze Quality Report`. Any other set:
   `🔍 Gaze Quality Report (sections: a, b)`, with sections in
   canonical order.
2. **Metadata.** `Branch` is omitted (with its ` · ` separator) when
   empty. Values come only from `payload.metadata`.
3. **Section order** is fixed: crap, quality, classification, health.
   Only selected sections render. `---` separates the header and each
   section.
4. **Closed vocabulary (FR-002).** In Gaze-rendered bytes, the only
   emoji are 🔍 📊 🧪 🏷️ 🏥 🟢 🟡 🔴 ⚪ ⚠️. A test scans the
   rendered output outside the AI slots for any other emoji code point
   and fails if it finds one.
5. **Failed step (FR-004).** The section body becomes
   `> ⚠️ <Step> analysis unavailable: <first line of error, ≤ 200 bytes>`.
   The heading is kept.
6. **Unavailable metric (FR-005).** The cell shows `⚪ N/A`. No proxy
   value is computed.
7. **Other callouts.** SSA degraded:
   `> ⚠️ SSA analysis degraded for {k} package(s): {list}` (quality
   section). Zero tests:
   `> ⚠️ Quality analysis found 0 tests for the selected packages.`
   No docs: `Documentation: no files found.`, rendered as a plain line,
   not an emoji callout.
8. **Tables.** Function names and test names are code spans. Cells
   escape `|` as `\|` and newlines as spaces. A name containing a
   backtick uses a double-backtick span. Names are never truncated.
9. **Numbers.** Floats use one decimal (`%.1f`). Percentages carry a
   `%` suffix. Counts are integers. Values are verbatim from the
   analysis data (FR-006), and rows follow the data's order:
   `worst_crap`, then `worst_coverage_tests`; gaps follow per-test
   order, capped at 10.
10. **AI slots (FR-017).** These are the only places AI text appears.
    With no outcome they are omitted entirely. The summary slot sits
    after the metadata. The recommendations slot follows the
    deterministic recommendations, or sits at the end of the report
    when `health` is not selected. A failure outcome renders the
    summary slot as `> ⚠️ AI narrative rejected: <detail>` or
    `> ⚠️ AI narrative unavailable: <detail>`, and no recommendations
    slot.
11. **Trailing newline.** The output ends with exactly one `\n`.

## Test obligations

- **Golden (SC-003):** each golden file is checked at renderer level
  (`RenderMarkdown`) and at CLI level (`runReport` with an injected
  `analyzeFunc` and a fixed clock). The CLI-level check covers the
  full, AI, and rejected outcomes.
- **Byte identity (SC-002):** render `payload-full.json` with three
  different valid answers and with no AI, strip the slots, and assert
  all four strings are equal.
- **Partial (US2 AS-2/3):** `report-partial.md`.
- **Unit tests:** cell escaping (pipe, backtick, newline, CJK name,
  200-rune name), title per section set, omission of an unselected
  section, the vocabulary scan, and rejection-callout placement.
