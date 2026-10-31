# Tasks: `gaze init --ci`

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

## 1. Scaffold core (template, version resolution, write path)

- [ ] 1.1 [P] Create `internal/scaffold/assets/ci/gaze-ai-report.yml`
  (design D9): provenance `#` header comment (scaffolded by
  `gaze init --ci`, pointer to `docs/guides/ci-integration.md`, note
  to adjust `on:` branch filters for non-`main` default branches, and
  a note that the `uses:` tag pin can be hardened to the tag's
  40-character commit SHA for immutable pinning);
  `name: Gaze AI Report`; `on: push`/`pull_request` filtered to
  `branches: [main]`; workflow-level `permissions: contents: read`
  plus a job-level `permissions: contents: read` grant on the
  `gaze-ai-report` job (CI-021 call-site scoping; the single decided
  placement shape shared with spec.md and design D9);
  CI-012 concurrency group (`group: ${{ github.ref }}-${{
  github.workflow }}`, `cancel-in-progress: ${{ github.ref !=
  'refs/heads/main' }}`); `test` job (`timeout-minutes: 30`, SHA
  pins `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
  # v4.2.2` and
  `actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff #
  v5.6.0` with `go-version: "__GO_VERSION__"`,
  `go test -race -count=1 -short -coverprofile=coverage.out ./...`,
  `actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
  # v4.6.2` with `name: coverage-out`, `path: coverage.out`);
  `gaze-ai-report` job (`needs: test`, `uses:
  unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@__GAZE_VERSION__`,
  `with: coverage-artifact: coverage-out` and
  `gaze-version: __GAZE_VERSION__` (the resolved, validated version
  threads into the reusable workflow's binary install, keeping the
  CI-installed gaze identical to the `uses:` pin and to the binary
  the consumer calibrates thresholds with) plus the three threshold
  inputs as comment-only TODO lines with no numeric values,
  `secrets: opencode-config: ${{ secrets.OPENCODE_CONFIG_CONTENT }}`).
  The template must pass the repo's `.yamllint.yml`.
- [ ] 1.2 [P] In `internal/scaffold/scaffold.go` (design D4, D11):
  add `CI bool` and `GazeVersion string` to `Options` with GoDoc
  comments; add the named floor constant (`v2.3.0`, commented as the
  first release tag containing the reusable workflow); add version
  resolution (`GazeVersion` override else build-time `Version`)
  validated with `golang.org/x/mod/semver` (`IsValid` +
  `Compare >= floor`), promoting `golang.org/x/mod` from indirect to
  direct in `go.mod` (`go mod tidy`); on invalid, pre-floor, `dev`,
  or empty resolution, return a fail-closed error naming the floor
  and directing the user to install from a release tag or pass
  `--gaze-version` — validated in `Run` **before** the `.opencode/`
  asset walk so a failed `--ci` run writes nothing.
- [ ] 1.3 In `internal/scaffold/scaffold.go` (design D5, D6): add a
  minimal line-scan parser for the `go` directive of
  `<target>/go.mod` (fallback `1.25` with a warning when the
  directive is unreadable); when `go.mod` is absent and `--ci` was
  requested, print a distinct warning naming the
  Go-module-at-repository-root requirement, skip CI scaffolding
  (non-fatal), and let the `.opencode/` scaffolding proceed.
- [ ] 1.4 In `internal/scaffold/scaffold.go` (design D7, D8): skip
  the `ci/` prefix in `Run`'s asset walk and in `assetPaths()` so no
  `.opencode/ci/` mirror is ever written and
  `TestEmbeddedAssetsMatchSource` enumerates only the nine mirrored
  assets; add the dedicated CI write path — read the embedded
  template, substitute `__GAZE_VERSION__` and `__GO_VERSION__` via
  `strings.ReplaceAll`, then apply exists/`--force` semantics
  (MkdirAll `.github/workflows/` 0o755, write 0o644), reporting
  through the existing `Result.Created`/`Skipped`/`Overwritten`
  fields with display path `.github/workflows/gaze-ai-report.yml`.
  The CI file is user-owned: never added to `isToolOwned`.
