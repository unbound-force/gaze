## Why

Gaze's Constitution (Principle III: Actionable Output) mandates that "Metrics MUST be comparable across runs so users can measure progress over time." Today, gaze produces rich per-function CRAP and GazeCRAP scores but has no built-in mechanism to compare results across runs or detect regressions.

The complytime/org-infra repository built this capability externally: a 315-line bash comparison script (`compare-crapload.sh`), a 63-line package resolver (`resolve-go-packages.sh`), and a 475-line Python test suite -- all operating on gaze's JSON output via `jq` and `sed`. This works, but every organization adopting gaze must either use the org-infra reusable workflow or rebuild the comparison logic. The comparison logic belongs in gaze itself.

Moving baseline comparison into gaze:
- Fulfills the constitutional mandate for cross-run comparability
- Eliminates ~850 lines of external bash/Python in org-infra
- Works identically in CI and locally (no workflow dependency)
- Requires no AI, no secrets, no network calls beyond `go test`
- Simplifies adoption: `gaze crap` auto-detects a baseline and compares

## What Changes

### CLI Behavior

`gaze crap` gains automatic baseline comparison. When `.gaze/baseline.json` exists, `gaze crap` loads it and produces comparison output (per-function deltas, regressions, improvements, new functions, removed functions). When the file does not exist, behavior is unchanged.

A `--baseline FILE` flag overrides the default path. Auto-detected baselines that are empty (0 bytes) are silently skipped, which handles the shell redirect race condition when creating a new baseline.

Exit code 1 when any regression or new-function violation is detected (CI gate behavior).

### Configuration

New `baseline` section in `.gaze.yaml`:

```yaml
baseline:
  file: .gaze/baseline.json      # default baseline path
  epsilon: 0.5                   # score change tolerance
  new_function_threshold: 30     # max CRAP for new functions
```

All fields have sensible defaults and are optional.

### Output Formats

Both `--format=text` and `--format=json` include comparison data when a baseline is loaded. JSON output adds `baseline_crap`, `baseline_gaze_crap`, `crap_delta`, `gaze_crap_delta`, and `status` fields to each score, plus `new_functions`, `removed_functions`, and `comparison` summary sections.

### Baseline Lifecycle

Saving: `gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json` (empty auto-detected baselines are silently skipped, so the shell redirect race is handled automatically).

Comparing: `gaze crap --coverprofile=coverage.out ./...` (automatic when baseline exists).

## Capabilities

### New Capabilities
- `baseline-comparison`: Per-function CRAP and GazeCRAP regression detection against a saved baseline, with configurable epsilon tolerance and new-function threshold
- `baseline-auto-detect`: Automatic baseline loading from `.gaze/baseline.json` when the file exists and is non-empty, with `--baseline` override
- `baseline-config`: `.gaze.yaml` `baseline` section for epsilon, new-function threshold, and baseline file path configuration

### Modified Capabilities
- `gaze crap`: Extended output with comparison data when baseline is available; exit code 1 on regression detection

### Removed Capabilities
- None

## Impact

### Files Modified
- `cmd/gaze/main.go` -- `--baseline` flag, baseline loading and comparison invocation in `runCrap`
- `internal/crap/crap.go` -- New comparison types (`ComparisonResult`, `FunctionDelta`, `ComparisonSummary`)
- `internal/crap/compare.go` -- New file: comparison logic (load baseline, match functions, compute deltas, determine pass/fail)
- `internal/crap/compare_report.go` -- New file: comparison output formatters (`WriteComparisonJSON`, `WriteComparisonText`)
- `internal/config/config.go` -- New `BaselineConfig` struct, added to `GazeConfig`
- `docs/reference/configuration.md` -- Document `baseline` config section

### External Impact
- complytime/org-infra can delete `compare-crapload.sh`, `resolve-go-packages.sh`, and `test_crapload_package_resolution.py`, replacing them with a direct `gaze crap` invocation
- Reusable workflow (`reusable_crapload_analysis.yml`) shrinks from ~250 lines to ~80 lines
- Consumer workflow (`ci_crapload.yml`) unchanged (still calls the reusable workflow)

### Backward Compatibility
- All new JSON fields use `omitempty` -- no output changes when no baseline exists
- Exit code behavior unchanged when no baseline exists
- No existing flags or config fields modified

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: PASS

Comparison logic uses deterministic numeric comparison with configurable epsilon tolerance. No heuristics, no AI, no probabilistic matching. Function matching uses `file:function` keys from gaze's own output, which already normalizes paths to module-relative form (`internal/crap/analyze.go:137-144`). The epsilon default (0.5) absorbs platform/toolchain noise without masking real regressions -- matching the value validated in production by org-infra.

### II. Minimal Assumptions

**Assessment**: PASS

The feature requires zero user setup beyond creating the initial baseline file. Auto-detection of `.gaze/baseline.json` follows Convention Over Configuration. No annotations, no project restructuring, no mandatory config file. The baseline format is gaze's own `--format=json` output -- no separate schema to learn. Users who never create a baseline see zero behavior change.

### III. Actionable Output

**Assessment**: PASS

This change directly fulfills the constitutional mandate that "Metrics MUST be comparable across runs so users can measure progress over time." Each function in the comparison output carries its baseline score, current score, delta, and status (regression/improvement/new/unchanged/removed). The JSON output is structured for machine consumption (CI workflows, PR comment formatters). The text output is human-readable for local development.

### IV. Testability

**Assessment**: PASS

The comparison logic is a pure function: given a baseline `Report` and a current `Report`, produce a `ComparisonResult`. No I/O, no global state, no external dependencies. Baseline loading is a separate function accepting an `io.Reader`. Both are independently testable with synthetic inputs. The `runCrap` function already uses a params struct with `io.Writer` for output -- comparison integration follows the same pattern.
