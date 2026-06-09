## 1. Configuration

- [x] 1.1 Add `BaselineConfig` struct to `internal/config/config.go` with fields: `File string` (default `.gaze/baseline.json`), `Epsilon float64` (default 0.5), `NewFunctionThreshold float64` (default 30). Add `Baseline BaselineConfig` field to `GazeConfig`. Update `DefaultConfig()`. Add validation: epsilon >= 0, new_function_threshold > 0.
- [x] 1.2 Add unit tests for baseline config loading: default values when omitted, override when specified, round-trip YAML marshal/unmarshal, invalid values (negative epsilon, zero threshold). Add test fixture YAML files under `internal/config/testdata/`.
- [x] 1.3 Update `docs/reference/configuration.md` with the `baseline` section: field descriptions, types, defaults, valid ranges, and complete example YAML.

## 2. Comparison Data Model

- [x] 2.1 Add comparison types to `internal/crap/crap.go`: `ComparisonResult` (embeds or wraps `*Report`, plus `Deltas []FunctionDelta`, `NewFunctions []Score`, `RemovedFunctions []Score`, `ComparisonSummary`), `FunctionDelta` (baseline `Score`, current `Score`, `CRAPDelta float64`, `GazeCRAPDelta *float64`, `Status FunctionStatus`), `ComparisonSummary` (regression/improvement/new/new-violation/removed/unchanged counts, epsilon, threshold, passed bool). All fields with `json` tags. `crap.Score` is NOT modified -- comparison data lives only in `FunctionDelta` and `ComparisonResult`.
- [x] 2.2 Add `FunctionStatus` string type with constants: `StatusRegression`, `StatusImprovement`, `StatusUnchanged`, `StatusNew`, `StatusNewViolation`, `StatusRemoved`.

## 3. Comparison Logic

- [x] 3.1 Create `internal/crap/compare.go` with `LoadBaseline(r io.Reader) (*Report, error)` -- deserialize a `crap.Report` from JSON using lenient parsing (ignore unknown fields). Return error on malformed input.
- [x] 3.2 Add `CompareOptions` struct to `compare.go`: `Epsilon float64`, `NewFunctionThreshold float64`.
- [x] 3.3 Implement `Compare(baseline *Report, current *Report, opts CompareOptions) *ComparisonResult` -- build lookup maps by `file:function` key, iterate current scores computing deltas, detect regressions (CRAP delta > epsilon, GazeCRAP delta > epsilon when baseline had coverage), improvements (crap_delta < -epsilon), unchanged (within epsilon). Collect new functions and new violations. Collect removed functions (in baseline, not in current). Compute pass/fail. Target >= 90% line coverage.
- [x] 3.4 Add unit tests for `LoadBaseline` (TestSC002_*): valid JSON, invalid JSON, empty input, baseline without GazeCRAP fields (pre-contract-coverage baseline), baseline with extra unknown fields (forward compatibility).
- [x] 3.5 Add unit tests for `Compare` (TestSC003_* through TestSC006_*): CRAP regression, CRAP improvement, within-epsilon unchanged, GazeCRAP regression with baseline coverage, GazeCRAP skip when baseline had no coverage, conflicting signals (CRAP regresses but GazeCRAP improves -- regression wins), new function below threshold, new function above threshold (violation), removed function, mixed results pass/fail, empty baseline, empty current. This list is exhaustive -- all branches in `Compare()` must be exercised.

## 4. Report Output

- [x] 4.1 Add `WriteComparisonJSON(w io.Writer, result *ComparisonResult) error` to a new `internal/crap/compare_report.go`. Assemble merged JSON by combining `Report` scores with `FunctionDelta` data: each score gets inline delta fields (`baseline_crap`, `crap_delta`, `baseline_gaze_crap`, `gaze_crap_delta`, `status`). Include `new_functions`, `removed_functions`, and `comparison` top-level sections. This is a separate output path from `WriteJSON` -- the CLI chooses which writer based on whether a baseline was loaded.
- [x] 4.2 Add `WriteComparisonText(w io.Writer, result *ComparisonResult) error` to `compare_report.go`. Output human-readable comparison: summary line (pass/fail, counts), regressions table, improvements table, new functions list (with violation marker), removed functions list. Follow existing `WriteText` formatting patterns (tabular, 80-column friendly).
- [x] 4.3 Add unit tests for `WriteComparisonJSON` (TestSC007_*): verify JSON structure, verify all status values appear correctly, verify `comparison.passed` field. Add a backward-compatibility regression test: render a normal `crap.Report` through `WriteJSON` and assert zero comparison fields appear (`baseline_crap`, `crap_delta`, `status`, `new_functions`, `removed_functions`, `comparison` must all be absent).
- [x] 4.4 Add unit tests for `WriteComparisonText` (TestSC008_*): verify pass/fail header, verify table formatting, verify empty sections are omitted.

