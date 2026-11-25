<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency on
  prior tasks in the group, (c) can safely execute without
  ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Reusable workflow: external-analyzer mode

- [x] 1.1 Add the `analyzer`, `language`, `packages`, `setup`,
      `node-version`, `python-version`, and `coverage-path` inputs
      to `.github/workflows/reusable_gaze_ai_report.yml` with the
      Go-preserving defaults from design D2; leave the
      `gaze-version` default untouched (CW-FR-100).
- [x] 1.2 Gate the report step by mode in
      `.github/workflows/reusable_gaze_ai_report.yml`
      (CW-FR-101, design D1): analyzer mode passes `--analyzer`,
      `--language` when non-empty, and the `packages` pattern —
      all consumed through `env:` bindings referenced as quoted
      POSIX variables — and omits `--coverprofile`; Go mode keeps
      the current command.
- [ ] 1.3 Add the caller `setup` evaluation at the top of the
      report step's `run:` block and the SHA-pinned
      `setup-node`/`setup-python` steps in
      `.github/workflows/reusable_gaze_ai_report.yml`
      (CW-FR-102, CW-FR-104; design D3, D5), and document the
      caller-trust model in the workflow header comment.
- [ ] 1.4 Make coverage-artifact validation mode-dependent in
      `.github/workflows/reusable_gaze_ai_report.yml`
      (CW-FR-103, design D4): Go mode keeps the coverprofile file
      check; the skipped-report path checks `coverage-path` with a
      file-or-directory existence check in analyzer mode.
- [ ] 1.5 Add a structural workflow-contract test that parses
      `.github/workflows/reusable_gaze_ai_report.yml` and asserts
      the new inputs, their defaults, the mode gating, and the
      `env:`-binding pattern (CW-FR-105; follow the
      `release_workflow_test.go` discipline).
- [ ] 1.6 Checkpoint: `go build ./cmd/gaze` and
      `go test -race -count=1 -short ./...` pass, and the
      workflow file passes the repo's YAML lint.

## 2. Scaffold: project flavor detection

- [ ] 2.1 Implement marker detection in
      `internal/scaffold/scaffold.go` (CS-FR-100, design D7):
      precedence `go.mod` > `package.json` > `pyproject.toml`;
      reword the no-marker warning to name the three markers; the
      Go flavor path stays byte-identical.
- [ ] 2.2 [P] Create
      `internal/scaffold/assets/ci/gaze-ai-report-node.yml`
      (CS-FR-101, design D9) and its golden
      `internal/scaffold/testdata/golden-gaze-ai-report-node.yml`.
- [ ] 2.3 [P] Create
      `internal/scaffold/assets/ci/gaze-ai-report-python.yml`
      (CS-FR-102, design D10 — verify the snake-eyes commands and
      analyzer identifier) and its golden
      `internal/scaffold/testdata/golden-gaze-ai-report-python.yml`.
- [ ] 2.4 Extend `internal/scaffold/ci_test.go` (and
      `cmd/gaze/main_test.go` where summary assertions live) with
      flavor-selection, go.mod-precedence, no-marker-warning,
      golden-match, and next-steps tests (CS-FR-100..CS-FR-104);
      assert the Go golden stays byte-identical.
- [ ] 2.5 Checkpoint: `go test -race -count=1 -short
      ./internal/scaffold/... ./cmd/gaze/...` passes.

## 3. TypeScript fixture job

- [ ] 3.1 Add the fixture job to `.github/workflows/test.yml`
      (CW-FR-105, design D11): check out gaze at the triggering
      ref, check out reading-stone at a pinned ref into a
      subdirectory, build gaze from the commit under test, run the
      caller setup and `gaze report ./src/...
      --analyzer=gaze-analyzer-typescript --format=json`, and
      assert exit 0.
- [ ] 3.2 Checkpoint: `go test -race -count=1 -short ./...` and
      `golangci-lint run` pass locally, and the workflow YAML
      parses.

## 4. Documentation and gates

- [ ] 4.1 Update `docs/guides/ci-integration.md`: an
      external-analyzer section covering the new inputs, the
      mode-dependent coverage contract, `setup` handling with the
      caller-trust model (design D3), and the Node/Python scaffold
      flavors.
- [ ] 4.2 [P] Update `docs/reference/cli/report.md` and
      `docs/protocol.md` where the external-analyzer CI invocation
      is documented.
- [ ] 4.3 [P] Update `README.md`'s CI Integration and
      Multi-Language Support sections for the external-analyzer CI
      path.
- [ ] 4.4 [P] Website Documentation Gate (AGENTS.md): file the
      `unbound-force/website` documentation issue via `gh issue
      create --repo unbound-force/website` covering the new
      `gaze init --ci` flavors and the reusable workflow's new
      inputs, before the implementing PR merges.
- [ ] 4.5 [P] Add the AGENTS.md Recent Changes entry for this
      change; confirm no embedded `.opencode/` assets were touched
      (File Sync Invariants unaffected).

## 5. Verification

- [ ] 5.1 CI parity (AGENTS.md Technical Guardrails): derive the
      exact commands from `.github/workflows/test.yml` and the
      MegaLinter configuration, then run them locally —
      `go build ./cmd/gaze`,
      `go test -race -count=1 -short ./...`, the e2e self-check
      suite (`go test -race -count=1 -run 'TestRunSelfCheck'
      -timeout 30m ./cmd/gaze/...`), and `golangci-lint run` —
      and fix any failures.
- [ ] 5.2 Constitution alignment verification (proposal
      Constitution Alignment section): confirm each principle's
      evidence in the implementation — I (self-describing inputs,
      goldens), II (Go behavior unchanged, opt-in mode), III
      (observable workflow contract, fixture job), IV
      (golden-pinned flavors, structural assertions) — and record
      the outcome in the implementation summary.
