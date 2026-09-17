# Tasks

<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Protocol Extension

- [x] 1.1 Add `MethodDocCoverage` constant (`"doc_coverage"`) to `internal/protocol/types.go`
- [x] 1.2 Add `DocCoverage bool` field to `Capabilities` struct in `internal/protocol/types.go` with JSON tag `"doc_coverage"`
- [x] 1.3 Add `DocCoverageParams`, `DocCoverageResult`, and `SymbolDocStatus` types to `internal/protocol/types.go`
- [x] 1.4 [P] Add `TestRoundTrip_DocCoverageResult` test in `internal/protocol/types_test.go`
- [x] 1.5 Update `docs/protocol.md` with `doc_coverage` method documentation, updated capability table, `SymbolDocStatus` type reference, and lifecycle diagram. Update the package doc comment in `internal/protocol/types.go` to reflect the correct method count (10 methods: 5 required + 5 optional).

Note: Tasks 1.1, 1.2, and 1.3 all modify `types.go` and MUST run sequentially. Task 1.4 depends on 1.3.

## 2. `apidoc` Sub-Package — Types and Coverage

- [x] 2.1 Create `internal/docscan/apidoc/types.go` — define `AnalyzerData`, `APICoverageReport`, `SymbolCoverage`, `StaleReference`, `CodeBlockIssue` types with JSON tags
- [x] 2.2 Create `internal/docscan/apidoc/coverage.go` — implement `ComputeCoverage` function returning `(*CoverageResult, error)` handling both native `DocCoverageResult` and heuristic fallback from `[]AnalyzedFunction`
- [x] 2.3 [P] Create `internal/docscan/apidoc/coverage_test.go` — test `ComputeCoverage` with synthetic `DocCoverageResult` (native path) and synthetic `[]AnalyzedFunction` + Markdown content (heuristic path); test nil-data no-op; test zero-symbol edge case

## 3. `apidoc` Sub-Package — Validation

- [x] 3.1 [P] Create `internal/docscan/apidoc/validation.go` — implement `ValidateReferences` (backtick-quoted symbol scanning, stale detection) and `ValidateCodeBlocks` (fenced code block language tag validation with generic-tag ignore list)
- [x] 3.2 [P] Create `internal/docscan/apidoc/validation_test.go` — test stale reference detection (renamed symbol, valid symbol, non-symbol backtick content), code block validation (wrong language, untagged, matching, generic tags)

## 4. `apidoc` Sub-Package — Analyze Orchestration

Depends on: Groups 2 and 3 (coverage + validation functions must exist).

- [x] 4.1 Create `internal/docscan/apidoc/analyze.go` — implement `Analyze(docs, data)` function that orchestrates `ComputeCoverage`, `ValidateReferences`, `ValidateCodeBlocks`, and assembles `APICoverageReport`
- [x] 4.2 [P] Create `internal/docscan/apidoc/analyze_test.go` — test full `Analyze` function: nil data returns nil, native doc_coverage path, heuristic path, combined coverage + stale refs + code block issues

## 5. Docscan Output Type and Integration

Depends on: Group 4 (`apidoc.Analyze` must exist).

- [x] 5.1 Add `DocscanOutput` struct to `cmd/gaze/main.go` (CLI layer) with `Documents []docscan.DocumentFile` and `APICoverage *apidoc.APICoverageReport` fields — NOT in `internal/docscan/` (would create circular import with `apidoc`)
- [x] 5.2 Update `runDocscan` in `cmd/gaze/main.go` — add `--analyzer` and `--language` flags, create adapter session (with `defer session.Close()`), call `session.DocCoverage()` (gated on capabilities), call `apidoc.Analyze`, output `DocscanOutput` JSON. If `doc_coverage` call fails at runtime, fall back to heuristic with warning.
- [x] 5.3 [P] Add tests for `runDocscan` with and without analyzer in `cmd/gaze/main_test.go` or dedicated test file — verify JSON output structure, nil api_coverage when no analyzer

## 6. Report Pipeline Integration

Depends on: Groups 4 and 5 (`apidoc.Analyze` and `DocscanOutput` must exist).

- [x] 6.0 Add `DocCoverage(ctx context.Context, params protocol.DocCoverageParams) (*protocol.DocCoverageResult, error)` method to `adapter.Session` in `internal/adapter/session.go`, gated on `s.caps.DocCoverage`. Uses `callAndUnmarshal` pattern. Returns nil when capability is false.
- [x] 6.1 Update `runDocscanStep` signature in `internal/aireport/runner_steps.go` to accept `*adapter.Session` parameter; when non-nil, call `session.DocCoverage()` and `analyze` (accept performance cost of duplicate call), then call `apidoc.Analyze`
- [x] 6.2 Update `pipelineStepFuncs` type and `runProductionPipeline` in `internal/aireport/runner.go` to pass the analyzer session to the docscan step. Add `AnalyzerSession *adapter.Session` to `RunnerOptions` or thread the session through `runProductionPipeline` parameter.
- [x] 6.3 [P] Update `runDocscanStep` tests in `internal/aireport/runner_steps_test.go` for the new signature (nil session path)
- [x] 6.4 [P] Update pipeline internal tests in `internal/aireport/pipeline_internal_test.go` for the new `docscanStep` function type
- [x] 6.5 [P] Update `compactDocscanField` in `internal/aireport/compact.go` to unmarshal the new `DocscanOutput` envelope instead of bare `[]DocumentFile` array — compact the `Documents` slice, pass through `APICoverage`. Update the 4 docscan-related tests in `compact_test.go`.

## 7. Fake Analyzer Update

Depends on: Group 1 (protocol types must exist).

- [x] 7.1 [P] Update fake analyzer binary at `internal/protocol/testdata/fake_analyzer/` to support `doc_coverage` method — respond to capability announcement and return synthetic `DocCoverageResult`
- [x] 7.2 [P] Add integration test in `internal/adapter/` verifying that when the analyzer announces `DocCoverage: true`, the session can call `doc_coverage` and unmarshal the result

## 8. Documentation and Verification

- [x] 8.1 Update `AGENTS.md` — add `internal/docscan/apidoc` to Architecture section, mention `doc_coverage` protocol method in Key Patterns
- [x] 8.2 Update `README.md` — add `--analyzer` flag documentation for `gaze docscan`, document the `api_coverage` JSON output section, **document the breaking change from bare `[]DocumentFile` array to `DocscanOutput` structured object with migration guidance**
- [x] 8.3 Verify constitution alignment — run `go test -race -count=1 -short ./...` and `golangci-lint run`, confirm all tests pass, confirm Principle I (Accuracy: native doc_coverage tested), Principle II (Minimal Assumptions: nil analyzer graceful fallback tested), Principle III (Actionable Output: JSON output includes specific symbols/files/lines), Principle IV (Testability: all apidoc functions tested with synthetic data)
- [x] 8.4 File website documentation issue — `gh issue create --repo unbound-force/website --title "docs: gaze docscan --analyzer flag and api_coverage JSON output"` tracking: new --analyzer/--language flags, api_coverage JSON section, breaking change from bare array to structured object, doc_coverage protocol method
- [x] 8.5 Verify GoDoc completeness — confirm package-level doc comment on `internal/docscan/apidoc`, GoDoc comments on all exported types (`APICoverageReport`, `SymbolCoverage`, `StaleReference`, `CodeBlockIssue`, `AnalyzerData`, `CoverageResult`) and functions (`Analyze`, `ComputeCoverage`, `ValidateReferences`, `ValidateCodeBlocks`)
<!-- spec-review: passed -->
<!-- code-review: passed -->
