# CI Workflow Requirements

<!--
Requirement numbering allocation:
- Allocated to this change: CIW-FR-001 .. CIW-FR-006
- Reserved for future ci-workflow changes: CIW-FR-007 and above
- Sibling capabilities' requirements live in their own delta
  directories (openspec/changes/<change>/specs/<capability>/) and,
  once synced, under openspec/specs/<capability>/spec.md.
- Earlier ci-workflow deltas predate FR numbering; their
  requirements are identified by name only and are not renumbered
  by this change.
-->

## ADDED Requirements

### Requirement: PR classification by net changed files

CIW-FR-001

`.github/workflows/test.yml` MUST include a `changes` job, bounded
by `timeout-minutes: 5`, that outputs a single `code`
classification:

- For `pull_request` events, the job MUST compute the pull
  request's net changed-file set as the three-dot diff from the
  pull request's base SHA to its head SHA with rename collapsing
  disabled, and MUST output `code=true` when any changed file
  is not a `.md` file under `.uf/dewey/learnings/`.
- The job MUST output `code=true` when any changed path has mode
  `120000` (a symlink) or `160000` (a submodule) on either side
  of the diff, including paths under `.uf/dewey/learnings/`.
- The job MUST output `code=false` only when the diff is non-empty,
  every changed file is a `.md` file under
  `.uf/dewey/learnings/`, and no changed path is a symlink or
  submodule.
- The job MUST output `code=true` when the diff is empty
  (fail-closed default).
- For `push` events, the job MUST output `code=true` without
  computing a diff.
- A failing diff command MUST fail the job.
- The job log MUST list the files, symlinks and submodules that
  produced a `code=true` verdict.

#### Scenario: Learnings-only pull request

- **GIVEN** a pull request whose net diff contains only regular
  `.md` files under `.uf/dewey/learnings/`
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=false`

#### Scenario: Mixed or code-only pull request

- **GIVEN** a pull request whose net diff contains at least one
  file outside `.uf/dewey/learnings/`
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=true`

#### Scenario: Code file renamed into the learnings path

- **GIVEN** a pull request renames `cmd/gaze/main.go` to
  `.uf/dewey/learnings/notes.md`
- **WHEN** the `changes` job runs with rename collapsing disabled
- **THEN** the changed-file list contains both the old and new
  paths, so the job outputs `code=true`

#### Scenario: Non-Markdown file under the learnings path

- **GIVEN** a pull request that adds `.uf/dewey/learnings/notes.txt`
  and no other files
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=true` and logs the path

#### Scenario: Submodule under the learnings path

- **GIVEN** a pull request that adds a submodule
  `.uf/dewey/learnings/sub.md` and no other files
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=true` and logs the path

#### Scenario: Symlink under the learnings path

