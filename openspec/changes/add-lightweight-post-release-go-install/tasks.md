# Tasks: Post-Release Install Check

<!--
  [P] markers indicate parallelizable tasks:
  a) Tasks touch different files.
  b) No task depends on another's output.
  c) No ordering requirement exists between them.
  Tasks touching the same file must NOT carry [P] relative to each other.
  Each group below has a single task, so no [P] markers apply.
-->

## 1. Workflow

- [x] 1.1 Add `.github/workflows/post-release-install-check.yml` (D1–D7):
  `workflow_run` of the `Release` workflow gated on
  `conclusion == 'success'`, plus `workflow_dispatch` with a required `tag`
  input; resolve the tag (dispatch input, else the most recent non-draft
  release via `gh api repos/$REPO/releases` with a 3-attempt/10s retry);
  run `go install github.com/unbound-force/gaze/v2/cmd/gaze@<tag>` with
  temporary `GOBIN`/`GOMODCACHE` under `RUNNER_TEMP` and a
  3-attempt/30s retry; assert via `go version -m` on `"$GOBIN/gaze"` that
  the module path is `github.com/unbound-force/gaze/v2` and the version is
  exactly the tag (distinct `::error::` diagnostics); run `gaze --version`;
  workflow-level `permissions: {}` with job-level `contents: read`, no
  checkout, pinned `actions/setup-go` SHA with `go-version: "1.25"` and
  `cache: false`, 15-minute timeout, concurrency group
  `post-release-install-check` with `cancel-in-progress: false`.

## 2. Documentation

- [x] 2.1 Update `AGENTS.md`: CI/CD list entry and Recent Changes entry.
  (No website issue: CI/CD pipeline change.)

## 3. Verification and process gates

- [x] 3.1 Constitution alignment per `proposal.md` (all four principles
  PASS; no production code changed).
- [x] 3.2 CI parity locally: `actionlint` and `yamllint -c .yamllint.yml`
  on the new workflow; check the `go version -m` parsing against sample
  build-info output (a matching `mod` line and a mismatching one); dry-run
  the release-list jq resolution read-only against
  `gh api repos/unbound-force/gaze/releases`.
- [ ] 3.3 After merge, dispatch the workflow on `main` with the current
  latest release tag and confirm the install and both build-info
  assertions pass.
- [ ] 3.4 After the next release, confirm the `workflow_run` path fired
  automatically and verified the exact published tag.
