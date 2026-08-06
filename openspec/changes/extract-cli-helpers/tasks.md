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

## 1. Create `internal/cliutil` Package

- [x] 1.1 [P] Create `internal/cliutil/cliutil.go` with `ValidateFormat` and `CaptureJSON` — package doc comment, GoDoc on both exported functions, imports only `bytes`, `encoding/json`, `fmt`, `io`
- [x] 1.2 [P] Create `internal/cliutil/cliutil_test.go` with table-driven tests: `ValidateFormat` (valid "text", valid "json", invalid "csv", empty string, error message format); `CaptureJSON` (success path, error propagation, empty output)

## 2. Migrate `cmd/gaze/main.go`

- [x] 2.1 Replace the 4 inline format validation blocks (lines 210, 498, 1057, 1493) with `cliutil.ValidateFormat(p.format)` calls; add `cliutil` import
- [x] 2.2 Extract `autoDetectMainPkg(pkgPath string, includeUnexported *bool)` private function; replace the 2 inline blocks (lines 269–271, 1126–1128) with calls to it
- [x] 2.3 Delete `loadGazeConfigBestEffort` function (lines 800–809); replace 3 call sites (lines 657, 730, 781) with `config.LoadFromDir(moduleDir)`
- [x] 2.4 Delete `captureReportJSON` function (lines 1794–1800); replace its call site with `cliutil.CaptureJSON`; remove the acknowledging comment at line 1793

## 3. Migrate `internal/aireport/runner.go`

- [x] 3.1 [P] Delete `captureJSON` function (lines 342–348); replace its call site(s) with `cliutil.CaptureJSON`; add `cliutil` import

## 4. Verify

- [x] 4.1 Run `go build ./cmd/gaze` — MUST compile cleanly
- [x] 4.2 Run `go test -race -count=1 -short ./...` — all tests MUST pass
- [x] 4.3 Run `golangci-lint run` — no new lint findings
- [x] 4.4 Verify constitution alignment: no behaviour changes (Principle I), no new assumptions (Principle II), output unchanged (Principle III), new helpers are unit-tested in isolation (Principle IV)
<!-- spec-review: passed -->