- [ ] 1.5 In `internal/scaffold/scaffold.go` (design D10): print the
  CI next-steps section through the existing `opts.Stdout` writer
  when CI scaffolding was requested and not skipped for a missing
  `go.mod`: the `gh secret set OPENCODE_CONFIG_CONTENT` one-liner
  (pointer to `docs/guides/ci-integration.md` for the config shape),
  the `jq -c .` single-line minify tip, the second-secret masking
  pattern pointer, and the paste-able threshold-guidance prompt (read
  the generated workflow, run `gaze report ./... --format=json`,
  propose permissive initial thresholds from the summary metrics,
  human approves final values, uncomment the threshold inputs).

## 2. CLI wiring

- [ ] 2.1 In `cmd/gaze/main.go` (design D11): add `ci` and
  `gazeVersion` to `initParams`; register `--ci` and
  `--gaze-version` flags on the init command; reject
  `--gaze-version` when `--ci` is not set; the `--gaze-version` flag
  help text states the value must be an existing gaze release tag
  `>= v2.3.0` (validation is offline and lexical — it cannot prove
  tag existence); thread both through
  `runInit` into `scaffold.Options`; update the command `Long` help
  text to mention the CI scaffolding and the Go-module-at-root
  requirement.

## 3. Tests

- [ ] 3.1 Version-resolution matrix (design D4), as a table-driven
  test of the pure resolution/validation function in
  `internal/scaffold/`: valid release `v2.4.0` passes; `v2.3.0`
  (floor, inclusive) passes; pre-release `v2.4.0-rc1` passes
  (contains the workflow); `v2.3.0-rc1` (a pre-release of the floor
  tag itself) fails — decided behavior: it sorts below `v2.3.0` by
  semver precedence (see design Risks); `v2.2.0` and `v2.2.9` fail;
  `dev`, empty,
  and malformed strings (`2.3.0` without `v`, `vv2.3.0`,
  `v2.3`) fail; `--gaze-version` override takes precedence over the
  build-time version; each failure returns the guidance text and the
  floor value.
- [ ] 3.2 Golden-file and YAML-parse validation (design D9): with
  fixed version/go-version inputs, the generated
  `.github/workflows/gaze-ai-report.yml` matches a committed golden
  file byte-for-byte (both placeholders substituted, none remaining),
  parses as YAML (via the existing `gopkg.in/yaml.v3` direct
  dependency), contains the expected `uses:` reference, the `with:`
  block's `coverage-artifact` and `gaze-version` inputs (the
  `gaze-version` value identical to the `uses:` pin), the
  workflow-level `permissions: contents: read` grant plus the
  `gaze-ai-report` job's job-level grant, SHA pins,
  CI-012 concurrency block, and comment-only threshold lines; add an
  assertion that the template's `uses:` path and the floor constant
  reference `reusable_gaze_ai_report.yml` / `v2.3.0` so a workflow
  rename or floor change cannot silently diverge from the template.
- [ ] 3.3 Skip-if-present semantics (design D7): scaffold into a temp
  dir, hand-edit the generated file (uncomment a threshold), rerun
  `--ci` without `--force` — file byte-identical and reported
  skipped (exists); rerun with `--force` — regenerated and reported
  overwritten.
- [ ] 3.4 Flag-gating behavior: plain `gaze init` (no `--ci`) writes
  no workflow file and produces `.opencode/` output identical to the
  pre-change manifest; `gaze init --ci` with a valid version in a
  Go-module-at-root temp dir writes both; `--ci` without `go.mod`
  prints the CI-skip warning, writes no workflow, and completes the
  `.opencode/` scaffolding; failed version validation (dev build,
  no override) writes no files at all (fail-closed before the walk);
  the combined case — dev build (or invalid override) in a directory
  without `go.mod` — fails on the version error before the go.mod
  check (no files written, no skip warning, version-validation
  guidance printed); end-to-end directive substitution: a temp dir
  whose `go.mod` declares `go 1.24` yields a generated file
  containing `go-version: "1.24"`, and a temp dir whose `go.mod` has
  no parseable directive yields `1.25` in the generated file with the
  fallback warning printed.
- [ ] 3.5 Go-mod directive parser matrix (design D5), as a
  table-driven test of the pure parser function in
  `internal/scaffold/`: `go 1.25` and `go 1.25.0` pass through;
  trailing whitespace and a trailing `\r` (CRLF go.mod) are stripped
  before matching; a go.mod with no `go` directive falls back to
  `1.25` with a warning; garbage directive values (`go banana`,
  `go 1.25" # note`) fall back to `1.25` with a warning — a value
  outside the accepted grammar is never emitted into the generated
  workflow.
