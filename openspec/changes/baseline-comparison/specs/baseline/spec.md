## ADDED Requirements

### SC-001: Baseline Loading

`gaze crap` MUST auto-detect a baseline file at `.gaze/baseline.json` when it exists and is non-empty. When the file exists and is non-empty, comparison mode MUST be activated automatically. When the file does not exist or is empty (0 bytes), `gaze crap` MUST behave identically to today (no comparison, no error).

#### Scenario: Auto-detect baseline
- **GIVEN** a file exists at `.gaze/baseline.json` containing valid `gaze crap --format=json` output
- **WHEN** the user runs `gaze crap --coverprofile=coverage.out ./...`
- **THEN** the output MUST include comparison data (deltas, regressions, improvements)

#### Scenario: No baseline file
- **GIVEN** no file exists at `.gaze/baseline.json` and `--baseline` is not specified
- **WHEN** the user runs `gaze crap --coverprofile=coverage.out ./...`
- **THEN** the output MUST be identical to current behavior (no comparison sections, exit 0)

#### Scenario: Explicit baseline path
- **GIVEN** a valid baseline file exists at a custom path
- **WHEN** the user runs `gaze crap --baseline /custom/path/baseline.json ./...`
- **THEN** the comparison MUST use the specified file instead of the default path

#### Scenario: Explicit baseline path not found
- **GIVEN** no file exists at the specified path
- **WHEN** the user runs `gaze crap --baseline /custom/path/baseline.json ./...`
- **THEN** `gaze crap` MUST exit with an error indicating the baseline file was not found

#### Scenario: Empty auto-detected baseline
- **GIVEN** a file exists at `.gaze/baseline.json` but is 0 bytes (empty)
- **WHEN** the user runs `gaze crap --coverprofile=coverage.out ./...`
- **THEN** the output MUST be identical to current behavior (no comparison, no error) -- the empty file is silently skipped

### SC-002: Baseline File Format

The baseline file format MUST be the JSON output of `gaze crap --format=json`. No separate schema, no transformation required. The `LoadBaseline` function MUST accept an `io.Reader` and return a `*crap.Report`. `LoadBaseline` MUST use lenient JSON parsing (ignore unknown fields, use zero values for missing optional fields) to support baselines created by older or newer gaze versions.

#### Scenario: Round-trip compatibility
- **GIVEN** the user runs `gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json`
- **WHEN** the saved file is later loaded as a baseline
- **THEN** all `scores` entries MUST be parsed with `file`, `function`, `crap`, and `gaze_crap` fields preserved

#### Scenario: Invalid baseline JSON (explicit flag)
- **GIVEN** a file at a custom path containing invalid JSON
- **WHEN** the user runs `gaze crap --baseline /custom/path/baseline.json ./...`
- **THEN** `gaze crap` MUST exit with an error indicating the baseline file is malformed

#### Scenario: Invalid baseline JSON (auto-detected)
- **GIVEN** a file at `.gaze/baseline.json` containing invalid JSON
- **WHEN** the user runs `gaze crap --coverprofile=coverage.out ./...`
- **THEN** the auto-detected malformed file MUST be silently skipped (no comparison, no error)

#### Scenario: Baseline from older gaze version
- **GIVEN** a baseline created by a gaze version that did not include `gaze_crap`, `fix_strategy`, or `quadrant` fields
- **WHEN** the baseline is loaded
- **THEN** missing optional fields MUST default to zero/nil values and comparison MUST proceed normally (D5 handles nil `gaze_crap`)

### SC-003: Per-Function Comparison

The comparison logic MUST match functions between baseline and current results using a composite key of `file:function`. For each matched function, the comparison MUST compute CRAP and GazeCRAP deltas. A function MUST be classified as `improvement` when `crap_delta < -epsilon` or `gaze_crap_delta < -epsilon` (when baseline had coverage). When signals conflict (e.g., CRAP regresses but GazeCRAP improves), `regression` MUST take precedence -- the function is classified as `regression`.

#### Scenario: CRAP regression detected
- **GIVEN** a baseline where function `Analyze` in `internal/crap/analyze.go` has CRAP score 9.2
- **WHEN** the current run produces CRAP score 12.5 for the same function (delta 3.3, exceeding epsilon 0.5)
- **THEN** the function MUST be classified as `regression` in the comparison output

