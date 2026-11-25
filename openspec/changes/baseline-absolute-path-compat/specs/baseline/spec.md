## ADDED Requirements

### Requirement: Baseline Path Reconciliation

When a baseline is loaded, `gaze crap` MUST reconcile each baseline `scores[].file` path to the module-relative representation used by current reports before `file:function` key matching is performed. A baseline path that is already relative MUST be left unchanged. Reconciliation MUST NOT alter which functions are genuinely new or removed.

#### Scenario: Module-relative baseline behavior is unchanged

- **GIVEN** a baseline whose entries all carry module-relative paths
- **WHEN** the comparison is performed
- **THEN** no reconciliation notice MUST be printed and the output MUST be identical to the behavior before this change

#### Scenario: Real new and removed functions remain detectable after reconciliation

- **GIVEN** a reconciled baseline, a current report that contains a function absent from the baseline with CRAP 42 (above the new-function threshold), and a baseline function absent from the current report
- **WHEN** the comparison is performed
- **THEN** the genuinely new function MUST be classified `new_violation`
- **AND** the genuinely absent function MUST appear in `removed_functions`
- **AND** the comparison MUST fail (exit non-zero) due to the new violation, exactly as it would with a module-relative baseline

### Requirement: Absolute Path Rewrite Rules

An absolute baseline path MUST be rewritten to module-relative form using the same `filepath.Rel` semantics the report pipeline applies when emitting reports, evaluated against the current module directory first and then, when set, the `--baseline-root` directory. A rewrite MUST only be applied when the resulting relative path stays within its base directory; a result that escapes the base (begins with `..`) MUST be treated as not lying under that base.

#### Scenario: Absolute baseline path matches module-relative current path

- **GIVEN** a baseline whose entry for `RuleResultMessage` has file `/<module-dir>/cmd/openscap-provider/xccdf/arf.go` and CRAP 72, and the current run reports the same function at `cmd/openscap-provider/xccdf/arf.go` with CRAP 72
- **WHEN** the comparison is performed with the baseline active
- **THEN** the function MUST be matched and classified `unchanged`
- **AND** it MUST NOT appear in `new_functions` or `removed_functions`
- **AND** it MUST NOT be classified `new_violation` even though its unchanged CRAP score exceeds the new-function threshold (default 30)

#### Scenario: Path escaping the module directory is not rewritten

- **GIVEN** a baseline entry with file `/<parent>/other/x.go` while the module directory is `/<parent>/mod` (a sibling directory, so relativizing yields `../other/x.go`), and `--baseline-root` is not set
- **WHEN** reconciliation is performed
- **THEN** the entry MUST NOT be rewritten and MUST count as unreconcilable

### Requirement: No Basename or Suffix Matching

Function matching between baseline and current results MUST NOT fall back to basename-only or suffix-only path comparison. Baseline entries with the same basename and function name in different directories MUST remain distinct entries that match only current entries with the identical relative path.

#### Scenario: Same basename and function in several files stays distinct