- [ ] 3.6 CI next-steps section assertions (design D10): a successful
  `--ci` run's stdout contains all four items — the
  `gh secret set OPENCODE_CONFIG_CONTENT` one-liner, the `jq -c`
  minify tip, the second-secret masking pointer, and the
  threshold-guidance prompt; a rerun that reports the file as
  skipped (exists) still prints the section; a run skipped for a
  missing `go.mod` prints none of it (suppression).
- [ ] 3.7 [P] Update `internal/scaffold/scaffold_test.go`:
  `TestEmbeddedAssetsMatchSource` and related mirror tests remain
  green with the `ci/` subtree excluded from `assetPaths()`; add an
  explicit assertion that no `.opencode/ci/` path is ever produced by
  a plain init run.
- [ ] 3.8 [P] In `cmd/gaze/main_test.go`: `--gaze-version` without
  `--ci` is rejected; flag values thread through `runInit` into
  `scaffold.Options` (extend the existing `runInit` test patterns).

## 4. Documentation

- [ ] 4.1 [P] `docs/reference/cli/init.md`: document `--ci` and
  `--gaze-version` (semver `>= v2.3.0`, dev-build fail-closed
  behavior), the generated workflow's shape (self-contained test job,
  report-only thresholds, skip-if-present ownership), and the
  Go-module-at-root requirement with the missing-`go.mod` skip
  behavior.
- [ ] 4.2 [P] `docs/guides/ci-integration.md`: add an install
  section ahead of the consumer example presenting `gaze init --ci`
  as the preferred adoption path; note that the scaffolded workflow
  is report-only until thresholds are uncommented and that the file is
  user-owned after first write.
- [ ] 4.3 [P] `README.md`: mention `--ci` in the init/CI-integration
  areas with a link to the reference page.
- [ ] 4.4 [P] `AGENTS.md`: add a note to the File Sync Invariants
  section that the `internal/scaffold/assets/ci/` subtree is excluded
  from the `.opencode/` mirror sync (no mirror exists; the 9-file
  manifest is unchanged).

## 5. Verification and process gates

- [ ] 5.1 Constitution-alignment verification against the proposal's
  PASS assessments: Accuracy (fail-closed pins, golden-file
  stability), Minimal Assumptions (plain-init byte-parity,
  go.mod-derived version, enforced precondition), Actionable Output
  (next-steps section, TODO markers), Testability (pure functions,
  temp-dir scaffolding, YAML-parse assertions).
- [ ] 5.2 CI parity locally, derived from `.github/workflows/test.yml`
  and `.github/workflows/mega-linter.yml`: `go build ./cmd/gaze`,
  `go test -race -count=1 -short ./...`, `golangci-lint run`, and
  yamllint (repo `.yamllint.yml`) over the new template asset — all
  green.
- [ ] 5.3 `openspec validate init-ci-scaffold --strict` reports no
  findings other than the known, pre-existing `.openspec.yaml`
  `originating_issue: 306` warning — uf-toolchain issue provenance
  that `/uf.finale` reads and the openspec CLI explicitly ignores
  (unknown keys have no effect), but which `--strict` fails on as a
  documented toolchain conflict; any other warning or error is a
  failure. (`openspec validate init-ci-scaffold` without `--strict`
  must report the change as valid.)
- [ ] 5.4 Spec artifacts committed and pushed before implementation
  begins (Spec Commit Gate); implementation proceeds only from the
  pushed baseline.
- [ ] 5.5 Website Documentation Gate: open a docs issue in
  `unbound-force/website` for the new user-facing `--ci` /
  `--gaze-version` flags before the implementing PR merges
  (`gh issue create --repo unbound-force/website --title "docs:
  gaze init --ci scaffolds the AI-report caller workflow" --body
  "..."` covering what changed, why it matters, and which pages need
  updating).
- [ ] 5.6 Review Council Gate: run `/uf.review-council` and resolve
  all REQUEST CHANGES findings until all four reviewers (Adversary,
  Architect, Guard, Tester) return APPROVE; minimal to no code
  changes between APPROVE and PR submission.
