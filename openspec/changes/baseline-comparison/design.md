## Context

Gaze produces rich per-function CRAP and GazeCRAP scores in JSON format, but has no built-in mechanism to compare results across runs. The complytime/org-infra repository built this externally as a bash script operating on gaze's JSON output. This design brings that comparison logic into gaze as a native Go feature, following the established patterns in the codebase.

Relevant existing patterns:
- `crap.Report` / `crap.Score` / `crap.Summary` types define the data model (`internal/crap/crap.go`)
- `crap.Analyze()` produces a `*Report` with relative file paths (`internal/crap/analyze.go:137-144`)
- `crap.WriteJSON()` / `crap.WriteText()` handle output formatting (`internal/crap/report.go`)
- `config.GazeConfig` uses nested structs with `yaml` tags, defaults in `DefaultConfig()` (`internal/config/config.go`)
- CLI commands use params structs with `io.Writer` for testability (`cmd/gaze/main.go`)
- Optional fields use pointer types with `json:"name,omitempty"` (established in `crap.Score`)

## Goals / Non-Goals

### Goals
- Deterministic per-function CRAP and GazeCRAP regression detection
- Auto-detection of baseline at `.gaze/baseline.json` (convention over configuration)
- CI gate behavior (exit 1 on regression)
- Structured JSON output for downstream consumers (PR comment formatters, dashboards)
- Human-readable text output for local development
- Configurable tolerance (epsilon) and new-function threshold via `.gaze.yaml`
- Works without AI, without secrets, without network calls beyond `go test`

