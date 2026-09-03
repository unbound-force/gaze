<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file --
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Defer FindModuleRoot in newCrapCmd

- [x] 1.1 In `cmd/gaze/main.go`, in `newCrapCmd`'s `RunE` closure (lines 924-957): Remove the `loader.FindModuleRoot(cwd)` call (lines 929-932). Pass `cwd` (already obtained from `os.Getwd()` at line 925) directly as the `moduleDir` field in the `crapParams` struct (line 948). The three lines `moduleDir, err := loader.FindModuleRoot(cwd)` / `if err != nil {` / `return fmt.Errorf("finding module root: %w", err)` / `}` are removed entirely.

## 2. Add FindModuleRoot to runCrap Go-native path

- [x] 2.1 In `cmd/gaze/main.go`, in `runCrap` (line 493): After the `if p.analyzerFlag != ""` dispatch block (lines 501-503) and before the contract coverage provider setup (line 509), add a `loader.FindModuleRoot` call. Insert:
  ```go
  // Resolve Go module root for Go-native analysis.
  moduleDir, err := loader.FindModuleRoot(p.moduleDir)
  if err != nil {
      return fmt.Errorf("finding module root: %w", err)
  }
  p.moduleDir = moduleDir
  ```
  This ensures all downstream uses of `p.moduleDir` (`crap.Analyze` at line 535, `resolveBaselineAndCompare` at line 563) receive the resolved module root.

## 3. Add Test

- [x] 3.1 **(Scenario 1: external analyzer bypasses FindModuleRoot)** Add a test in `cmd/gaze/main_test.go` verifying that `runCrap` with `analyzerFlag` set does not require `FindModuleRoot`. Construct a `crapParams` with `analyzerFlag` set to `"nonexistent-analyzer"`, `moduleDir` set to `os.TempDir()` (a directory without `go.mod`), `patterns` set to `[]string{"."}`, and valid `stdout`/`stderr` writers (`bytes.Buffer`). Call `runCrap(p)`. Assert that: (a) the returned error is non-nil, (b) the error does NOT contain `"finding module root"` or `"no go.mod found"`, and (c) the error DOES contain an analyzer-related substring (e.g., `"discovering analyzer"` or `"not found"` from `adapter.Discover` or `exec.LookPath`). This proves the external analyzer path bypasses `FindModuleRoot` and reaches the analyzer discovery logic.
- [x] 3.2 **(Scenario 3: FindModuleRoot failure in Go-native path)** Add a test in `cmd/gaze/main_test.go` verifying that `runCrap` without `analyzerFlag`, with `moduleDir` set to a temp directory without `go.mod`, returns an error containing `"finding module root"`. This proves the `FindModuleRoot` call was moved into `runCrap` and the error wrapping format is preserved.
- [x] 3.3 **(Scenario 2: Go-native path regression)** Verify that existing tests exercising the Go-native CRAP path (e.g., `TestRunCrap*` tests that use valid Go module directories) continue to pass after the code move. No new test needed — the existing tests exercise `runCrap` with valid `moduleDir` values, and the new `FindModuleRoot` call inside `runCrap` is idempotent when given a directory that already is a module root. Confirm by running `go test -race -count=1 -short ./cmd/gaze/...` and verifying all existing CRAP tests pass.

## 4. Verification

- [x] 4.1 Run `go build ./...` to verify compilation.
- [x] 4.2 Run `go test -race -count=1 -short ./cmd/gaze/...` to verify crap command tests pass.
- [x] 4.3 Run `go test -race -count=1 -short ./...` to verify full short test suite passes.
- [x] 4.4 Run `golangci-lint run` to verify no lint violations (or `go vet ./...` if golangci-lint is not installed locally).
- [x] 4.5 Constitution alignment: verify Principle II (Minimal Assumptions) — `gaze crap --analyzer` no longer assumes a Go module exists.
<!-- spec-review: passed -->
<!-- code-review: passed -->