## 5. CLI Integration

- [x] 5.1 Add `--baseline` string flag to the `crap` command in `cmd/gaze/main.go`. Wire baseline detection logic: `--baseline FILE` uses explicit path (error if not found or empty); otherwise check config `baseline.file`, then default `.gaze/baseline.json` (skip silently if not found or empty).
- [x] 5.2 Integrate into `runCrap`: after `crap.Analyze()` produces a `*Report`, if baseline is loaded, call `Compare()` and write comparison output via `WriteComparisonJSON` or `WriteComparisonText` (based on format flag). When no baseline is loaded, use existing `WriteJSON`/`WriteText`. Evaluate baseline comparison before `checkCIThresholds` so comparison output is always visible. Set exit code 1 if `ComparisonResult.Passed` is false OR if `checkCIThresholds` fails (both gates independent).
- [x] 5.3 Load baseline config from `.gaze.yaml` in `runCrap` (reuse existing `loadConfig` pattern). Pass `Epsilon` and `NewFunctionThreshold` from config to `CompareOptions`. Validate config values before use.

## 6. Integration Testing

Integration tests in this section read local JSON fixtures and run pure functions. They do NOT spawn subprocesses and complete in milliseconds, so they do NOT use `testing.Short()` guards.

- [x] 6.1 Add integration test (TestSC003_RegressionDetection): use two separate static fixture directories (baseline fixture and modified fixture) under `internal/crap/testdata/`. Run `gaze crap --format=json` on baseline fixture, save output, then run on modified fixture with `--baseline`, verify regression is detected and exit code is 1.
- [x] 6.2 Add integration test (TestSC001_NoBaselineLoaded): verify that when no baseline is loaded, WriteJSON produces zero comparison fields.
- [x] 6.3 Add integration test (TestSC001_ExplicitBaselineNotFound): verify `--baseline /nonexistent` produces an error.
- [x] 6.4 Add integration test (TestSC006_ComparisonPasses): verify that when baseline exists and no regressions are present, `gaze crap` exits with code 0 and includes comparison summary in output.
- [x] 6.5 Add sample baseline JSON fixture under `internal/crap/testdata/` for unit tests.

## 7. Documentation

- [x] 7.1 Update `README.md` with baseline comparison section: how to create a baseline (including `mkdir -p .gaze`), how auto-detection works, how to override/opt-out, recommendation to commit `.gaze/baseline.json` to version control, example CI workflow.
- [x] 7.2 Update `docs/reference/cli/crap.md` with: `--baseline` flag in the Flags table, updated configuration interaction (now reads `baseline` section from `.gaze.yaml`), exit code semantics for regression detection, example output with comparison sections.
- [x] 7.3 Update `AGENTS.md` with baseline-comparison in the Recent Changes section and any new patterns introduced.
- [x] 7.4 Create GitHub issue in `unbound-force/website` repository to track documentation updates for the baseline comparison feature (new CLI flags, configuration section, CI integration workflow). **NOTE**: To be created at PR submission time with: `gh issue create --repo unbound-force/website --title "docs: baseline comparison feature" --body "New --baseline flag on gaze crap, .gaze.yaml baseline config section, CI integration workflow examples. Affects: CLI reference, configuration reference, getting started guide."`

## 8. Constitution Alignment Verification

- [x] 8.1 Verify all four principles: Accuracy (deterministic comparison, configurable epsilon, exhaustive regression tests), Minimal Assumptions (zero setup beyond baseline file, convention over configuration), Actionable Output (per-function deltas with statuses, JSON + text, cross-run comparability fulfilled), Testability (`Compare` is pure function, `LoadBaseline` accepts `io.Reader`, >= 90% coverage target, `testing.Short()` guards on integration tests).
<!-- spec-review: passed -->
<!-- code-review: passed -->
