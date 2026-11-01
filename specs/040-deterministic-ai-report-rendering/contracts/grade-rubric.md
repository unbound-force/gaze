# Contract: Grade rubric, severity, deterministic recommendations

Covers FR-007 through FR-011 and SC-004. `rubric_version: "1"`.
The rubric is published in `docs/reference/cli/report.md`, section
"Grading rubric". A test parses that docs table and asserts it equals
the code constants (Layer 2 seam).

## Graded input

Every graded value is the **displayed** value:

- Floats (avg line coverage, avg complexity) are formatted with
  `strconv.FormatFloat(v, 'f', 1, 64)` and parsed back to tenths.
- Ratios (CRAPload, GazeCRAPload) use integer tenths of a percent:
  `tenths = (2*n*1000 + total) / (2*total)`, where `total =
  crap.summary.total_functions`. The ratio is displayed as
  `tenths/10` with one decimal.
- Contract coverage is the integer `ReportSummary.AvgContractCoverage`
  (the same value `--min-contract-coverage` gates on).

Boundaries are inclusive. A dimension receives the first grade whose
bound it satisfies, scanning from A downward, and F otherwise.

| Grade | CRAPload % ≤ | GazeCRAPload % ≤ | Avg Line Coverage % ≥ | Contract Coverage % ≥ | Avg Complexity ≤ |
|-------|-----:|-----:|-----:|-----:|-----:|
| A  | 2.0  | 2.0  | 90.0 | 80 | 3.0 |
| A- | 4.0  | 4.0  | 85.0 | 70 | 4.0 |
| B+ | 6.0  | 6.0  | 80.0 | 60 | 5.0 |
| B  | 8.0  | 8.0  | 75.0 | 50 | 6.0 |
| B- | 10.0 | 10.0 | 70.0 | 40 | 7.0 |
| C+ | 12.5 | 12.5 | 65.0 | 30 | 8.0 |
| C  | 15.0 | 15.0 | 60.0 | 20 | 9.0 |
| C- | 20.0 | 20.0 | 55.0 | 15 | 10.0 |
| D  | 30.0 | 30.0 | 45.0 | 10 | 12.0 |
| F  | > 30.0 | > 30.0 | < 45.0 | < 10 | > 12.0 |

## Unavailable (⚪ N/A, `grade: null`, `value: null`)

| Dimension | N/A when |
|-----------|----------|
| CRAPload, Avg Line Coverage, Complexity | CRAP step not selected, CRAP step failed, or `total_functions == 0` |
| GazeCRAPload | as above, or `summary.gaze_crapload` absent |
| Contract Coverage | Quality step not selected or failed, `AvgContractCoverage == nil`, or `quality_summary.total_tests == 0` |

## Grade → severity → marker (FR-008)

| Grades | Severity | Marker |
|--------|----------|--------|
| A, A-, B+ | `low` | 🟢 |
| B, B-, C+, C | `moderate` | 🟡 |
| C-, D, F | `critical` | 🔴 |
| N/A | `unavailable` | ⚪ |

## Scorecard detail text (exact templates)

| Dimension | Detail |
|-----------|--------|
| CRAPload | `{n}/{total} functions ({r}%) at or above threshold {t}` |
| GazeCRAPload | `{n}/{total} functions ({r}%) at or above threshold {gt}` |
| Avg Line Coverage | `{v}% average line coverage` |
| Contract Coverage | `{v}% average contract coverage across {tests} tests` |
| Complexity | `{v} average cyclomatic complexity` |
| any N/A | `{reason}` — one of `not selected`, `analysis failed`, `no functions analyzed`, `GazeCRAP unavailable`, `no tests analyzed` |

`{t}` / `{gt}` are `crap_threshold` / `gaze_crap_threshold`
formatted with `%g`. Values are taken verbatim from the analysis
data (FR-006).

## Deterministic recommendations (FR-009)

**Part 1, fix strategies.** Take the first 5 entries of
`crap.summary.recommended_actions` in their given order. They are
never re-ranked (FR-006). Each entry is joined to its `crap.scores[]`
entry by (package, function, file, line) to obtain `line_coverage`.

| Condition (first match) | Severity |
|-------------------------|----------|
| `line_coverage == 0` or `quadrant == Q4_Dangerous` | critical 🔴 |
| otherwise | moderate 🟡 |

| fix_strategy | Text template |
|--------------|---------------|
| `add_tests` | ``Add tests for `{fn}` — {cov}% line coverage, complexity {cc}, CRAP {crap} ({file}:{line}).`` |
| `add_assertions` | ``Strengthen assertions for `{fn}` — tests execute it without verifying its contracts{; GazeCRAP {g}} ({file}:{line}).`` |
| `decompose` | ``Decompose `{fn}` — complexity {cc} keeps CRAP at {crap} even at full coverage ({file}:{line}).`` |
| `decompose_and_test` | ``Decompose `{fn}`, then add tests — complexity {cc}, {cov}% line coverage, CRAP {crap} ({file}:{line}).`` |

**Part 2, data conditions**, appended in this order:

| Source | Condition | Severity | Text |
|--------|-----------|----------|------|
| `ssa_degraded` | `Summary.SSADegraded` | moderate 🟡 | `Resolve SSA degradation in {k} package(s) — quality and contract data are partial there.` |
| `skipped_tests` | `Summary.SkippedTests > 0` | low 🟢 | ``Map {k} skipped test(s) to targets with `gaze quality --target` — no target function could be inferred.`` |
| `ambiguous_classification` | classify available and `Ambiguous > Contractual + Incidental` | low 🟢 | `Document contracts for ambiguous side effects — {a} of {total} effects ({p}%) are ambiguous.` |

That gives at most 8 items. If there are none, the section prints
`No recommendations — no functions at or above the CRAP threshold.`

## Test obligations (SC-004)

- Table-driven: for every boundary in every column, one case exactly
  on the bound and one case one tenth (or one point) past it.
- N/A matrix: each N/A condition.
- Ratio rounding: n/total pairs whose exact ratio sits on either side
  of `x.x5`.
- Severity map: all 10 grades plus N/A.
- Recommendation templates: one case per strategy and per condition,
  plus the cap and ordering.
