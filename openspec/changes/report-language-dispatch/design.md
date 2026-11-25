# Design: Report Language-Only External Dispatch

## Context

`runReport` (`cmd/gaze/main.go`) delegates its external-analyzer decision to
two extracted helpers, each of which gates on `--analyzer` alone:

- `resolveReportModuleDir` — returns the working directory when
  `p.analyzerFlag != ""` (the analyzed project may not be a Go module),
  otherwise requires a Go module root via `loader.FindModuleRoot`.
- `applyReportAnalyzeFunc` — builds the external-analyzer `AnalyzeFunc` via
  `buildExternalReportAnalyzeFunc` when `p.analyzerFlag != ""`, then falls
  through to the injected test seam and the production Go-native default.

The sibling commands dispatch on either flag: `runCrap`, `runQuality`, and
`runDocscan` all use `p.analyzerFlag != "" || p.languageFlag != ""`
(`runCrap` carries the explanatory comment "--language alone triggers
tier-3 PATH-convention discovery (gaze-analyzer-<language>), matching
runDocscan"). `gaze report` is therefore the only external-analyzer-capable
command that ignores `--language` alone (issue #282 — the D7 deferral
recorded in `openspec/changes/quality-classification-counts/`, whose
`runCrap`/`runQuality` fixes landed with PR #281).

Today a `--language`-only `gaze report` either fails or lies:

- Outside a Go module the run dies with `finding module root` — the same
  failure class issues #250/#257 fixed for `--analyzer` — because
  `resolveReportModuleDir` only takes its working-directory branch when
  `--analyzer` is set.
- Inside a Go module the flag is silently inert: the Go-native pipeline
  runs against Go source with no diagnostic, violating the constitution's
  Minimal Assumptions principle (explicit signals are "enforced at
  analysis entry points — never silently ignored").

The reference docs already promise the fixed behavior
(`docs/reference/cli/report.md`: "With `--analyzer` (or `--language` alone,
when discovery resolves an analyzer), the report is built from the external
analyzer's JSON-RPC data"), and the `/gaze` agent's layered detection
(quality-classification-counts design D4) supplies only `--language` for
heuristic-detected projects that rely on tier-3 discovery.

## Goals / Non-Goals

### Goals

- `gaze report --language <lang>` selects the external analyzer pipeline,
  with dispatch, module-dir resolution, and failure diagnostics identical
  to `gaze crap`, `gaze quality`, and `gaze docscan`
- Pin the dispatch contract with a hermetic regression test
- Keep the no-flag Go-native default byte-identical

### Non-Goals

- No `internal/aireport` changes — the external path already replaces the
  standard step sequence through the `opts.AnalyzeFunc` override (see D2)
- No protocol, schema, flag-surface, or `.gaze.yaml` changes
- No `gaze analyze` changes — that command's `--language` dispatch is a
  separate deferral (quality-classification-counts D12)
- No new discovery tiers or analyzer-contract changes

## Decisions

### D1: Widen both gate conditions in the extracted helpers

Both helpers switch their condition from `p.analyzerFlag != ""` to
`p.analyzerFlag != "" || p.languageFlag != ""` — the identical condition
the sibling commands already use:

- `resolveReportModuleDir` — `--language`-only runs use the working
  directory instead of requiring a Go module root.
- `applyReportAnalyzeFunc` — `--language`-only runs build the
  external-analyzer `AnalyzeFunc`.

The two gates must widen together: widening only `applyReportAnalyzeFunc`
leaves `--language`-only runs outside a Go module dying at
`resolveReportModuleDir` with the misleading module-root error; widening
only `resolveReportModuleDir` leaves the flag inert inside a Go module.
Each alone is a broken half-path.

Alternatives rejected:

- A single combined gate inside `runReport` before the helpers — the
  helpers are the established structure (extracted in a prior
  decomposition); each gate sits where its decision is consumed, and the
  helpers stay independently testable.
- Leaving `resolveReportModuleDir` unchanged — the module-root failure for
  external-analyzer runs is exactly the class issues #250/#257 fixed for
  `--analyzer`.

### D2: No `internal/aireport` changes

Issue #282 speculates that `runReport`'s
`runCRAPStep`/`runQualityStep`/`runDocscanStep` wiring "needs to honor the
language-only session". It does not: the external path replaces the entire
standard step sequence through the `opts.AnalyzeFunc` override
(`applyReportAnalyzeFunc` → `buildExternalReportAnalyzeFunc` → the external
report path), so those step functions never execute when the external path
is selected. Widening a gate condition in the two `cmd/gaze` helpers is the
complete fix.

The plumbing below the gates already threads `p.languageFlag` (landed with
PR #281): `applyReportAnalyzeFunc` passes it to
`buildExternalReportAnalyzeFunc`, which passes it into the shared session
initializer that calls `adapter.Discover(analyzerFlag, languageFlag, cfg)`
and returns the language-specific `no analyzer found for language %q`
error when the analyzer flag is empty and discovery fails. The fix is a
gate-condition widening, not new wiring.

The test-seam ordering in `applyReportAnalyzeFunc` is preserved: the
external gate remains ahead of the injected `p.analyzeFunc` branch.
Existing report tests that inject `analyzeFunc` set neither flag, so the
widened condition stays false for them — the seam's meaning is unchanged.

### D3: Hermetic regression test mirroring the sibling pattern

`TestReportWithLanguageOnly_TriggersExternalPath`
(`cmd/gaze/external_analyzer_test.go`) mirrors
`TestCrapWithLanguageOnly_TriggersExternalPath` /
`TestQualityWithLanguageOnly_TriggersExternalPath`:

- `t.Chdir(t.TempDir())` — no `go.mod`, so a `finding module root` error
  would prove the Go-native path was (incorrectly) taken.
- `languageFlag: "zz-nodiscover-test-language"` — the undiscoverable
  sentinel the sibling tests use, exercising tier-3 discovery failure.
- `format: "json"` — a native format, so the run needs no AI adapter
  (`buildReportAdapter` returns nil for non-text formats).
- Assert the error is non-nil, contains `no analyzer found for language`,
  and does not contain `finding module root`.

The unchanged paths are already pinned: the Go-native default by
`TestRunReport_GoNativePath_FindModuleRootFailure`, and the
`--analyzer`-alone external path by the existing external report tests
(e.g., `TestReportWithExternalAnalyzer_DocscanPopulated`).

### D4: Docs verification and the Website Documentation Gate

`docs/reference/cli/report.md` already documents the fixed behavior — the
`--language` flag row ("Target language for external analyzer discovery")
and the external-data paragraph ("With `--analyzer` (or `--language` alone,
when discovery resolves an analyzer), ..."). The task is verification:
adjust wording only where inaccurate; do not rewrite. README.md is
reviewed for the same. Because the behavior change is user-facing (a
previously ignored or failing flag combination now dispatches to external
discovery), the Website Documentation Gate applies: a tracking issue is
filed in `unbound-force/website`.

## Risks / Trade-offs

- **Behavior change for `--language`-plus-Go-module runs**: a user who ran
  `gaze report --language <lang>` inside a Go module previously received
  Go-native analysis; they now receive the external report (or the
  language-specific discovery error). Accepted: the flag's documented
  meaning is external discovery, the reference docs already promise the
  new behavior, and the constitution forbids silently ignoring the
  signal.
- **Test-seam interaction**: a test that both injects `analyzeFunc` and
  sets `languageFlag` would now take the external path. No existing test
  does (verified); the seam order is documented in D2.
- **No schema/protocol risk**: no serialization, protocol, or config
  surface is touched.

## Open Questions

- None material. The `gaze analyze` command's missing `--language`
  dispatch remains deferred by quality-classification-counts D12 and is
  out of scope.

## Constitution Alignment

Reference: assessed in proposal.md against the Gaze constitution v1.3.0 —
Accuracy PASS (reports come from the analyzer that can observe the
project, pinned by a regression test), Minimal Assumptions PASS
(`--language` enforced at the analysis entry point; no new flags or
config), Actionable Output PASS (the failure mode improves from the
misleading `finding module root` to the language-specific discovery
error), Testability PASS (two gate widenings verified by a hermetic
regression test; unchanged paths pinned by existing tests).
