# Tasks: Validate Workflow Floor

<!--
[P] markers indicate parallelizable tasks:
a) Tasks touch different files.
b) No task depends on another's output.
c) No ordering requirement exists between them.
Tasks touching the same file must NOT carry [P] relative to each other.
-->

## 1. Validator script and release workflow wiring

- [x] 1.1 Implement `.github/scripts/validate-workflow-floor.sh` per design
  D1–D6: CLI `-- <release-tag> [repo-root]`, floor extraction from
  `internal/scaffold/scaffold.go` (D2), explicit semver comparison with
  pre-release rules (D3), legacy-tree skip and workflow-presence check (D4),
  above-floor tag existence and contents verification (D5), offline operation
  (D6).
- [x] 1.2 Add the `validate-workflow-floor` job to
  `.github/workflows/release.yml` (sibling of `validate-module-major`: pinned
  checkout SHA with `fetch-depth: 0`, `contents: read`, 5-minute timeout,
  `RELEASE_TAG` env) and change `preflight` to
  `needs: [validate-module-major, validate-workflow-floor]` (D1, D6).
- [x] 1.3 Update the `.github/workflows/release.yml` header comment to
  describe the workflow-floor validation.

## 2. CI test script and wiring

- [x] 2.1 Implement `.github/scripts/test-validate-workflow-floor.sh` per
  design D7–D8: repository-invariant assertions (floor constant present and
  parseable, reusable workflow file present in tree) followed by the
  synthetic-repo matrix covering all fourteen cases from D8, using the
  `set +e` / capture / `set -e` pattern and `mktemp -d` with `trap` cleanup.
- [x] 2.2 Wire the test script into `.github/workflows/test.yml` on the
  Go 1.25 matrix leg beside the module-major test step
  (`Test release workflow-floor validation`).

## 3. Documentation

- [x] 3.1 Update the `AGENTS.md` CI/CD section's release.yml description to
  mention workflow-floor validation. (No website issue: CI/CD pipeline changes
  are exempt from the Website Documentation Gate.)

## 4. Verification and process gates

- [x] 4.1 Constitution alignment: confirm the four constitution principles
  (Accuracy, Minimal Assumptions, Actionable Output, Testability) each hold
  for the implementation, per the assessments in `proposal.md`.
- [x] 4.2 CI parity locally, deriving commands from `.github/workflows/`:
  run both release-validation test scripts, `yamllint` on the modified
  workflow files, `go build ./...`, `go test -race -count=1 -short ./...`,
  and `golangci-lint run`.
- [x] 4.3 `openspec validate validate-workflow-floor --strict` passes
  (tolerating only the known pre-existing `.openspec.yaml` warning); the
  non-strict validation reports the change as valid.
- [x] 4.4 Spec Commit Gate: commit and push all spec artifacts before any
  implementation commit lands.
- [x] 4.5 Review Council Gate: run `/uf.review-council` and resolve all
  findings until all four reviewers APPROVE; make minimal to no changes
  between APPROVE and PR submission.