- **GIVEN** a pull request that adds a symlink
  `.uf/dewey/learnings/link.md` and no other files
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=true` and logs the symlink path

#### Scenario: Push event defaults to code

- **GIVEN** a push to `main`
- **WHEN** the `changes` job runs
- **THEN** it outputs `code=true` without computing a diff

#### Scenario: Diff failure fails closed

- **GIVEN** the diff command fails (for example a base SHA missing
  from the fetched history)
- **WHEN** the classification step runs
- **THEN** the `changes` job fails

### Requirement: Build and test jobs gate on classification

CIW-FR-002

The `unit-and-integration` and `e2e` jobs in
`.github/workflows/test.yml` MUST declare `needs: changes` and MUST
be skipped only when `needs.changes.outputs.code` is exactly
`'false'` (`if: needs.changes.outputs.code != 'false'`); any other
output value, including an empty one, MUST run them. When they
execute, their steps, matrix legs, thresholds, and timeouts MUST
be identical to the pre-change workflow. The `gaze-ai-report` job
MUST be skipped on learnings-only pull requests — provided
transitively by its existing `needs: unit-and-integration`, since
a job whose needed job was skipped is itself skipped — and MUST
otherwise remain unchanged.

#### Scenario: Learnings-only pull request skips build and test

- **GIVEN** a pull request classified `code=false`
- **WHEN** the Test workflow runs
- **THEN** `unit-and-integration`, `e2e`, and `gaze-ai-report` are
  all skipped

#### Scenario: Code pull request runs full CI

- **GIVEN** a pull request classified `code=true`
- **WHEN** the Test workflow runs
- **THEN** both matrix jobs run with the same steps, thresholds,
  and matrix legs as before this change
- **AND** `gaze-ai-report` runs as before, consuming the coverage
  artifact

#### Scenario: Missing classification runs full CI

- **GIVEN** a `changes` job that succeeded but produced an empty
  or non-`false` `code` output
- **WHEN** the Test workflow evaluates the gated jobs
- **THEN** `unit-and-integration` and `e2e` run

### Requirement: Aggregate CI OK is the sole required check

CIW-FR-003

`.github/workflows/test.yml` MUST provide a `ci-ok` job named
`CI OK`, bounded by `timeout-minutes: 5`, whose `needs` are exactly
`changes`, `unit-and-integration`, and `e2e`. It MUST NOT need
`gaze-ai-report`, which stays advisory. The job MUST run even when
every needed job was skipped, and:

- MUST fail when any needed job concluded `failure` or `cancelled`
- MUST pass when every needed job concluded `success` or `skipped`
- MUST log each needed job's name and result before deciding

Every `test.yml` job whose failure must block a merge MUST be
listed in `ci-ok.needs`; every other `test.yml` job MUST be
explicitly designated advisory in the CIW-FR-006 contract test.

Branch protection on `main` MUST require `CI OK` and MUST NOT
require the four matrix leg check names.

#### Scenario: All gated jobs skipped

- **GIVEN** a learnings-only pull request where the classification
  succeeded and every gated job was skipped
- **WHEN** `ci-ok` runs
- **THEN** it passes

#### Scenario: A needed job failed or was cancelled

- **GIVEN** a pull request where any job `ci-ok` needs concluded
  `failure` or `cancelled`
- **WHEN** `ci-ok` runs
- **THEN** it fails
- **AND** its log names the job and its result

#### Scenario: Classification failure blocks the PR

- **GIVEN** a pull request whose `changes` job failed
- **WHEN** `ci-ok` runs
- **THEN** it fails

#### Scenario: AI report failure does not block merge

- **GIVEN** a code pull request where `unit-and-integration` and
  `e2e` succeeded and `gaze-ai-report` failed
- **WHEN** `ci-ok` runs
- **THEN** it passes

#### Scenario: Learnings-only PR is mergeable quickly

- **GIVEN** a pull request whose changed files are all under
  `.uf/dewey/learnings/`, after branch protection requires only
  `CI OK`
- **WHEN** the Test workflow completes
- **THEN** the pull request shows every required check passing
  within a couple of minutes, without the build, test, or lint
  steps having run

### Requirement: MegaLinter skips learnings-only PRs

CIW-FR-004

`.github/workflows/mega-linter.yml` MUST include a `changes` job,
bounded by `timeout-minutes: 5`, whose classify step is
byte-identical to the one in `.github/workflows/test.yml`
(CIW-FR-001 contract), and its `megalinter` job MUST declare
`needs: changes` and be skipped only when
`needs.changes.outputs.code` is exactly `'false'`. The
classification MUST be computed from the pull request's net diff
against its merge base — not from event path filtering — so that
merging `main` into a learnings-only pull request does not
re-trigger the lint job.

#### Scenario: Learnings PR updated from main stays skipped

- **GIVEN** a learnings-only pull request that merges `main` into
  its branch to satisfy the up-to-date requirement
- **WHEN** the MegaLinter workflow runs on the synchronize event
- **THEN** the net diff against the merge base still contains only
  files under `.uf/dewey/learnings/`
- **AND** the `megalinter` job is skipped

#### Scenario: Code PR runs MegaLinter

- **GIVEN** a pull request classified `code=true`
- **WHEN** the MegaLinter workflow runs
- **THEN** the `megalinter` job runs with its existing steps,
  permissions, and auto-fix behavior unchanged

### Requirement: Push runs on main are unchanged

CIW-FR-005

For `push` events on `main`, every Test and MegaLinter job MUST
execute exactly as before this change: the classification outputs
`code=true`, all gated jobs run, and the gaze AI report dogfoods
the pushed commit. This preserves the check names
`.github/workflows/release.yml` preflight requires on `main` HEAD
(`Unit + Integration Tests (Go 1.24)`,
`Unit + Integration Tests (Go 1.25)`, `MegaLinter`).

#### Scenario: Push to main runs full CI

- **GIVEN** a push to `main`
- **WHEN** the Test and MegaLinter workflows run
- **THEN** `unit-and-integration` (both legs), `e2e` (both legs),
  `gaze-ai-report`, and `megalinter` all execute with their
  pre-change steps and semantics
- **AND** the check names required by `release.yml` preflight
  report on the pushed commit

### Requirement: Workflow wiring contract test

CIW-FR-006

A Go test in `cmd/gaze/` MUST parse `.github/workflows/test.yml`
and `.github/workflows/mega-linter.yml` and fail when any of the
following no longer holds: the `changes` jobs and their `code`
output, `permissions: contents: read`, and `timeout-minutes: 5`;
the `!= 'false'` gates on `unit-and-integration`, `e2e`, and
`megalinter`; the `ci-ok` name, `!cancelled()` condition,
timeout, and exact `needs` set; every other `test.yml` job being
either needed by `ci-ok` or on the test's advisory allowlist
(`gaze-ai-report`); and byte-identical classify steps across the
two workflows.

#### Scenario: New job added without a gate decision

- **GIVEN** a new job added to `test.yml` that is neither in
  `ci-ok.needs` nor on the advisory allowlist
- **WHEN** `go test ./cmd/gaze/` runs
- **THEN** the contract test fails naming the job

#### Scenario: Classify steps drift

- **GIVEN** the classify step in `mega-linter.yml` differs from
  the one in `test.yml`
- **WHEN** `go test ./cmd/gaze/` runs
- **THEN** the contract test fails

## MODIFIED Requirements

None.

## REMOVED Requirements

None.

### Requirement: Secret scan on learnings-only pull requests

`.github/workflows/test.yml` MUST include a `gitleaks` job that
runs only when `changes` outputs `code=false`, scans the pull
request's commits (`base..head`) with a gitleaks binary whose
download is verified against a pinned SHA-256, and is listed in
`ci-ok`'s `needs`.

#### Scenario: Secret in a learning

- **GIVEN** a learnings-only pull request whose commits contain a
  string matching a gitleaks rule
- **WHEN** the `gitleaks` job runs
- **THEN** it fails and `CI OK` fails

#### Scenario: Code pull request

- **GIVEN** a pull request with `code=true`
- **WHEN** CI runs
- **THEN** the `gitleaks` job is skipped and MegaLinter's gitleaks
  scan covers the pull request as today
