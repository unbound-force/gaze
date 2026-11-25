# Design: Post-Release Install Check

## Context

Every release gate in `.github/workflows/release.yml` verifies repository
state before publication: `validate-module-major` (tag major vs. the
`go.mod` module path), `validate-workflow-floor`,
`validate-ai-report-default`, and the org-infra preflight (tag format,
uniqueness, semver ordering, CI status, unreleased commits). None of them
can see what the public Go module proxy serves. The README documents that
earlier v2 tags were published with an incompatible module path and cannot
be installed from the `/v2` module path — every one of those gates passed.
Because the proxy content-caches a served version permanently, such a tag is
permanently uninstallable and needs a new tag to remediate.

The install canary (`install-canary.yml`) watches this surface, but on a
daily schedule against whatever release GitHub marks Latest: up to a day of
detection latency for a new publication, prereleases never checked, and a
drift-detection (not publication-verification) purpose. Issue #297 asks for
the missing layer: a lightweight smoke test of the exact published tag,
immediately after publication.

## Goals / Non-Goals

### Goals

- Verify that the exact tag a successful Release run just published installs
  from proxy.golang.org, automatically, without waiting for the canary.
- Verify any published tag on demand (`workflow_dispatch`), including
  prereleases and older tags.
- Keep pre-merge CI hermetic: no `push`/`pull_request` triggers, no network
  calls outside the post-release/dispatch contexts.
- Follow the canary's operational patterns: bounded inline retries,
  `::error::` diagnostics with expected vs. actual values, pinned action
  SHAs, `cache: false`, workflow-level `permissions: {}` with job-level
  `contents: read`, and a CI-012 concurrency group.

### Non-Goals

- Drift detection between releases (the canary's job; unchanged).
- Any change to `release.yml`, signing, or Homebrew behavior.
- Proving a single-shot consumer install succeeds: like the canary (its
  design D5), the check retries, so a pass proves eventual installability.
- Notification routing beyond GitHub's default failure email.

## Decisions

### D1: A dedicated workflow, not an extension of the canary or a release job

Alternatives considered:

- Extending `install-canary.yml` with a tag-input mode — rejected. Triggers
  are workflow-level (`on:` applies to every job), so mixing the canary's
  `schedule` + input-less `workflow_dispatch` with `workflow_run` +
  dispatch-with-`tag` forces `if:` guards onto every job and tangles two
  concurrency models: the canary's static group exists so a dispatch may
  cancel a nightly run, while an exact-tag verification should never be
  cancelled by an unrelated canary run. The two workflows also answer
  different questions (between-release drift vs. publication of a specific
  tag) and fail for different reasons.
- A final job inside `release.yml` — rejected. It would modify the release
  workflow (issue non-goal), inherit its write-permission context, and
  colour the release run itself red when the *publication* succeeded but
  the proxy lags — conflating "release failed" with "release not yet
  consumable".

A new `.github/workflows/post-release-install-check.yml` keeps the release
pipeline untouched, gives the check its own failure signal, and mirrors the
canary's file structure closely enough to review at a glance.

### D2: Automatic trigger via `workflow_run` of `Release`, gated on success

`on: release: published` cannot work: GoReleaser publishes releases with
`GITHUB_TOKEN`, and events created with `GITHUB_TOKEN` do not start
workflow runs (the reason the canary uses a schedule). The Release workflow
itself, however, is human-dispatched, and the completion of a workflow run
emits a `workflow_run` event regardless of the token its steps used — the
canary's design D1 already evaluated this trigger as "fires reliably" and
rejected it only because a once-per-release trigger cannot see
between-release drift. For publication verification, once-per-release is
exactly the desired cadence.

The job runs when the Release run concluded successfully:

```yaml
if: >-
  github.event_name == 'workflow_dispatch'
  || (github.event_name == 'workflow_run'
      && github.event.workflow_run.conclusion == 'success')
```

Gating on `conclusion == 'success'` avoids the canary's noted drawback of
running on failed or cancelled releases. Trade-off: a release *published*
by a run that later fails (e.g., at macOS signing) gets no automatic check;
the canary covers it within a day and `workflow_dispatch` closes it
immediately. Accepted.

### D3: Tag resolution — dispatch input, or the most recent non-draft release

- `workflow_dispatch` path: the required `tag` input is the tag under
  verification. A cheap guard rejects an empty or non-`v`-prefixed input;
  `go install` is the authoritative validator of everything else.
- `workflow_run` path: the event payload does not carry the Release run's
  dispatch inputs, so the tag is resolved as the most recent non-draft
  release:
  `gh api repos/$REPO/releases --jq '[.[] | select(.draft == false)][0].tag_name'`
  with a bounded retry (3 attempts, 10s apart) for REST transients.

The canary's resolver (`gh release view`) was deliberately not reused: it
returns the release GitHub marks Latest — non-draft, non-prerelease (canary
design D2) — while the Release preflight runs with `allow_prerelease:
true`. After a prerelease publication, `gh release view` would silently
resolve the previous stable tag and verify the wrong release. The
releases-list query includes prereleases and excludes drafts, and right
after a successful Release run the most recent non-draft release is the one
that run published (the preflight's tag-uniqueness check precludes a
conflicting publication).

Re-running an old successful Release run fires the trigger again; the check
then resolves and verifies the most recent release rather than the re-run's
tag. Accepted: the verified statement ("the most recent published release
is proxy-installable at its tag") remains true, and the run is idempotent
and read-only.

### D4: Temporary `GOBIN`/`GOMODCACHE`, verified by absolute path

The job sets `GOBIN` and `GOMODCACHE` to fresh directories under
`$RUNNER_TEMP`, so no shared or pre-populated cache can satisfy module
resolution and the verified binary is exactly the one this run installed.
Verification addresses `"$GOBIN/gaze"` by absolute path, which removes the
canary's `command -v`/PATH guard class entirely (no dependence on
`GOPATH/bin` being on `PATH`). `actions/setup-go` keeps `cache: false` (no
`go.sum` to key on, canary D4); a fresh runner has an empty build cache
anyway, so `GOCACHE` isolation would add nothing. This satisfies the
issue's "temporary GOBIN and module caches" requirement and is a
determinism guarantee: the pass means the public proxy path was exercised.

### D5: Assert the served module and exact version from build info

`go install` does not set the GoReleaser ldflags, so `gaze --version`
prints `dev` and cannot prove what was built (canary D3). The check parses
`go version -m "$GOBIN/gaze"` and makes two separate assertions with
distinct `::error::` diagnostics:

- the `mod` line's module path equals `github.com/unbound-force/gaze/v2`
  (a mismatch is the documented historical failure class and means the
  release is permanently uninstallable — remediation is a new tag);
- the `mod` line's version equals the tag under verification (a mismatch
  means the proxy served different content than requested).

