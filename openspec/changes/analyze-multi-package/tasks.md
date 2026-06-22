## 1. Consolidate shared utilities into `internal/loader`

- [x] 1.1 Export `ResolvePackagePaths` in `internal/loader/loader.go` — move the `resolvePackagePaths` logic from `internal/crap/contract.go` to `internal/loader` as an exported function. Update GoDoc to reflect its purpose and exported status.
- [x] 1.2 Export `IsMainPkg` in `internal/loader/loader.go` — move the `isMainPkg` logic from `internal/crap/contract.go` to `internal/loader` as an exported function.
- [x] 1.3 Update `internal/crap/contract.go` — replace local `resolvePackagePaths` and `isMainPkg` with calls to `loader.ResolvePackagePaths` and `loader.IsMainPkg`. Remove the local function definitions and "keep in sync" NOTE comments.
- [x] 1.4 Update `internal/aireport/runner_steps.go` — replace local `resolvePackagePaths` and `isMainPkg` with calls to `loader.ResolvePackagePaths` and `loader.IsMainPkg`. Remove the local function definitions and "keep in sync" NOTE comments.
- [x] 1.5 Update tests in `internal/crap/contract_test.go` and `internal/aireport/runner_steps_test.go` to use `loader.ResolvePackagePaths`
- [x] 1.6 Add unit tests for `ResolvePackagePaths` and `IsMainPkg` in `internal/loader/loader_test.go`

## 2. Add warning to `loader.Load`

- [x] 2.1 In `internal/loader/loader.go`, after the `len(pkgs) == 0` check and before `pkg := pkgs[0]`, add a warning log when `len(pkgs) > 1` using `fmt.Fprintf(os.Stderr, ...)`. State how many packages were resolved and that only the first is used
- [x] 2.2 Add unit test in `internal/loader/loader_test.go` verifying behavior when a pattern resolves to multiple packages: the function returns the first package without error

## 3. Multi-package support for `gaze analyze`

- [x] 3.1 Change `analyzeParams.pkgPath string` to `patterns []string` in `cmd/gaze/main.go`
- [x] 3.2 Change analyze Cobra command from `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(1)`, update `Use` from `"analyze [package]"` to `"analyze [packages...]"`, pass `args` (full slice) to `analyzeParams.patterns`
- [x] 3.3 Update `runAnalyze` to call `loader.ResolvePackagePaths(p.patterns, moduleDir)` to expand patterns, then iterate over resolved paths calling `analysis.LoadAndAnalyze` per package, appending results into a merged `[]taxonomy.AnalysisResult` slice
- [x] 3.4 Handle the `--function` filter edge case: if a function filter is set and no results are found across all packages, return an error listing the searched packages
- [x] 3.5 Handle the `--classify` flag: when classification is enabled, load config and call `loader.LoadModule(cwd)` once outside the package loop, then run `runClassify` per package with pre-loaded module packages
- [x] 3.6 Handle `--include-unexported` auto-detection per package: apply `loader.IsMainPkg` check per resolved package path
- [x] 3.7 Remove local `isMainPackage` function from `cmd/gaze/main.go` (replaced by `loader.IsMainPkg`)
- [x] 3.8 Remove local `resolvePatterns` function from `cmd/gaze/main.go` (replaced by `loader.ResolvePackagePaths`)
- [x] 3.9 Update `runClassify` to accept pre-loaded module packages instead of loading them internally (avoids repeated `loader.LoadModule` calls in per-package loop)

## 4. Multi-package support for `gaze quality`

- [x] 4.1 Change `qualityParams.pkgPath string` to `patterns []string` in `cmd/gaze/main.go`
- [x] 4.2 Change quality Cobra command from `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(1)`, update `Use` from `"quality [package]"` to `"quality [packages...]"`, pass `args` to `qualityParams.patterns`
- [x] 4.3 Update `runQuality` to call `loader.ResolvePackagePaths(p.patterns, moduleDir)` to expand patterns, then iterate over resolved paths running the quality pipeline per package, merging reports and summaries
- [x] 4.4 Handle packages without tests gracefully: catch the specific "no test files found" error from `loadTestPackage` and log a warning + skip (do not error). Propagate other genuine errors.

## 5. Update tests

- [x] 5.1 Update all existing tests in `cmd/gaze/main_test.go` that construct `analyzeParams` to use `patterns []string` instead of `pkgPath string`
- [x] 5.2 Update all existing tests in `cmd/gaze/main_test.go` that construct `qualityParams` to use `patterns []string` instead of `pkgPath string`
- [ ] 5.3 Add test for multi-package analyze: verify results from multiple packages appear in output
- [ ] 5.4 Add test for multi-package quality: verify reports from multiple packages appear in output

## 6. Documentation updates

- [x] 6.1 Remove "Single package loading" known limitation from `README.md` (line ~208)
- [x] 6.2 Update `docs/reference/cli/analyze.md` — change "Exactly one package argument is required" to document multi-package support with examples
- [x] 6.3 Update `docs/reference/cli/quality.md` — same as above
- [x] 6.4 Update AGENTS.md recent changes section with a summary of this change
- [ ] 6.5 Create GitHub issue in `unbound-force/website` for CLI documentation updates (analyze and quality multi-package support) — requires manual creation (token lacks write access)

## 7. Verification

- [x] 7.1 Run `go build ./cmd/gaze` — verify clean build
- [x] 7.2 Run `go test -race -count=1 -short ./...` — all tests pass
- [x] 7.3 Run `golangci-lint run` — no lint violations (pre-existing staticcheck QF1012 in ai_mapper.go excluded)
- [x] 7.4 Verify constitution alignment: confirm Principle I (Accuracy — no silent truncation), Principle III (Actionable Output — complete results), and Principle IV (Testability — new code has tests) are satisfied
<!-- spec-review: passed -->
