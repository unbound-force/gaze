## 1. Classify and gate the Test workflow

- [x] 1.1 Add the `changes` job to `.github/workflows/test.yml`:
      job id `changes`, name `Changes`, `timeout-minutes: 5`,
      `permissions: contents: read`, the repo's SHA-pinned
      `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683`
      with `fetch-depth: 0`, and the single classify step from
      design D1, including the symlink check (job output `code`)
- [x] 1.2 Add `needs: changes` and
      `if: needs.changes.outputs.code != 'false'` to the
      `unit-and-integration` and `e2e` jobs (design D2), changing
      nothing else about them
- [x] 1.3 Add the `ci-ok` job to `.github/workflows/test.yml`
      per design D3 (`name: CI OK`, `needs: [changes,
      unit-and-integration, e2e]` — not `gaze-ai-report`,
      `if: ${{ !cancelled() }}`, `timeout-minutes: 5`, aggregate
      step reading `needs` from an env var, printing one
      `job: result` line per need, then `jq -e`)
- [x] 1.4 Leave `gaze-ai-report` and
      `.github/workflows/reusable_gaze_ai_report.yml` untouched
      (design D4); add a brief comment in `test.yml` noting the
      job skips transitively on learnings-only PRs and is
      advisory (outside `ci-ok`)
- [x] 1.5 Checkpoint: `actionlint` passes, `yamllint -c
      .yamllint.yml .github/workflows/test.yml
      .github/workflows/mega-linter.yml` passes, and
      `git diff --check` is clean

## 2. Gate MegaLinter

- [x] 2.1 Add the `changes` job to
      `.github/workflows/mega-linter.yml` with a classify step
      byte-identical to `test.yml`'s (design D5;
      `timeout-minutes: 5`, `permissions: contents: read`,
      `fetch-depth: 0`)
- [x] 2.2 Add `needs: changes` and
      `if: needs.changes.outputs.code != 'false'` to the
      `megalinter` job, changing nothing else about it
- [x] 2.3 Checkpoint: `actionlint` passes, `yamllint -c
      .yamllint.yml .github/workflows/mega-linter.yml` passes,
      and `git diff --check` is clean

## 3. Workflow contract test

- [x] 3.1 Add `cmd/gaze/ci_workflow_test.go` (design D8,
      CIW-FR-006; standard `testing` plus `gopkg.in/yaml.v3`,
      following `cmd/gaze/release_workflow_test.go`) asserting the
      `changes` jobs, the `!= 'false'` gates, the `ci-ok` name,
      condition, timeout and exact `needs` set, the
      needed-or-advisory drift guard (allowlist
      `gaze-ai-report`), and byte-identical classify steps
- [x] 3.2 Checkpoint: `go test -race -count=1 -run
      'TestCIWorkflow' ./cmd/gaze/` passes, and temporarily
      editing a gate or classify step in a scratch copy makes it
      fail

## 4. Documentation

- [x] 4.1 [P] Update `AGENTS.md` CI/CD section (Test and
      MegaLinter items): describe the `changes` classification
      job, job gating, the `CI OK` aggregate as the sole required
      check (advisory `gaze-ai-report` outside it), the contract
      test, and unchanged push-to-main behavior
- [x] 4.2 [P] Sweep `docs/guides/ci-integration.md` for
      statements invalidated by PR-time gating (claims that
      checks always run on PRs) and update where needed

## 5. Local verification

- [x] 5.1 Local CI parity before opening the PR: re-run the
      phase 1-3 checkpoint commands, plus `go build ./...` and
      `go test -race -count=1 -short ./...`

## 6. Constitution alignment verification

- [x] 6.1 Verify the proposal's alignment claims against the
      implemented change: the learnings-PR flow stays fully
      automated with no new human intervention (Autonomous
      Collaboration); no new third-party actions were introduced
      (Composability First); the `changes` log shows the
      changed-file list and verdict, and `CI OK` prints per-job
      results (Observable Quality); and every scenario in
      `specs/ci-workflow/spec.md` maps to the phase 3 contract
      test or a step of the design.md Rollout runbook
      (Testability)

## 7. Align with sibling repos

- [x] 7.1 Count only `.md` files under `.uf/dewey/learnings/` as
      learnings, and classify submodules (mode `160000`) as code,
      in both workflows' byte-identical classify step
- [x] 7.2 Add the `gitleaks` job (runs on `code=false`, pinned
      SHA-256, scans `base..head`) and add it to `ci-ok`'s needs
- [x] 7.3 Update the contract test: new label and mode pins,
      `ci-ok` needs, non-`.md` and submodule behavior scenarios,
      and `TestCIWorkflow_GitleaksScansLearningsOnlyPRs`
