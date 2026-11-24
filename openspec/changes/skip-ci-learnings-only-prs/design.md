## Context

`test.yml` runs on every push to `main` and every pull request:
`unit-and-integration` (matrix Go 1.24/1.25 — build, unit and
integration tests with coverage, release-validation scripts, the
gaze threshold check, coverage upload), `e2e` (same matrix,
30-minute self-check tests), and `gaze-ai-report` (a
reusable-workflow call that needs `unit-and-integration` and
consumes its `coverage-out` artifact). `mega-linter.yml` runs
MegaLinter with auto-fix commits on pull requests. Branch
protection on `main` — observed via the API — is classic
protection with `strict: true` (branches must be up to date) and
four required status checks: the matrix leg names `Unit +
Integration Tests (Go 1.24)`, `Unit + Integration Tests (Go 1.25)`,
`E2E Tests (Go 1.24)`, `E2E Tests (Go 1.25)`. MegaLinter and Gaze
AI Report are not required checks.

Learnings PRs only add Markdown under `.uf/dewey/learnings/` and
are opened, approved, and merged by automation (gh-hud,
uf-divisor, auto-merge). With `strict: true`, each must be updated
from `main` and re-run full CI before it can merge.

Constraints discovered during inspection:

- C1: workflow-level `paths-ignore` cannot gate a workflow that
  provides a required check — the check never reports and the PR
  waits on "Expected" forever.
- C2: a matrix job skipped by a job-level `if:` reports once with
  the matrix expression unexpanded, so the per-leg required names
  (`… (Go 1.24)`) never report.
- C3: skipping a reusable-workflow caller reports only the
  caller's job name, not the nested checks inside it.
- C4: classification must survive update-from-main merges: the
  merge commit itself touches code files, so filtering on the
  event's pushed files misclassifies; only the pull request's net
  diff against its merge base stays learnings-only.
- C5: repo convention pins all actions by SHA; a new third-party
  action (for example `dorny/paths-filter`) adds supply-chain
  surface for work a plain `git diff` already does.
- C6: no Go test currently asserts on `test.yml` or
  `mega-linter.yml` (verified by grep; `release_workflow_test.go`
  guards only the release caller). This change adds one (D8).
- C7: `release.yml` preflight (`ci_checks`, line 112) requires the
  check names `Unit + Integration Tests (Go 1.24)`,
  `Unit + Integration Tests (Go 1.25)`, and `MegaLinter` to pass on
  `main` HEAD, so push runs on `main` must keep reporting them.

## Goals / Non-Goals

### Goals

- Learnings-only PRs show all required checks passing within a
  couple of minutes, without running build, tests, or linters.
- Mixed or code-only PRs run full CI exactly as today.
- Push runs on `main` are unchanged.
- Misclassification fails closed — full CI or a blocked PR — and
  never a silent skip.
- No learnings PR is ever hard-blocked by the rollout itself.

### Non-Goals

- Fixing the cancelled-runs-count-as-failures behavior that
  blocked #334 (still possible on code PRs).
- Touching `release.yml`, `install-canary.yml`,
  `post-release-install-check.yml`, `fullsend.yaml`,
  `prioritize.yml`, or `reusable_gaze_ai_report.yml`.
- Production Go code changes (one workflow contract test is
  added, D8).

## Decisions

### D1: classify with a `changes` job running plain git diff

A new `changes` job in `test.yml` (repeated in `mega-linter.yml`,
D5) declares `timeout-minutes: 5`, checks out with
`fetch-depth: 0`, and classifies in one step:

```yaml
- name: Classify change
  id: classify
  run: |
    set -euo pipefail
    if [ "${{ github.event_name }}" != "pull_request" ]; then
      echo "code=true" >> "$GITHUB_OUTPUT"
      exit 0
    fi
    base="${{ github.event.pull_request.base.sha }}"
    head="${{ github.event.pull_request.head.sha }}"
    files="$(git diff --name-only --no-renames "${base}...${head}")"
    raw="$(git diff --raw --no-renames "${base}...${head}")"
    outside="$(printf '%s\n' "$files" \
      | grep -v '^\.uf/dewey/learnings/' || true)"
    links="$(printf '%s\n' "$raw" \
      | awk '$1 ~ /120000/ || $2 ~ /120000/ { print $NF }')"
    if [ -z "$files" ] || [ -n "$outside" ] || [ -n "$links" ]; then
      echo "code=true" >> "$GITHUB_OUTPUT"
      echo "files outside .uf/dewey/learnings/:"
      echo "$outside"
      echo "symlinks:"
      echo "$links"
    else
      echo "code=false" >> "$GITHUB_OUTPUT"
      echo "learnings-only pull request"
    fi
```

