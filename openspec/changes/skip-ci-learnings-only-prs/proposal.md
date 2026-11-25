# Skip CI for Learnings-Only PRs

## Why

Learnings PRs — `chore(dewey): store learnings from …` on branches
`chore/dewey-learnings-*` — only add Markdown files under
`.uf/dewey/learnings/`. gh-hud opens them from the fleet's checkouts,
uf-divisor approves them when every file is a learning
(jflowers/gh-hud#18), and auto-merge lands them. Nothing in them can
break a build, a test, or a lint, yet each one runs the repository's
full CI on every push and on every update from `main`.

Branch protection on `main` requires up-to-date branches. Observed
via the API: classic branch protection with `strict: true` and four
required status checks — `Unit + Integration Tests (Go 1.24)`,
`Unit + Integration Tests (Go 1.25)`, `E2E Tests (Go 1.24)`,
`E2E Tests (Go 1.25)`. The merge train therefore lands learnings PRs
one at a time: update from `main`, wait for full CI (roughly 30
minutes), merge, repeat. On 2026-11-24 gaze had 16 open learnings
PRs, and the queue stalled behind unbound-force/gaze#334, whose
duplicate runs `test.yml`'s concurrency group cancelled; GitHub
counted the cancelled runs as failures, so #334 sat BLOCKED with
green CI and held up every learnings PR behind it. (Originating
issue: #372.)

## What Changes

- `.github/workflows/test.yml` gains a `changes` job that classifies
  each pull request from its net changed-file set — the three-dot
  diff from the pull request's base SHA to its head SHA with rename
  collapsing disabled — and outputs `code=true` when any changed
  file lies outside `.uf/dewey/learnings/` or any changed path is a
  symlink. An empty diff also classifies as `code=true`; a failing
  diff fails the job (fail-closed). Push events classify as
  `code=true` without computing a diff. The job is bounded by
  `timeout-minutes: 5`.
- The `unit-and-integration` and `e2e` matrix jobs gain
  `needs: changes` and `if: needs.changes.outputs.code != 'false'`,
  so they skip only on an explicit learnings-only verdict. Job-level
  gating is safe because the required check becomes the aggregate
  job below, not the matrix leg names.
- A new `ci-ok` job (`CI OK`, needing `changes`,
  `unit-and-integration`, and `e2e`, running even when all of them
  were skipped) prints each needed job's result, fails when any
  needed job failed or was cancelled, and passes when every needed
  job succeeded or was skipped. It becomes the sole required status
  check; the four matrix names leave branch protection.
  `gaze-ai-report` stays advisory and outside `ci-ok`.
- `.github/workflows/mega-linter.yml` gains an equivalent `changes`
  job with a byte-identical classify step, and its `megalinter` job
  gates on the same output. MegaLinter is not a required check, so
  job-level gating is sufficient there.
- A Go contract test, `cmd/gaze/ci_workflow_test.go`, pins the
  wiring: gates, the `ci-ok` needs set, a needed-or-advisory rule
  for every `test.yml` job, and classify-step parity.
- The `gaze-ai-report` job and
  `.github/workflows/reusable_gaze_ai_report.yml` are untouched: the
  job is skipped transitively on learnings-only pull requests (it
  already needs `unit-and-integration`), and it only has a coverage
  artifact to consume when that job ran.

### Out of scope

- The cancelled-runs-count-as-failures behavior that blocked #334;
  it can still affect code PRs. Learnings PRs simply stop running
  the heavy jobs that expose them.
- Changes to `release.yml`, `install-canary.yml`,
  `post-release-install-check.yml`, `fullsend.yaml`,
  `prioritize.yml`, or `reusable_gaze_ai_report.yml`.
- Production Go code changes.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `ci-workflow`: pull requests are classified by net changed files;
  build, test, and lint jobs gate on that classification; an
  aggregate `CI OK` job replaces the matrix leg names as the
  required status check; a contract test pins the wiring.

### Removed Capabilities

- None.

## Impact

- **Internal surfaces**: `.github/workflows/test.yml`,
  `.github/workflows/mega-linter.yml`, new
  `cmd/gaze/ci_workflow_test.go`, `AGENTS.md` (CI/CD section),
  `docs/guides/ci-integration.md` (sweep for invalidated
  statements).
- **Dependent workflow**: `.github/workflows/release.yml`
  preflight (`ci_checks`, line 112) requires
  `Unit + Integration Tests (Go 1.24)`,
  `Unit + Integration Tests (Go 1.25)`, and `MegaLinter` to pass on
  `main` HEAD. Push runs on `main` keep classifying as code, so
  those jobs keep running and reporting under the same names
  (CIW-FR-005); `release.yml` itself is not edited.
- **Repository configuration**: branch protection on
  `unbound-force/gaze` `main` migrates its required status checks
  from the four matrix names to `CI OK`; `strict` up-to-date
  enforcement stays.
- **Behavior matrix**:
  - learnings-only pull request — build, test, lint, and AI-report
    jobs skip; `CI OK` passes within a couple of minutes
  - any pull request that also touches other files, or adds a
    symlink — full CI exactly as today
  - push to `main` — full CI exactly as today
- **Rollout**: protection-first, just-in-time cutover (design D6
  and Rollout runbook). Once `CI OK` has passed on the implementing
  PR, protection switches to `CI OK` and the PR merges immediately,
  so no learnings PR is ever hard-blocked on the old matrix names.
  Other PRs lose nothing: `strict` already makes them update from
  `main`, which runs the new workflow. The previous required
  checks are saved and the runbook includes a rollback command.
- **Trust boundary**: classification runs on a read-only checkout
  with no secrets, but the path prefix is a classification signal,
  not a trust boundary: only `.md` files count, symlinks and
  submodules classify as code, and learnings remain untrusted
  Markdown that Dewey indexes. Skipping MegaLinter also skips its
  gitleaks scan on learnings-only PRs, so a dedicated `gitleaks`
  job scans those PRs' commits and gates `CI OK`.
- **Protected gates**: no coverage thresholds, CRAP limits, or
  severity definitions change; the gaze threshold check step is
  untouched and still runs on every code pull request, and `CI OK`
  needs the job that runs it.
- **Website documentation**: not required — CI/CD pipeline changes
  are exempt per `AGENTS.md`.

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS. The fully agent-driven learnings-PR flow —
gh-hud opens, uf-divisor approves, auto-merge lands — keeps working
without humans babysitting a CI queue. The CI decision is expressed
as a workflow artifact (the `changes` job output and the `CI OK`
aggregate check) that other automation consumes, not as runtime
coupling or human intervention.

### II. Composability First

**Assessment**: PASS. gaze remains standalone: no new third-party
actions (classification is a plain `git diff` on the already-pinned
checkout), each workflow gates independently, and consumers of the
reusable AI-report workflow are untouched. Reverting the change and
running the rollback restores today's behavior with no residue.

### III. Observable Quality

**Assessment**: PASS. The classification verdict and its input —
the changed-file and symlink lists — are visible in the `changes`
job log, so a misclassification can be diagnosed from the run
itself. `CI OK` prints each needed job's result and reports one
machine-consumable aggregate with standard check-run semantics, and
the required-check migration is verifiable through the branch
protection API.

### IV. Testability

**Assessment**: PASS. Classification is a deterministic function of
the merge-base diff, reproducible locally with
`git diff --name-only --no-renames` and `git diff --raw`. A Go
contract test pins the gates, the `ci-ok` needs set, and classify
parity, and the design's Rollout runbook verifies each behavior —
learnings-only PR, mixed PR, update from `main`, push to `main` —
on real pull requests.
