## Why

`gaze report` is the only external-analyzer-capable command that ignores `--language` alone. `runCrap`, `runQuality`, and `runDocscan` all dispatch to the external analyzer path when `--analyzer` **or** `--language` is set; `runReport`'s extracted helpers still gate on `--analyzer` only (issue #282 — the LOW-severity deferral recorded in PR #281's known issues and in `openspec/changes/quality-classification-counts/`).

Today `gaze report --language python ./src` either fails or lies:

- **Outside a Go module** the run dies with a module-root error (`finding module root` / `no go.mod found`) — the same failure class issues #250/#257 fixed for `--analyzer` — because `resolveReportModuleDir` only takes its working-directory branch when `--analyzer` is set.
- **Inside a Go module** the flag is silently inert: the Go-native pipeline runs against Go source, producing the wrong analysis for a non-Go project with no diagnostic. The project constitution (II. Minimal Assumptions) forbids exactly this: unavoidable assumptions "MUST be explicit in documentation and enforced at analysis entry points — never silently ignored".

The reference docs already promise the fixed behavior — `docs/reference/cli/report.md`: "With `--analyzer` (or `--language` alone, when discovery resolves an analyzer), the report is built from the external analyzer's JSON-RPC data" — and the `/gaze` agent's layered detection (quality-classification-counts design D4) supplies only `--language` for heuristic-detected projects that rely on tier-3 PATH-convention discovery (`gaze-analyzer-<language>`).

Closes #282.

## What Changes

- Widen the two external-analyzer gates in `runReport`'s extracted helpers (`cmd/gaze/main.go`) from `p.analyzerFlag != ""` to `p.analyzerFlag != "" || p.languageFlag != ""` — the identical condition `runCrap`/`runQuality`/`runDocscan` already use:
  - `resolveReportModuleDir` — `--language`-only runs use the working directory instead of requiring a Go module root.
  - `applyReportAnalyzeFunc` — `--language`-only runs build the external-analyzer `AnalyzeFunc`.
- Add a regression test proving `--language`-only `gaze report` reaches external discovery (language-specific error, never a module-root error), mirroring `TestCrapWithLanguageOnly_TriggersExternalPath` / `TestQualityWithLanguageOnly_TriggersExternalPath`.
- Verify the reference docs match the shipped behavior, and file the Website Documentation Gate issue for the user-facing behavior change.

Explicitly **not** changed (verified during planning):

- The plumbing below the gates: `applyReportAnalyzeFunc` already threads `p.languageFlag` into `buildExternalReportAnalyzeFunc` → `initExternalSession` → `adapter.Discover`, and `initExternalSession` already returns the language-specific error `no analyzer found for language %q` (both landed with PR #281). The fix is a gate-condition widening, not new wiring.
- `internal/aireport` step functions. Issue #282 speculates that `runCRAPStep`/`runQualityStep`/`runDocscanStep` "would need to honor the language-only session", but the external path replaces the entire standard step sequence via the `opts.AnalyzeFunc` override (`buildExternalReportAnalyzeFunc` → `runExternalReport`), so those steps never execute on the external path. No `internal/aireport` changes are needed.

## Capabilities

### New Capabilities

- `report-language-dispatch`: `gaze report` dispatches to the external analyzer pipeline when `--language` is set without `--analyzer` — module-dir resolution, three-tier discovery, language-specific failure diagnostics, and Go-native default all matching `gaze crap`, `gaze quality`, and `gaze docscan`.

### Modified Capabilities

- `gaze report` command: `--language` alone now selects the external analyzer path (previously ignored).

### Removed Capabilities

- None.

## Impact

### Files Modified

- `cmd/gaze/main.go` — two gate conditions (`resolveReportModuleDir`, `applyReportAnalyzeFunc`) plus their doc comments
- `cmd/gaze/external_analyzer_test.go` — `TestReportWithLanguageOnly_TriggersExternalPath` regression test
- `docs/reference/cli/report.md` / `README.md` — verify wording matches behavior; adjust only where inaccurate
- `openspec/changes/report-language-dispatch/` — this change

### Behavior Changes

- `gaze report --language <lang>` outside a Go module: previously failed with a module-root error; now dispatches externally, or fails with the actionable `no analyzer found for language "<lang>"` when discovery finds nothing.
- `gaze report --language <lang>` inside a Go module: previously ran Go-native analysis silently; now dispatches externally.
- `gaze report` with neither flag: byte-identical Go-native path (the widened condition is still false).
- No protocol, schema, flag-surface, or `internal/aireport` changes.

## Constitution Alignment

Assessed against the Gaze project constitution (`.specify/memory/constitution.md` v1.3.0).

### I. Accuracy

**Assessment**: PASS

A `--language`-only `gaze report` inside a Go module today reports Go-native analysis for a non-Go project — inaccurate output with no warning. After the change the report is built from the analyzer that can actually observe the project, and the dispatch contract is pinned by an automated regression test (accuracy claims backed by regression tests).

### II. Minimal Assumptions

**Assessment**: PASS

`--language` is an explicit user signal at the analysis entry point; the constitution requires such signals to be "enforced at analysis entry points — never silently ignored". The change enforces it. No new flags, config keys, annotations, or source restructuring are introduced, and the no-flag Go-native default is unchanged.

### III. Actionable Output

**Assessment**: PASS

The failure mode outside Go modules improves from the misleading `finding module root` to `no analyzer found for language "<lang>"`, which names the concrete remediation (provide an analyzer for that language). Successful runs produce the correct external report in every existing output format, and metrics stay comparable across runs because the external pipeline is the one `--analyzer` already uses.

### IV. Testability

**Assessment**: PASS

The change is two gate-condition widenings verified by a hermetic regression test (`t.Chdir(t.TempDir())` plus an undiscoverable language sentinel) following the established `TestXxxWithLanguageOnly_TriggersExternalPath` pattern — no external services, no subprocess analyzer required. The existing report test suite (which injects `analyzeFunc` without language flags) pins the unchanged Go-native path.
<!-- scaffolded by uf vdev -->
