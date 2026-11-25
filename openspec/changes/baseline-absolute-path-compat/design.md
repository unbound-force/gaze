## Context

The baseline comparison feature (`openspec/changes/baseline-comparison/`) matches functions between a saved baseline and the current report using a literal `file:function` composite key (`scoreKey` in `internal/crap/compare.go:55-57`, design D2 of that change). The path lifecycle:

1. Complexity providers emit absolute file paths; `computeScores` uses them for coverage lookups.
2. `Analyze` Step 5b (`internal/crap/analyze.go:154-161`, introduced by commit `6fbaafb` "fix: relativize Score.File paths in JSON output") rewrites every `Score.File` with `filepath.Rel(moduleDir, scores[i].File)` before the report is returned.
3. Baselines are saved by redirecting `gaze crap --format=json` output, so current-version baselines carry module-relative paths — but baselines saved before `6fbaafb` (v1.9.x and earlier, per issue #330) carry absolute paths.

`LoadBaseline` (`compare.go:34-50`) deserializes the historical representation verbatim (lenient JSON, per D5/R6 of the original design), so an absolute-path baseline is keyed as `/<abs>/cmd/x.go:Func` while the current report keys the same function as `cmd/x.go:Func`. Nothing matches. The CLI flow is `runCrap` → `resolveBaselineAndCompare` (`cmd/gaze/main.go:1035-1045`) → `resolveBaselinePath` (D4 order: `--baseline` flag → `baseline.file` config → default `.gaze/baseline.json`) → `loadAndCompare` (`main.go:924-946`) → `openAndLoadBaseline` → `crap.LoadBaseline` → `crap.Compare`. Load errors are fatal when the baseline was explicit and silently skipped (`nil, nil`) when auto-detected — the silent skip exists for the shell-redirect race on empty files. Gates run in `evaluateCrapGates` (`main.go:1080-1098`, D7 ordering: baseline gate before threshold gate).

Field report (#330, complytime/complytime-providers): 282 `new_functions`, 191 `removed_functions`, 0 `unchanged`, 1 false `new_violation` (a function with CRAP 72 whose body is byte-identical to the merge base — above the default new-function threshold of 30), exit 1. Because zero functions match, `regressions: 0` carries no information: the gate fails falsely and cannot detect real regressions at the same time.

## Goals / Non-Goals

### Goals

- Deterministic reconciliation of historical absolute-path baselines to the module-relative representation before `file:function` matching
- An explicit migration mechanism for relocated checkouts: `--baseline-root <dir>`
- A fail-loud, actionable incompatibility diagnostic when baseline paths can be reconciled by neither the current module directory nor `--baseline-root`
- Zero behavior change for module-relative baselines (the normal case): no reconciliation, no notices, byte-identical output
- Preservation of all score semantics: CRAP/GazeCRAP values, epsilon, thresholds, status classification, pass/fail logic, exit codes, and the comparison JSON schema
- The five regression cases mandated by epic #337 exit criteria, each as a named test
- No basename or suffix-only matching, ever (epic #337 Phase 1 rule)

### Non-Goals

- Rewriting baseline files on disk — reconciliation happens in memory at comparison time; a write/migrate mode would need backup, atomicity, and idempotency semantics disproportionate to a one-command regeneration
- Symmetric normalization of current-report paths — Step 5b already normalizes the current side; a current-side absolute path only arises when package patterns point outside the module, a pre-existing and distinct condition
- Cross-OS baselines — a baseline generated on Windows (`C:\...` paths) is not absolute per `filepath.IsAbs` on a Unix host, is left untouched, and will not match; deterministic translation would require separator and drive-letter heuristics, exactly the class of conflation-prone matching the epic forbids. The zero-match warning (D6) surfaces this, and regeneration is the remedy
- New JSON output fields — reconciliation statistics and diagnostics go to stderr only, keeping the machine-readable contract stable
- `gaze report` baseline integration — still scoped out per the original design; `internal/aireport` contains no baseline code (verified)
- Function-rename matching — a rename remains a removed+new pair by design (original D2)

## Decisions

### D1: Reconciliation is a deterministic, no-I/O, in-place normalization of a freshly loaded value; `LoadBaseline` and `Compare` are untouched

`LoadBaseline(r io.Reader)` keeps its pure-deserialization contract (lenient parse, D5/R6 of the original change — its existing tests and the forward/backward compatibility guarantee stay valid). `Compare` stays a pure function over two `Report` values, unaware of path history.

A new file `internal/crap/baseline_compat.go` adds, schematically:

```go
// ReconcileBaselinePaths rewrites absolute baseline Score.File paths to the
// module-relative form used by current reports.
func ReconcileBaselinePaths(baseline *Report, moduleDir, baselineRoot string) (ReconciliationStats, error)

// BaselinePathError reports baseline entries whose absolute paths could not
// be reconciled; carries the count and example paths for the CLI diagnostic.
type BaselinePathError struct { ... }
```

The function performs deterministic, no-I/O, in-place normalization of the freshly loaded `baseline` (safe: the `Report` was unmarshaled for this run and has no other holder) and returns either a stats struct (`Normalized int`) or a typed error. The CLI layer composes the user-facing message (it alone knows the baseline file path). Rationale: separation of parse / normalize / compare concerns; each independently testable with synthetic `Report` values (Constitution IV: Testability; convention pack AP-001/AP-002/AP-007 — business logic under `internal/`, CLI delegates).

### D2: The reconciliation algorithm is deterministic full-prefix relativization

For each baseline score entry, in order:

1. Empty or non-absolute `File` → untouched (already module-relative).
2. `filepath.Rel(moduleDir, File)` succeeds and the result does not escape `moduleDir` → rewrite. The escape test rejects `rel == ".."` and `strings.HasPrefix(rel, ".." + separator)` — required because `filepath.Rel("/a/b", "/a/bc/x.go")` "succeeds" with `../bc/x.go`.
3. Otherwise, when `baselineRoot != ""`: `filepath.Rel(baselineRoot, File)` succeeds and does not escape `baselineRoot` → rewrite.
4. Otherwise: the entry is unreconcilable; collect it.

If any entry is unreconcilable, return `BaselinePathError` carrying the count and example paths. Using the identical `filepath.Rel`-against-moduleDir semantics as Step 5b guarantees agreement with the current report by construction: both sides pass through the same relativization. Same-basename files in different directories remain distinct because the entire path prefix must match the root — no basename or suffix comparison exists anywhere in the path (epic rule). Deterministic root order (moduleDir first, then `baselineRoot`) is specified for reproducibility; in practice an absolute path cannot lie under two different absolute roots.

Rejected alternative: reconcile only when a comparison yields zero matches (detect-then-retry). Rejected as a heuristic trigger — a legitimately all-new codebase would misfire it, and a partially-matching (hand-edited) baseline would skip normalization entirely. Always-on normalization is correct by construction and O(n) string operations.

### D3: Incompatibility is a hard error in both explicit and auto-detected modes

`loadAndCompare` returns the `BaselinePathError` (wrapped with the baseline file path and remediation text); `resolveBaselineAndCompare` propagates it; `runCrap` returns it at the existing `if baselineErr != nil` site (`main.go:664-666`), producing a non-zero exit before any comparison output is written.

This is a deliberate deviation from the silent-skip rule for auto-detected baselines. The silent skip exists for unreadable/unparseable files (the shell-redirect race, non-adoption tolerance). A parsed-but-incompatible baseline is neither: the user has adopted baselines and the file is valid. Silently skipping it would convert today's false failure into a silent false pass of a governance gate — strictly worse (Constitution III: Observable Quality; project principle Accuracy: false signals erode trust). The diagnostic must name the baseline file, the unreconcilable count, at least one example path, and both remediations:

```
baseline /path/to/baseline.json: 12 entries use absolute paths outside the
current module (e.g. /home/alice/gaze/internal/crap/analyze.go).
Regenerate the baseline with the current gaze version:
  gaze crap --format=json --coverprofile=coverage.out ./... > /path/to/baseline.json
or re-run with --baseline-root /home/alice/gaze (the module root the
baseline was generated under).
```

Parse/IO errors keep their existing handling unchanged (fatal when explicit, silently skipped when auto-detected) — the spec pins this boundary so the redirect-race behavior is preserved.

### D4: `--baseline-root` is a CLI-only migration affordance

`newCrapCmd` registers `--baseline-root` (string, default `""`), threaded `runCrapParams.baselineRoot` → `resolveBaselineAndCompare` → `loadAndCompare` → `ReconcileBaselinePaths`. The flag is inert when no baseline is loaded or the baseline has no absolute paths. There is deliberately no `.gaze.yaml` key: the root describes a historical machine state (where an old checkout lived), and a config knob would outlive its purpose and silently mis-normalize future baselines. Flag-only also satisfies the issue's "supported, explicit compatibility/migration mechanism" requirement — the user states the historical root explicitly, and a wrong root fails visibly (D6 warning plus normal gate behavior) rather than guessing.

Migration guide (to be surfaced in docs):

| Situation | Action |
|---|---|
| Same checkout path as when the baseline was generated | Nothing — auto-reconciliation handles it |
| Relocated checkout (different machine or directory) | `gaze crap --baseline-root <old-module-root> ./...` once, then regenerate |
| Historical root unknown or cross-OS baseline | Regenerate: `gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json` |

### D5: Semantics preservation — only `File` is rewritten, and only in memory

Reconciliation modifies exactly one field of baseline score entries: `File`. CRAP/GazeCRAP values, epsilon, thresholds, `classifyDelta`, `isNewFunctionViolation`, `buildComparisonSummary`, and `ComparisonResult.Passed` are untouched. The comparison output (deltas, removed-function lists) therefore displays normalized baseline paths — deliberate, since rendering `/home/alice/gaze/internal/crap/analyze.go` beside the current `internal/crap/analyze.go` for the same function would be confusing. Rejected alternative: match via a side key map while displaying original paths — rejected for inconsistent representation and extra state. The JSON schema gains no fields; stats and diagnostics are stderr-only. Rejected alternative: a `reconciliation` section in comparison JSON — rejected as YAGNI with schema and consumer impact for migration-time-only information.

### D6: Observability — normalization notice and generalized zero-match warning

- When `Normalized > 0`, one stderr line reports the count and recommends regenerating the baseline (e.g. `baseline: normalized 191 absolute paths to module-relative (historical baseline format; consider regenerating)`). It fires in both explicit and auto-detected modes; stderr never pollutes `--format=json` stdout.
- When a comparison matches zero functions — baseline and current both non-empty, no baseline entry matching any current entry — a stderr warning names the likely causes (incorrect `--baseline-root`, baseline generated on another OS, entirely rewritten codebase). This generalizes the issue's core observation ("zero matched functions means `regressions: 0` cannot be trusted") to every way it can arise, including conditions this change deliberately does not auto-fix. The warning never changes the exit code: it is observability, not a gate (gate values are protected; adding a new exit condition is unnecessary because the underlying comparison already fails loudly when it should). This warning is permanently `stderr`-observability — it `MUST NOT` be promoted to a gate or exit-code change without a new spec; a legitimately all-functions-removed codebase producing zero matches is accepted noise.

### D7: Testing strategy — the five epic-mandated cases plus boundary matrix

Pure-function table tests for `ReconcileBaselinePaths` (synthetic `Report` values, `t.TempDir()` roots, TC-006): relative untouched; absolute under moduleDir; absolute sibling of moduleDir (`/mod/../other`, must be rejected via the `..` test); absolute under `baselineRoot`; absolute under neither → `BaselinePathError` with correct count and examples; empty `File`; empty baseline (no-op, no error); mixed relative+absolute; stats counting. Comparison-level regression tests run `Compare` over reconciled baselines for the five epic cases (spec Coverage Strategy): same function absolute-vs-relative matched `unchanged` with CRAP above threshold and not `new_violation`; same basename/function in two directories matched distinctly; relocated checkout with `--baseline-root` matched and gate passing, without it producing the typed error; genuine new (`new_violation`) and removed functions after reconciliation; unchanged-score-above-threshold gate passing. CLI-level tests via the `runCrap` params pattern (AP-003): flag threading, auto-detected incompatible baseline erroring with remediation substrings, notice and warning on stderr, and the pre-existing unparseable-auto-detected silent-skip test continuing to pass. All tests are in-process (no subprocesses), so no `testing.Short()` guards; the full suite runs under `-race -count=1`.

### D8: Scope of code changes

- `internal/crap/baseline_compat.go` (new): `ReconcileBaselinePaths`, `ReconciliationStats`, `BaselinePathError`
- `internal/crap/baseline_compat_test.go` (new): D7 tests
- `cmd/gaze/main.go`: `--baseline-root` flag registration, `runCrapParams.baselineRoot`, threading through `resolveBaselineAndCompare`/`loadAndCompare`, notice/warning/diagnostic rendering (the zero-match warning hooks where the `ComparisonResult` is available, after `crap.Compare`)
- `internal/crap/compare.go`, `compare_report.go`, `crap.go`: unchanged
- `internal/config`: unchanged (no new config)
- Docs: `docs/reference/cli/crap.md` (flag row, reconciliation section), `docs/guides/ci-integration.md` (migration guidance), `README.md` (baseline section note); website issue in `unbound-force/website` (new user-facing flag; precedent #150)

## Risks / Trade-offs

### R1: Auto-reconciliation can mask a baseline from a different module at the same path

A baseline generated by another repository checked out at the same directory would relativize cleanly and then match few or no current functions. Residual risk accepted: matching remains full `file:function`, so wrong-module baselines produce near-zero matches, which the D6 zero-match warning surfaces. This is no worse than the status quo for module-relative baselines.

### R2: A wrong `--baseline-root` relativizes to paths that do not exist

The user supplies an incorrect historical root; entries relativize to plausible-but-wrong relative paths and appear as removed+new, failing the gate with violations. Mitigation: the D6 zero-match warning (the common outcome) and the fact that the prefix strip is deterministic — a root that produces matching paths is, for those entries, effectively correct. No false-pass path exists that silent matching would not also allow. Accepted.

### R3: Auto-detected incompatible baselines change failure text, not exit direction

Today such baselines exit 1 with false violations; after this change they exit 1 with the truthful diagnostic. No CI workflow that was passing can start failing (the configuration was already failing), and no failing workflow keeps its old misleading message. Documented as intended behavior change (spec: Incompatible Baseline Diagnostic).

### R4: Cross-OS baselines remain unsupported

Deterministic translation needs separator and drive-letter heuristics — the conflation-prone class the epic explicitly forbids. The zero-match warning surfaces the condition; regeneration is the documented remedy. Non-goal, recorded here so reviewers do not re-litigate it as an oversight.

### R5: In-place mutation of the loaded baseline

`ReconcileBaselinePaths` mutates its argument. Safe because the `Report` is freshly unmarshaled per run with no concurrent holder; the contract is documented in GoDoc. A copy-returning variant was rejected as unnecessary allocation and a wider API.
