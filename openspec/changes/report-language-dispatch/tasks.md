# Tasks: Report Language-Only External Dispatch

## 1. Dispatch gate widening (cmd/gaze)

- [x] 1.1 Widen the two external-analyzer gates in `runReport`'s extracted
  helpers (`cmd/gaze/main.go`) from `p.analyzerFlag != ""` to
  `p.analyzerFlag != "" || p.languageFlag != ""` — the identical condition
  `runCrap`/`runQuality`/`runDocscan` already use (design D1):
  `resolveReportModuleDir` (`--language`-only runs use the working
  directory instead of requiring a Go module root) and
  `applyReportAnalyzeFunc` (`--language`-only runs build the
  external-analyzer `AnalyzeFunc` through the existing
  `buildExternalReportAnalyzeFunc` plumbing — no `internal/aireport`
  changes, design D2). Update both helpers' GoDoc comments to state that
  either flag selects the external path, preserving the
  injected-`analyzeFunc` seam ordering after the external gate.
- [ ] 1.2 [P] Add `TestReportWithLanguageOnly_TriggersExternalPath` to
  `cmd/gaze/external_analyzer_test.go`, mirroring
  `TestCrapWithLanguageOnly_TriggersExternalPath` /
  `TestQualityWithLanguageOnly_TriggersExternalPath` (design D3):
  `t.Chdir(t.TempDir())` (no `go.mod`, so a module-root error would prove
  the Go-native path was taken), `languageFlag:
  "zz-nodiscover-test-language"` (the undiscoverable sentinel the sibling
  tests use), `format: "json"` (native format — no AI adapter needed);
  assert the error is non-nil, contains `no analyzer found for language`,
  and does not contain `finding module root`.

## 2. Documentation

- [ ] 2.1 [P] Verify documentation matches the shipped behavior (design
  D4, Documentation Validation Gate): `docs/reference/cli/report.md` (the
  `--language` flag row and the external-data paragraph already promise
  the fixed behavior — adjust only where inaccurate), `README.md`, and
  the `--language` flag help text; file the Website Documentation Gate
  issue in `unbound-force/website` for the user-facing behavior change
  (a previously ignored or failing flag combination now dispatches to
  external analyzer discovery).

## 3. Verification

- [ ] 3.1 [P] `openspec validate report-language-dispatch` passes with no
  issues other than the expected advisory for the `originating_issue`
  metadata key (required by /uf.finale for PR-body issue threading;
  unrecognized — and ignored — by openspec). Any requirement, scenario,
  or artifact issue is a failure.
- [ ] 3.2 [P] CI parity per the CI Parity Gate (derive exact commands
  from `.github/workflows/test.yml` and
  `.github/workflows/mega-linter.yml`): at minimum `go build ./...`,
  `go test -race -count=1 -short ./cmd/gaze/...`, the full
  `go test -race -count=1 -short ./...`, and `golangci-lint run` all
  pass.
- [ ] 3.3 [P] Constitution alignment re-verified against the implemented
  code: the proposal's assessments hold — `--language` is enforced at
  the analysis entry point (Minimal Assumptions), language-only reports
  come from the analyzer that can observe the project with a
  language-specific failure diagnostic (Accuracy, Actionable Output),
  and the dispatch contract is pinned by the hermetic regression test
  (Testability).
- [ ] 3.4 Spec artifacts committed and pushed before implementation
  begins (Spec Commit Gate).