- The three-dot diff (`base...head`) compares from the merge
  base, so a PR updated from `main` is classified by its net
  diff (C4).
- `--no-renames` lists both sides of a rename, so a code file
  renamed into `.uf/dewey/learnings/` classifies as code.
- Any changed path whose old or new mode is `120000` (a symlink)
  classifies as code, even under `.uf/dewey/learnings/`: a
  symlink there can point at arbitrary repository content, so
  the path prefix alone is not trusted for it.
- An empty diff yields `code=true`; `set -euo pipefail` turns a
  failing diff into a failed job (fail-closed). Both diffs are
  captured into variables before filtering, so a failing `git`
  is never masked by a `|| true` on a pipeline.
- Push events short-circuit to `code=true` (D7).
- The captured-variable form avoids `grep -q`: under
  `set -o pipefail`, `grep -q`'s early exit can SIGPIPE the
  producer and flip the pipeline status. It also logs the files
  and symlinks that triggered `code=true`, keeping the verdict
  diagnosable from the run (Observable Quality).
- `timeout-minutes: 5` bounds a hung checkout or diff instead of
  the 360-minute default.
- Rejected: `dorny/paths-filter` (C5) and native `paths-ignore`
  (C1 for test.yml; C4 makes event-path filtering wrong for
  merge-train updates in either workflow).

### D2: gate the matrix jobs at the job level, failing safe

`unit-and-integration` and `e2e` get `needs: changes` and
`if: needs.changes.outputs.code != 'false'`. The jobs skip only on
an explicit `false`; an empty, missing, or malformed output runs
full CI. Because the expression has no status-check function,
GitHub applies an implicit `success()`: when the `changes` job
itself fails, the gated jobs are skipped and `CI OK` fails on the
`changes` failure (D3), so a broken classifier blocks the PR
rather than passing it.

Job-level gating is normally unsafe for matrix jobs (C2), but D3
moves the required check off the leg names, making the
unexpanded-name reporting moot. The alternative — gating steps so
each leg still checks out and exits fast — was rejected because
every leg still pays checkout and setup-go time and the required
checks stay coupled to the matrix leg names.

### D3: one aggregate `ci-ok` job as the only required check

```yaml
ci-ok:
  name: CI OK
  needs: [changes, unit-and-integration, e2e]
  if: ${{ !cancelled() }}
  runs-on: ubuntu-latest
  timeout-minutes: 5
  steps:
    - name: Aggregate CI result
      env:
        NEEDS: ${{ toJSON(needs) }}
      run: |
        set -euo pipefail
        printf '%s\n' "$NEEDS" \
          | jq -r 'to_entries[] | "\(.key): \(.value.result)"'
        printf '%s\n' "$NEEDS" | jq -e \
          'to_entries | all(.value.result == "success"
                             or .value.result == "skipped")' \
          > /dev/null
```

The first `jq` prints one `job: result` line per needed job, so a
red `CI OK` names the job that caused it without opening the run
graph. `jq -e` then exits non-zero when the predicate is false, so
any `failure` or `cancelled` need fails the job; `success` and
`skipped` pass. Passing `needs` through an environment variable
keeps the JSON out of the shell script text. This resolves C2 and
C3 (leg names and nested reusable checks stop being required) and
makes future job-list changes free of protection edits.

`gaze-ai-report` is deliberately not in `needs`. It is advisory
today (not a required check), depends on an external AI provider
and secret, and fails on provider outages or a rejected narrative;
requiring it through `CI OK` would make it merge-blocking. The
quality thresholds it reports are already enforced by the gaze
threshold check step inside `unit-and-integration`, which `CI OK`
does need.

Required-needs rule: every `test.yml` job whose failure must block
a merge MUST be listed in `ci-ok.needs`. Every other job MUST be
named in the contract test's advisory allowlist (today:
`gaze-ai-report`). The D8 contract test fails when a job is in
neither set, so adding a job to `test.yml` forces an explicit
blocking-or-advisory decision instead of silently escaping the
gate.