`gaze --version` still runs, as in the canary and the reusable workflow,
to prove the binary executes on the runner. The local parsing and
comparisons are deterministic and are not retried.

### D6: Bounded retries on network steps only; 15-minute timeout

Same shape as canary D5: 3 attempts with 10s backoff for the `gh api`
resolve step, 3 attempts with 30s backoff for `go install` — where
just-published-tag propagation lag is the *expected* case for this
workflow, not the edge case it is for the canary. The final attempt stays
fatal with the existing `::error::` diagnostics, preserving the spec's
MUST-fail contract. A retry `uses:` action was rejected: it would add a
SHA-pinning dependency for behavior three lines of bash provide. The job
timeout is 15 minutes, covering the worst-case retry path.

### D7: Concurrency — queue, never cancel

A static CI-012 group `post-release-install-check` with
`cancel-in-progress: false`. Every run is a distinct verification signal
worth keeping (an automatic run for a new tag, a maintainer dispatch for an
old one); runs are short (bounded by the 15-minute timeout), so back-to-
back releases queue rather than race. This differs from the canary, where a
dispatch intentionally cancels a superseded nightly run.

### Toolchain and permissions

Same pinned `actions/setup-go` SHA as the other workflows, `go-version:
"1.25"` mirroring the reusable AI-report workflow's default (comment
beside the value naming the source), `cache: false`, no checkout
(consumers install from the proxy; a `go.mod` in the working directory
could change resolution — canary D4). Workflow-level `permissions: {}`,
job-level `contents: read`; `gh api` needs only read access to the
releases list.

## Risks / Trade-offs

- **`workflow_run` uses the default-branch copy of the workflow file.** The
  check activates only for Release runs completing after this change lands
  on `main`. The first release cut after merge is verified via manual
  dispatch (task 3.3) until the first automatic run.
- **Detection window bounded by retries.** If proxy propagation exceeds the
  ~1-minute retry window, the check fails though the release becomes
  installable minutes later; the failure text names the tag and suggests
  re-dispatching. The canary provides a next-day backstop. Accepted — the
  issue asks for brief retries, not indefinite ones.
- **Published-but-red-run releases skip automatic verification** (D2
  success gating). Accepted; canary and dispatch cover it.
- **Tag resolution is indirect on the automatic path** (D3) because
  `workflow_run` payloads lack dispatch inputs. Accepted; the releases-list
  query is precise in the post-release context.
- **Failure notification** goes to the default workflow-failure recipient
  (same as the canary). Routing to an issue or channel is a possible
  follow-up.

## Constitution Alignment

The proposal assesses all four Gaze constitution principles as PASS; the
decisions above carry them: D4–D5 verify what the proxy actually served
rather than what the gates intended (Accuracy); D2–D4 keep the check's
environment consumer-shaped with no checkout, secrets, or repository
assumptions (Minimal Assumptions); D5–D6 give every failure mode a
distinct, actionable diagnostic with expected vs. actual values
(Actionable Output); the `workflow_dispatch` input makes the whole check
independently exercisable without cutting a release (Testability).
