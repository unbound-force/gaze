## Why

`gaze analyze ./...` silently analyzes only the first matched package and discards the rest (GitHub issue #107). A user running `analyze ./...` for review or CI gets a silently incomplete report — exit 0, no warning, output looks complete. This is a Principle I (Accuracy) and Principle III (Actionable Output) violation: the tool produces partial results without any signal that they are partial.

The root cause is `loader.Load()` at `internal/loader/loader.go:51`, which takes `pkgs[0]` and discards remaining packages. The `analyze` command uses `cobra.ExactArgs(1)`, so it accepts `./...` as a single string but the loader only processes the first resolved package.

The same limitation exists in `gaze quality` (`cobra.ExactArgs(1)`, single `pkgPath string`).

Meanwhile, `gaze crap` and `gaze report` already handle multi-package patterns correctly using `resolvePackagePaths` to expand patterns, then iterating per package. The fix pattern is well-established in the codebase.

## What Changes

### New Capabilities
- `gaze analyze [packages...]`: Accepts one or more package patterns (including `./...`), resolves them to individual packages, analyzes each, and merges results into a single output.
- `gaze quality [packages...]`: Same multi-package expansion for the quality command.

### Modified Capabilities
- `loader.Load`: Emits a warning log when `len(pkgs) > 1` to prevent silent truncation if any future caller uses it with a wildcard pattern.

### Removed Capabilities
- `gaze analyze` single-package limitation: The README known-limitation entry ("Single package loading") is removed.

## Impact

- **Files modified**: `cmd/gaze/main.go` (analyze and quality command setup, params structs, runner functions), `internal/loader/loader.go` (warning on multi-package), `README.md` (remove known limitation), `docs/reference/cli/analyze.md`, `docs/reference/cli/quality.md`.
- **Backward compatibility**: Fully backward-compatible. Existing single-package invocations (`gaze analyze ./internal/crap`) continue to work identically. The change expands `ExactArgs(1)` to `MinimumNArgs(1)` and changes `pkgPath string` to `patterns []string`.
- **JSON output**: When multiple packages are analyzed, the JSON output is a merged array of all analysis results (same schema, more entries). Text output concatenates per-package sections.
- **CI impact**: No CI threshold changes needed. The `gaze report` pipeline already handles multi-package; this change only affects the `analyze` and `quality` direct CLI commands.

## Constitution Alignment

Assessed against the Gaze project constitution (v1.3.0).

### I. Accuracy

**Assessment**: PASS

This change directly improves accuracy by eliminating silent truncation of analysis results. When a user requests analysis of `./...`, all matched packages will be analyzed instead of only the first. The current behavior silently drops packages, which means the tool misses side effects in dropped packages — a false negative by omission.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions are introduced. The change uses the existing `resolvePackagePaths` pattern already proven in `crap` and `report`. Users do not need to change their invocation patterns — `gaze analyze ./internal/crap` continues to work identically.

### III. Actionable Output

**Assessment**: PASS

This change fixes a Principle III violation. Currently, `gaze analyze ./...` produces output that appears complete but is silently partial. After this change, the output will include all matched packages, and the results will be directly actionable for the full scope the user requested.

### IV. Testability

**Assessment**: PASS

The `runAnalyze` and `runQuality` functions already accept params structs with `io.Writer` for stdout/stderr, making them unit-testable without subprocess execution. The multi-package iteration will follow the same testable pattern used in `crap` and `report`. Coverage strategy: unit tests for the pattern expansion and result merging, plus integration tests verifying multi-package output.