`!cancelled()` refines the originating issue's `if: always()`
sketch: `always()` also runs when the workflow run itself was
cancelled (for example by the concurrency group), converting a
cancelled run into a failed `CI OK`; `!cancelled()` keeps
cancelled-as-cancelled while still running when needs were
skipped or failed. Required checks evaluate the latest run on
each head SHA, and the superseding run re-reports there.

### D4: `gaze-ai-report` stays untouched

It already declares `needs: unit-and-integration`; a job whose
needed job was skipped is itself skipped, so it skips on
learnings-only PRs with no edit — and it has no coverage artifact
to consume there anyway. A brief comment in `test.yml` records the
transitive skip and that the job is advisory (outside `ci-ok`).

This is why the delta spec modifies no existing requirement: the
existing `ci-workflow` requirements from the
`reusable-gaze-ai-report` change govern the reusable AI-report
contract and push-to-main behavior, and the "AI report MUST run
only where secrets are available" requirement is an upper bound
on where the report runs, not violated by skipping it on
learnings-only pull requests. CIW-FR-005 preserves push-to-main
behavior.

### D5: `mega-linter.yml` gets its own `changes` job

Workflows cannot share jobs. The same `changes` job runs there
with a byte-identical classify step (contract pinned by
CIW-FR-004 and asserted by D8) and `timeout-minutes: 5`, and the
`megalinter` job gates with
`if: needs.changes.outputs.code != 'false'` — acceptable because
MegaLinter is not a required check. The auto-fix commit steps live
inside the gated job, so learnings-only PRs also stop receiving
auto-fix pushes.

A local composite action under `.github/actions/` (no SHA pin
needed) would remove the duplication; it was rejected for now to
keep both workflows self-contained, with D8's byte-identity
assertion as the drift guard instead.

### D6: protection-first, just-in-time cutover in one PR

The rollout must never leave a window where `main` gates the
matrix jobs while protection still requires the matrix leg names:
skipped legs never report those names (C2), which would hard-block
every learnings PR until a manual protection edit. The cutover
therefore switches protection immediately before merging the
implementing PR, following the Rollout runbook below:

1. Readiness: the implementing PR is a code PR, so every job runs
   ungated on it. Confirm `CI OK` and all four matrix legs passed
   on its head SHA.
2. Save the current required checks, then make `CI OK` the only
   required status check (`strict` unchanged) and read it back.
3. Merge the implementing PR immediately. It already carries a
   passing `CI OK` on its up-to-date head SHA.
4. If the merge does not happen promptly, roll back by restoring
   the saved four names.

Other open PRs lose nothing: `strict: true` already forces each
of them to update from `main` after the implementing PR merges,
and that update runs the new workflow, which reports `CI OK`.
Between steps 2 and 3 no other PR can merge, a window bounded to
minutes. After merge, protection already lists `CI OK`, so no
learnings PR ever waits on the old names. Rejected: migrating
after merge (learnings PRs hard-blocked until the admin edit) and
a two-PR rollout (ungated `ci-ok` first, gating second), which
doubles the review and merge cost for the same ordering guarantee.

### D7: push-to-main behavior is preserved by construction

The classify step short-circuits to `code=true` on push events,
so every gated job runs exactly as today (CIW-FR-005), and the
check names `release.yml` preflight requires on `main` HEAD keep
reporting (C7). No workflow trigger, concurrency, or permission
changes are made.

### D8: a Go contract test pins the workflow wiring

`cmd/gaze/ci_workflow_test.go` (package `main`, beside
`release_workflow_test.go`, standard library `testing` plus the
existing `gopkg.in/yaml.v3` dependency) parses `test.yml` and
`mega-linter.yml` and asserts:

- each workflow has a `changes` job with output `code`,
  `permissions: contents: read`, and `timeout-minutes: 5`;
- `unit-and-integration`, `e2e`, and `megalinter` need `changes`
  and carry exactly `needs.changes.outputs.code != 'false'`;
- `ci-ok` is named `CI OK`, runs on `${{ !cancelled() }}`, has
  `timeout-minutes: 5`, and its `needs` set equals
  `{changes, unit-and-integration, e2e}`;
- every `test.yml` job other than `ci-ok` is in `ci-ok.needs` or
  in the advisory allowlist `{gaze-ai-report}` (drift guard, D3);
- the classify step's `run` string is byte-identical in both
  workflows (D5).

## Rollout runbook

Admin token required for the protection calls. Repository:
`unbound-force/gaze`; branch: `main`.

### Stage 1: readiness on the implementing PR