- **GIVEN** a baseline with absolute-path entries for `util.go:Helper` under `/<module-dir>/pkg/a/util.go` and `/<module-dir>/pkg/b/util.go`, and a current report containing both `pkg/a/util.go:Helper` and `pkg/b/util.go:Helper`
- **WHEN** the comparison is performed
- **THEN** each baseline entry MUST match the current entry with the identical relative path
- **AND** no cross-directory conflation (a baseline entry matching the wrong file's function) MUST occur

### Requirement: Incompatible Baseline Diagnostic

When one or more baseline entries carry absolute paths that can be reconciled by neither the current module directory nor the `--baseline-root` directory, `gaze crap` MUST fail with a non-zero exit before any comparison output is written. The error MUST name the baseline file path, state the number of unreconcilable entries, include at least one example offending path, and state both remediations: regenerating the baseline, and re-running with `--baseline-root` set to the historical module root.

#### Scenario: Relocated checkout without `--baseline-root`

- **GIVEN** a baseline whose entries carry absolute paths rooted at `/home/alice/gaze` while the current module directory is `/Users/jay/gaze`, and `--baseline-root` is not set
- **WHEN** the user runs `gaze crap --baseline .gaze/baseline.json ./...`
- **THEN** `gaze crap` MUST exit non-zero
- **AND** the error MUST identify the baseline file, the count of unreconcilable entries, an example absolute path, and both remediations
- **AND** no comparison result claiming functions as new or removed MUST be produced

#### Scenario: Auto-detected incompatible baseline is a hard error

- **GIVEN** a file at `.gaze/baseline.json` whose entries carry unreconcilable absolute paths
- **WHEN** the user runs `gaze crap ./...` without `--baseline`
- **THEN** `gaze crap` MUST exit non-zero with the incompatibility diagnostic rather than silently skipping the comparison or producing a misclassified comparison

### Requirement: Incompatibility Error Scope

The incompatibility failure MUST occur regardless of whether the baseline was loaded via the `--baseline` flag or auto-detected. Baselines that cannot be read or parsed MUST retain their existing handling: fatal when explicitly requested, silently skipped when auto-detected.

#### Scenario: Unparseable auto-detected baseline remains silently skipped

- **GIVEN** a file at `.gaze/baseline.json` containing invalid JSON
- **WHEN** the user runs `gaze crap ./...` without `--baseline`
- **THEN** the file MUST be silently skipped with no comparison and no error, matching the behavior before this change

### Requirement: `--baseline-root` Flag

The `gaze crap` command MUST accept a `--baseline-root <dir>` flag supplying the historical module root under which an absolute-path baseline was generated. When set, absolute baseline paths lying under that root MUST be rewritten relative to it before matching. The flag MUST be inert when no baseline is loaded or when the baseline contains no absolute paths. The historical root MUST be configurable only via this CLI flag; no `.gaze.yaml` key MUST be introduced for it.

#### Scenario: Relocated checkout with `--baseline-root`

- **GIVEN** a baseline whose entries carry absolute paths rooted at `/home/alice/gaze`, a current module directory at `/Users/jay/gaze`, and current report paths matching the baseline's suffix structure (e.g., `internal/crap/analyze.go:Analyze` on both sides)
- **WHEN** the user runs `gaze crap --baseline .gaze/baseline.json --baseline-root /home/alice/gaze ./...`
- **THEN** the absolute entries MUST be rewritten relative to `/home/alice/gaze` and matched against the current report
- **AND** a function with unchanged scores MUST be classified `unchanged` and the comparison MUST pass when no regressions or new violations exist

#### Scenario: Flag inert without absolute-path baseline

- **GIVEN** no baseline file exists, or the active baseline contains only module-relative paths
- **WHEN** the user runs `gaze crap --baseline-root /some/dir ./...`
- **THEN** the flag MUST have no effect on behavior or output

### Requirement: Normalization Notice

When reconciliation rewrites one or more baseline paths, `gaze crap` MUST print a one-line stderr notice stating the normalized count and recommending that the baseline be regenerated. The notice MUST NOT change the exit code.

#### Scenario: Normalization notice

- **GIVEN** an active baseline whose absolute paths were rewritten during reconciliation
- **WHEN** the comparison completes
- **THEN** a one-line notice MUST appear on stderr stating the normalized count and recommending baseline regeneration

### Requirement: Zero-Match Warning

When a comparison matches zero functions — the baseline and the current report are both non-empty but no baseline entry matches any current entry — `gaze crap` MUST print a stderr warning that the baseline may not correspond to the current module, naming the likely causes. The warning MUST NOT by itself change the exit code. This warning is permanently `stderr`-observability — it `MUST NOT` be promoted to a gate or exit-code change without a new spec; a legitimately all-functions-removed codebase producing zero matches is accepted noise.

#### Scenario: Zero-match warning

- **GIVEN** an active baseline and a current report where no baseline entry matches any current entry (for example, an incorrect `--baseline-root` relativized entries to paths that do not exist in the current report)
- **WHEN** the comparison completes
- **THEN** a warning MUST appear on stderr stating that the baseline may not correspond to the current module and naming the likely causes (an incorrect `--baseline-root`, a baseline generated on another operating system, or an entirely rewritten codebase)
- **AND** the warning MUST NOT by itself change the exit code

### Requirement: Reconciliation Semantics Preservation

Reconciliation MUST modify only the `file` field of baseline score entries. CRAP and GazeCRAP values, epsilon, new-function thresholds, status classification, pass/fail logic, and exit codes MUST be preserved unchanged, and the comparison JSON output MUST contain no new fields.

#### Scenario: Comparison JSON schema unchanged

- **GIVEN** an active reconciled baseline
- **WHEN** the user runs `gaze crap --format=json ./...`
- **THEN** the JSON output MUST use the same schema as comparison output before this change, with no fields added by reconciliation (diagnostics appear on stderr only)

### Coverage Strategy

New reconciliation code MUST achieve >= 90% line coverage via unit tests. The five regression cases mandated by epic #337 (unbound-force/gaze#330 exit criteria) MUST each have a named test:

1. Historical absolute vs. module-relative paths for the same function (matched, `unchanged`, no false `new_violation` above threshold)
2. The same basename/function in several files (no cross-directory conflation)
3. A relocated checkout (matched with `--baseline-root`; actionable incompatibility error without it)
4. Real new/removed functions (detection preserved after reconciliation)
5. An unchanged score above the new-function threshold (matched, not `new_violation`)

Tests use synthetic `crap.Report` values and `t.TempDir()` module directories — no subprocess execution, no `testing.Short()` guards required.
