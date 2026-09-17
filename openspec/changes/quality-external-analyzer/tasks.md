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

## 1. Build Quality Reports from External Mappings

- [x] 1.1 Add `BuildQualityFromMappings` function in `internal/adapter/quality.go` that converts `[]protocol.AssertionMappingData` + `[]taxonomy.AnalysisResult` into `[]taxonomy.QualityReport` and `*taxonomy.PackageSummary`. Use `quality.ComputeContractCoverage` for contract coverage computation. Group mappings by `test_function`, compute over-specification from incidental-classified assertions, populate gap hints. Set `AssertionDetectionConfidence` to 0 (external analyzers provide per-mapping confidence, not aggregate detection confidence).
- [x] 1.2 [P] Add `internal/adapter/quality_test.go` with table-driven tests for `BuildQualityFromMappings`: (a) mappings with full classification → correct contract coverage and over-specification, (b) no mappings → empty reports with zero coverage, (c) unclassified effects → treated as contractual, (d) mappings targeting incidental effects → over-specification counted, (e) multiple test functions → separate QualityReport per test, (f) summary aggregation correct.

## 2. Wire External Analyzer Path in CLI

- [x] 2.1 In `cmd/gaze/main.go`: remove the `--analyzer` rejection block in `runQuality`, add `runQualityWithExternalAnalyzer` function that follows the `runCrapWithExternalAnalyzer` pattern: call `initExternalSession`, fetch side effects via `providers.SideEffects.AllResults()`, fetch test mappings via `providers.ContractCoverage` (or direct `test_mapping` call), call `BuildQualityFromMappings`, then pass results to `writeQualityReport`. Add flag validation: reject `--target` and `--ai-mapper` when `--analyzer` is set (design D6/D7).
- [x] 2.2 In `cmd/gaze/main.go` `newQualityCmd`: remove the two `cmd.Flags().MarkHidden` calls for `analyzer` and `language` flags to unhide them in `--help`.

## 3. Graceful Degradation

- [x] 3.1 In `runQualityWithExternalAnalyzer`: when `providers.Capabilities.TestMapping` is `false`, print warning to stderr ("test_mapping not supported by analyzer; contract coverage unavailable"), produce quality report with side effects but zero contract coverage and reason `"test_mapping_unavailable"`, exit 0 unless threshold flags are set (in which case, return threshold violation error).

## 4. Tests

- [x] 4.1 Add CLI-level tests in `cmd/gaze/main_test.go` (or a new `cmd/gaze/quality_external_test.go`): (a) `--analyzer` + `--target` returns error, (b) `--analyzer` + `--ai-mapper` returns error, (c) `--analyzer` flag no longer rejected (requires fake analyzer — use the existing `internal/protocol/testdata/fake_analyzer/` pattern). Verify output schema compliance for JSON format.
- [x] 4.2 [P] Add `internal/adapter/quality_degraded_test.go`: test `BuildQualityFromMappings` with nil/empty mappings to verify graceful degradation produces valid zero-coverage reports.

## 5. Documentation

- [x] 5.1 Update `README.md` if `gaze quality` flags or usage examples are documented — add `--analyzer` and `--language` usage.
- [x] 5.2 Update `AGENTS.md` Recent Changes section with a summary of this change.

## 6. Verification

- [x] 6.1 Run `go test -race -count=1 -short ./...` and `golangci-lint run` — all must pass.
- [x] 6.2 Verify `gaze quality --help` shows `--analyzer` and `--language` flags (no longer hidden).
- [x] 6.3 Verify constitution alignment: Composability (external analyzer remains optional — Go-native path unchanged), Observable Quality (JSON output conforms to existing schema), Testability (new functions tested in isolation with synthetic data).
- [x] 6.4 Verify new functions in `internal/adapter/quality.go` have CRAP score < 30 (if gaze is available: `gaze crap -- ./internal/adapter/`).

<!-- spec-review: passed -->