- Record in the PR description that no website documentation
  issue is required (CI/CD pipeline changes are exempt per
  `AGENTS.md`).
- `gh pr checks <PR>` — confirm `Changes`, all four matrix legs,
  `MegaLinter`, `Gaze AI Report`, and `CI OK` ran, and that the
  four legs and `CI OK` passed on the head SHA
  (`gh pr view <PR> --json headRefOid,statusCheckRollup`).
- Confirm the PR is up to date with `main`.

### Stage 2: switch required checks

```bash
gh api repos/unbound-force/gaze/branches/main/protection/required_status_checks \
  > required-checks-before.json
printf '%s' '{"strict": true, "checks": [{"context": "CI OK"}]}' \
  | gh api -X PATCH \
      repos/unbound-force/gaze/branches/main/protection/required_status_checks \
      --input -
gh api repos/unbound-force/gaze/branches/main/protection/required_status_checks \
  --jq '{strict: .strict, contexts: .contexts}'
```

The readback MUST show `strict: true` and `contexts: ["CI OK"]`.

### Stage 3: merge

Merge the implementing PR immediately (`gh pr merge <PR>` with the
repository's usual strategy). If it cannot merge within minutes,
run the rollback.

### Rollback

```bash
printf '%s' '{"strict": true, "checks": [
  {"context": "Unit + Integration Tests (Go 1.24)"},
  {"context": "Unit + Integration Tests (Go 1.25)"},
  {"context": "E2E Tests (Go 1.24)"},
  {"context": "E2E Tests (Go 1.25)"}]}' \
  | gh api -X PATCH \
      repos/unbound-force/gaze/branches/main/protection/required_status_checks \
      --input -
```

If the change already merged and must be undone, revert the PR
first, then run the rollback, and read back as in Stage 2.

### Stage 4: post-merge verification

1. Open a throwaway learnings-only PR (one new file under
   `.uf/dewey/learnings/`); verify the matrix, e2e, megalinter,
   and gaze-ai-report jobs are skipped and `CI OK` passes within
   a couple of minutes (CIW-FR-001/002/003 scenarios).
2. Push a second commit touching a file outside
   `.uf/dewey/learnings/`; verify full CI runs as today
   (CIW-FR-002 code-PR scenario).
3. Revert that commit and merge `main` into the branch; verify
   the jobs stay skipped (CIW-FR-004 scenario).
4. Observe the next push to `main`; verify full CI runs unchanged
   and the `release.yml` preflight check names report on `main`
   HEAD (CIW-FR-005 scenario).
5. Close the throwaway PR and delete its branch.

## Risks / Trade-offs

- **Cutover window** (D6): between switching protection and
  merging, no other PR can merge. Bounded to minutes; the saved
  `required-checks-before.json` and the rollback command restore
  the previous state.
- **Branch-protection state lives outside the repository.** The
  required-checks switch to `CI OK` is a GitHub settings change no
  in-repo contract test can pin; the Rollout runbook's readback
  (Stage 2) and post-merge verification (Stage 4) are the check,
  and the saved `required-checks-before.json` plus the rollback
  command are the recovery path.
- **Classification logic duplicated** across two workflows:
  small, contract-pinned by CIW-FR-001/004 and byte-identity
  asserted by D8.
- **Learnings content is untrusted input.** Learnings are Markdown
  later indexed by Dewey and read by agents; skipping CI does not
  change how that content is consumed, but the path prefix is only
  a classification signal, not a trust boundary. Symlinks under
  the prefix classify as code (D1). Skipping MegaLinter on
  learnings-only PRs also skips its gitleaks secret scan there;
  the push-to-main MegaLinter run still scans every merged
  learning, and uf-divisor still reviews each learnings PR.
- **Cancelled-run accounting** (#334) still affects code PRs —
  out of scope.
- **`fetch-depth: 0`** on the changes job fetches full history —
  seconds at this repository's size; `timeout-minutes: 5` bounds
  it.
- **Empty-diff PRs** classify as code and run full CI —
  conservative, and cannot hang.

Constitution alignment is assessed in the proposal; the relevant
decision notes: the CI verdict remains a workflow artifact other
automation consumes, keeping the learnings-PR flow fully
automated (Autonomous Collaboration); no new third-party actions
or mandatory couplings are introduced (Composability First); the
classification log, the per-job `CI OK` table, and the single
aggregate keep every decision observable with provenance
(Observable Quality); and the D8 contract test pins the wiring
(Testability).