#### Scenario: CRAP improvement detected
- **GIVEN** a baseline where function `Analyze` has CRAP score 12.5
- **WHEN** the current run produces CRAP score 8.0 (crap_delta = -4.5, which is < -epsilon)
- **THEN** the function MUST be classified as `improvement`

#### Scenario: Score within epsilon tolerance
- **GIVEN** a baseline where function `Analyze` has CRAP score 9.2 and epsilon is 0.5
- **WHEN** the current run produces CRAP score 9.5 (delta 0.3, within epsilon)
- **THEN** the function MUST be classified as `unchanged`

#### Scenario: GazeCRAP regression when baseline had coverage
- **GIVEN** a baseline where function `Analyze` has `gaze_crap` 14.1 (non-nil, non-zero)
- **WHEN** the current run produces `gaze_crap` 18.3 (delta 4.2, exceeding epsilon)
- **THEN** the function MUST be classified as `regression`

#### Scenario: GazeCRAP not evaluated when baseline had no coverage
- **GIVEN** a baseline where function `NewFunc` has `gaze_crap` nil or 0
- **WHEN** the current run produces `gaze_crap` 25.0
- **THEN** the GazeCRAP delta MUST NOT trigger a regression (the function's status depends only on CRAP delta)

### SC-004: New Function Detection

Functions present in the current results but absent from the baseline MUST be classified as `new`. A new function with a CRAP score exceeding the `new_function_threshold` (default 30) MUST be classified as `new_violation`.

#### Scenario: New function below threshold
- **GIVEN** a function `helperFunc` exists in current results but not in the baseline
- **WHEN** its CRAP score is 12.0 (below threshold 30)
- **THEN** it MUST be classified as `new` (informational, not a failure)

#### Scenario: New function above threshold
- **GIVEN** a function `complexFunc` exists in current results but not in the baseline
- **WHEN** its CRAP score is 42.0 (above threshold 30)
- **THEN** it MUST be classified as `new_violation` (counts as a failure)

### SC-005: Removed Function Detection

Functions present in the baseline but absent from the current results MUST be reported as `removed`. Removed functions MUST NOT count as regressions or violations.

#### Scenario: Function removed
- **GIVEN** a function `oldFunc` exists in the baseline but not in the current results
- **WHEN** the comparison is performed
- **THEN** the function MUST appear in the `removed_functions` section with its baseline scores
- **AND** the comparison MUST still pass (removed is informational)

### SC-006: Pass/Fail Gate

The comparison MUST produce a pass/fail determination. The comparison passes when there are zero regressions AND zero new-function violations. `gaze crap` MUST exit with code 1 when the comparison fails. This is a second independent exit-1 gate alongside the existing `--max-crapload` / `--max-gaze-crapload` threshold gate. Both gates are evaluated independently -- exit 1 if either fails. Comparison output MUST always be written regardless of threshold results.

#### Scenario: Comparison passes
- **GIVEN** the comparison finds 0 regressions, 3 improvements, 2 new functions (all below threshold), 1 removed function
- **WHEN** the pass/fail is determined
- **THEN** the result MUST be `passed: true` and exit code MUST be 0

#### Scenario: Comparison fails on regression
- **GIVEN** the comparison finds 1 regression
- **WHEN** the pass/fail is determined
- **THEN** the result MUST be `passed: false` and exit code MUST be 1

#### Scenario: Comparison fails on new violation
- **GIVEN** the comparison finds 0 regressions but 1 new-function violation
- **WHEN** the pass/fail is determined
- **THEN** the result MUST be `passed: false` and exit code MUST be 1

### SC-007: JSON Comparison Output

When `--format=json` is used with an active baseline, the JSON output MUST be produced by `WriteComparisonJSON` (separate from `WriteJSON`) and MUST include comparison-specific fields on each score and top-level comparison sections. The `crap.Score` type is NOT modified; the JSON is assembled by the comparison writer from `FunctionDelta` data.

Per-score fields in JSON output (assembled from `FunctionDelta`, all `omitempty`):
- `baseline_crap` (`*float64`): CRAP score from baseline
- `baseline_gaze_crap` (`*float64`): GazeCRAP score from baseline
- `crap_delta` (`*float64`): current CRAP minus baseline CRAP
- `gaze_crap_delta` (`*float64`): current GazeCRAP minus baseline GazeCRAP
- `status` (`string`): one of `regression`, `improvement`, `unchanged`

Top-level sections:
- `new_functions` (`[]object`): functions not in baseline, with `status` (`new` or `new_violation`)
- `removed_functions` (`[]object`): functions in baseline but not current, with baseline scores
- `comparison` (`object`): summary counts and pass/fail

#### Scenario: JSON output with baseline
- **GIVEN** an active baseline
- **WHEN** the user runs `gaze crap --format=json ./...`
- **THEN** each score object MUST include `baseline_crap`, `crap_delta`, and `status` fields
- **AND** the output MUST include `new_functions`, `removed_functions`, and `comparison` top-level keys

#### Scenario: JSON output without baseline
- **GIVEN** no baseline is loaded
- **WHEN** the user runs `gaze crap --format=json ./...`
- **THEN** the output MUST be produced by the normal `WriteJSON` path with zero comparison-specific fields or sections

### SC-008: Text Comparison Output

When `--format=text` (default) is used with an active baseline, the text output MUST append comparison sections after the normal CRAP report: comparison summary, regressions table, improvements table, new functions list, and removed functions list.

#### Scenario: Text output with regressions
- **GIVEN** an active baseline with 2 regressions and 3 improvements
- **WHEN** the user runs `gaze crap ./...`
- **THEN** the text output MUST include a "Regressions" section listing the 2 functions with their baseline, current, and delta scores
- **AND** an "Improvements" section listing the 3 functions

### SC-009: Configuration

The `.gaze.yaml` file MUST support a `baseline` section with the following fields:

- `file` (string, default `.gaze/baseline.json`): path to the baseline file
- `epsilon` (float64, default 0.5): minimum score delta to trigger regression/improvement
- `new_function_threshold` (float64, default 30): CRAP score above which a new function is a violation

Epsilon MUST be >= 0. `new_function_threshold` MUST be > 0. Invalid values MUST produce a clear error message.

The CLI flag `--baseline` MUST override the config file path. Epsilon and new-function threshold SHOULD be configurable only via `.gaze.yaml` (no CLI flags needed for values that rarely change).

#### Scenario: Config overrides defaults
- **GIVEN** a `.gaze.yaml` with `baseline.epsilon: 1.0` and `baseline.new_function_threshold: 20`
- **WHEN** the comparison is performed
- **THEN** epsilon MUST be 1.0 and new-function threshold MUST be 20

#### Scenario: Config file absent
- **GIVEN** no `.gaze.yaml` exists
- **WHEN** the comparison is performed
- **THEN** epsilon MUST be 0.5 and new-function threshold MUST be 30

#### Scenario: Invalid config values
- **GIVEN** a `.gaze.yaml` with `baseline.epsilon: -1`
- **WHEN** `gaze crap` loads the config
- **THEN** `gaze crap` MUST exit with an error indicating epsilon must be >= 0

### Function Status Values

The `FunctionStatus` type enumerates all valid status values used in comparison output:

| Value | Where Used | Meaning |
|-------|-----------|---------|
| `regression` | `scores[].status` | CRAP or GazeCRAP increased beyond epsilon |
| `improvement` | `scores[].status` | CRAP or GazeCRAP decreased beyond epsilon |
| `unchanged` | `scores[].status` | Score delta within epsilon tolerance |
| `new` | `new_functions[].status` | Function not in baseline, CRAP below threshold |
| `new_violation` | `new_functions[].status` | Function not in baseline, CRAP above threshold |
| `removed` | `removed_functions[].status` | Function in baseline but not in current results |

### Coverage Strategy

New code in `internal/crap/compare.go` and `internal/crap/compare_report.go` MUST achieve >= 90% line coverage via unit tests. All branches in `Compare()` MUST be exercised (the test list in task 3.5 is exhaustive, not illustrative). Integration tests (tasks 6.1-6.4) read local JSON fixtures and run pure functions -- they do NOT spawn subprocesses and complete in milliseconds, so they do NOT use `testing.Short()` guards.

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