### Non-Goals
- Quality-level comparison (per-test contract coverage deltas, assertion mapping changes) -- CRAP scores transitively reflect quality changes
- Classification-level comparison (per-effect confidence shifts) -- too noisy for CI gates
- Automatic baseline update (users explicitly save baselines via stdout redirection)
- PR comment generation (caller's responsibility; gaze provides structured data)
- AI-enhanced regression narratives (future enhancement, not this change)
- Side-effect-level diffing via stable IDs (infrastructure exists but is a separate feature)
- `gaze report` baseline integration -- baseline comparison is scoped to `gaze crap` in this iteration; `gaze report` runs CRAP analysis internally but does not accept `--baseline`; this is a future enhancement

## Decisions

### D1: Comparison lives in `internal/crap/compare.go`

The comparison logic is a pure function: `Compare(baseline *Report, current *Report, opts CompareOptions) *ComparisonResult`. No I/O, no global state. Baseline loading is a separate function `LoadBaseline(r io.Reader) (*Report, error)` that deserializes a `crap.Report` from JSON.

This keeps comparison testable with synthetic `Report` values and follows the codebase pattern of small, focused functions (per spec 009 decomposition approach).

### D2: Function matching uses `file:function` composite key

Functions are matched between baseline and current by their `file` + `function` fields (e.g., `internal/crap/analyze.go:Analyze`). This is the same key org-infra's bash script uses and is stable across runs for the same codebase state.

Limitations accepted: function renames appear as a "removed" + "new" pair. This is intentional -- a rename that changes the function's identity should be re-evaluated against the new-function threshold. Receivers are not included in the key because `Score.Function` already includes receiver information when present (e.g., `(*OllamaAdapter).Format`).

### D3: Comparison result wraps the existing `Report` with clean type separation

The comparison output is a `ComparisonResult` that wraps the current `*Report` (via embedding or field) and adds comparison-specific data. The core `crap.Score` type is NOT modified -- no comparison fields are added to it. Instead, comparison data lives in `FunctionDelta` structs that pair baseline and current `Score` values with computed deltas and status.

The `WriteComparisonJSON` function assembles the merged JSON output by combining `Report` scores with `FunctionDelta` data, producing enriched score objects with delta fields inline. The `WriteComparisonText` function produces human-readable comparison tables. Both are separate from `WriteJSON`/`WriteText` -- the CLI chooses which writer to call based on whether a baseline was loaded.

This preserves clean type boundaries: `Score` remains a pure analysis result, `FunctionDelta` carries comparison semantics, and `ComparisonResult` is the top-level envelope. The JSON wire format includes delta fields on each score object (for consumer convenience), but the Go types are cleanly separated. Existing consumers of `WriteJSON`/`WriteText` are completely unaffected -- they never see comparison types.

### D4: Auto-detection with explicit overrides

Baseline detection order:
1. `--baseline FILE` flag → use specified path (error if not found or empty)
2. `.gaze.yaml` `baseline.file` → use configured path (skip silently if not found or empty)
3. Default `.gaze/baseline.json` → use if exists and non-empty (skip silently otherwise)

The "skip silently" behavior for auto-detected paths is deliberate: users who haven't adopted baselines see zero behavior change. Empty files are silently skipped to handle the shell redirect race condition (the shell truncates the output file before gaze starts, so `.gaze/baseline.json` is 0 bytes during the save command). The "error if not found" behavior for explicit `--baseline` is deliberate: an explicit flag implies intent, and a missing or empty file is a user error.

### D5: GazeCRAP regression only when baseline had coverage

When a function's baseline `gaze_crap` is nil or zero (no contract coverage data existed), any GazeCRAP value in the current run is not treated as a regression. This prevents false regressions when contract coverage is first measured for a function. This matches org-infra's proven behavior.

### D6: Configuration defaults match org-infra

- `epsilon`: 0.5 (absorbs platform/toolchain noise)
- `new_function_threshold`: 30 (CRAP score above which a new function is a violation)
- `baseline.file`: `.gaze/baseline.json`

These defaults are production-validated by org-infra across multiple repositories. Users can override via `.gaze.yaml` without CLI flags for the common case.

### D7: Exit code semantics

- Exit 0: no regressions, no new-function violations (pass)
- Exit 1: at least one regression or new-function violation (fail)
- No baseline loaded: exit 0 (comparison not performed, not a failure)

This maps directly to CI gate expectations. The exit code is determined by `ComparisonResult.Passed` which is computed from `Regressions == 0 && NewViolations == 0`.

Note: `gaze crap` already has threshold-based exit 1 logic via `--max-crapload` and `--max-gaze-crapload` flags (`cmd/gaze/main.go:602-605`, enforced by `checkCIThresholds`). Baseline comparison adds a second independent exit-1 gate. Both gates (baseline comparison and thresholds) are evaluated independently -- exit code is 1 if either gate fails. The comparison output is always written regardless of threshold results, so users see the full picture. In `runCrap`, comparison evaluation should happen before `checkCIThresholds` so that comparison output is visible even when thresholds also fail.

### D8: Text output includes comparison sections

When baseline comparison is active, the text report appends:
- Comparison summary (pass/fail, counts)
- Regressions table (function, baseline CRAP, current CRAP, delta)
- Improvements table (same format)
- New functions list (with violation flag if above threshold)
- Removed functions list

These follow the existing text report patterns in `crap.WriteText()` which already outputs tabular data for worst offenders and quadrant distribution.

## Risks / Trade-offs

### R1: Function renames cause false signals

A renamed function appears as "removed" (old name) + "new" (new name, possibly flagged as violation). This is acceptable because:
- Renames are infrequent
- The "removed" entry is informational (not a regression)
- The "new" entry is only a violation if CRAP > threshold, which is a valid check regardless of rename
- Detecting renames reliably (by matching on CRAP profile or function body similarity) adds complexity disproportionate to the benefit

### R2: Baseline staleness

The baseline is a committed file that drifts from the current main branch state. Functions added on main after the baseline was created will appear as "new" indefinitely until the baseline is updated. This is acceptable because:
- "New" without violation (CRAP below threshold) is informational, not a failure
- Periodic baseline refresh is a normal workflow (similar to lockfile updates)
- A future enhancement could warn when the baseline is old, but this is not required for the initial feature

### R3: Package path changes between runs

If the module path or directory structure changes, `file:function` keys will not match. This is an edge case (module renames are rare) and the result is correct: all functions appear "new" against the old baseline, triggering a baseline refresh. No special handling needed.

### R4: Performance is unchanged

Comparison is a lightweight in-memory operation (iterate two sorted arrays). The baseline file is small (tens of KB for typical projects). The dominant cost remains `go test -coverprofile` and the CRAP analysis pipeline. No measurable performance impact.

### R5: Configuration drift between baseline and current runs

If the baseline was created with different analysis settings (e.g., different `--include-unexported` flag, different `.gaze.yaml` classification thresholds), scores may shift in ways that appear as regressions but are actually configuration-induced. This is an accepted limitation -- users should re-create baselines after changing analysis configuration. The baseline file does not record which settings were used; adding a metadata section is a future enhancement.

### R6: Baseline schema evolution

The baseline format is gaze's own JSON output, which evolves as new fields are added (e.g., `fix_strategy`, `contract_coverage_reason` were recent additions). `LoadBaseline` uses `json.Unmarshal` which is lenient by default in Go: unknown fields are ignored, missing optional fields get zero values. This provides forward and backward compatibility without explicit versioning. A baseline created with an older gaze version (missing `gaze_crap` fields) works correctly because D5 already handles nil `gaze_crap`. A baseline created with a newer version (extra fields) is silently accepted.

### R7: Baseline file lifecycle

The `.gaze/baseline.json` file should be committed to version control for CI gate usage (the whole point is comparing PRs against a known-good state). Users performing local-only comparisons may choose to gitignore it, but this is not the recommended workflow. Documentation (README, CLI reference) should include `mkdir -p .gaze` as part of the baseline creation instructions, since the directory does not exist by default.
