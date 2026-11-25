# Post-Release Install Check

## Why

The release pipeline's gates all verify repository state *before* publication:
`validate-module-major` (tag major vs. the `go.mod` module path),
`validate-workflow-floor`, `validate-ai-report-default`, and the org-infra
preflight (tag format, uniqueness, semver ordering, CI status). None of them
verify that the release GitHub actually published is consumable through the
public Go module proxy (`proxy.golang.org`) — the path every `go install`
consumer, including the reusable AI-report workflow's `Install gaze (release)`
step, depends on.

The gap is not hypothetical. The README documents that earlier v2 tags were
published with an incompatible module path and cannot be installed from the
`/v2` module path; every pre-flight gate passed for those tags. Because the
proxy content-caches a version permanently, such a tag is *permanently*
uninstallable — remediation requires cutting a new tag. The failure is only
visible by installing the published tag through the public proxy.

The install canary (`release-install-canary`) closes part of this gap, but on
its own terms: it runs daily against whatever release GitHub marks Latest, so
a freshly published tag waits up to a day for its first signal, prereleases
are never checked, and the canary's purpose is drift detection between
releases rather than verification of a specific publication. Issue #297 asks
for the missing layer: a lightweight smoke test that verifies the exact
published tag immediately after publication.

## What Changes

- New workflow `.github/workflows/post-release-install-check.yml`:
  - Runs automatically when the `Release` workflow completes successfully
    (`workflow_run` gated on the run's conclusion). This is how the change
    handles the `GITHUB_TOKEN` limitation: GoReleaser publishes releases with
    `GITHUB_TOKEN`, and events created with `GITHUB_TOKEN` do not start
    workflow runs, so `on: release: published` would never fire — but the
    Release workflow itself is human-dispatched, and its *completion* emits a
    `workflow_run` event reliably.
  - Also runs on demand via `workflow_dispatch` with a required `tag` input,
    verifying any published tag (including older ones and prereleases).
  - Resolves the tag to verify: the dispatch input, or — on the post-release
    path — the most recent non-draft GitHub release, which right after a
    successful Release run is the release that run published.
  - Installs `go install github.com/unbound-force/gaze/v2/cmd/gaze@<tag>`
    from the public proxy into temporary `GOBIN` and `GOMODCACHE` directories
    under `RUNNER_TEMP`, with a bounded inline retry (3 attempts, 30s
    backoff) absorbing normal proxy propagation lag for a just-published tag.
  - Runs `go version -m` on the installed binary and asserts both the module
    path (`github.com/unbound-force/gaze/v2`) and the exact requested version,
    with distinct `::error::` diagnostics for each failure mode; runs
    `gaze --version` to prove the binary executes.
  - Read-only: workflow-level `permissions: {}`, job-level `contents: read`;
    no checkout, no secrets beyond `github.token`, 15-minute job timeout, and
    a CI-012 concurrency group.
- `AGENTS.md`: CI/CD list entry and Recent Changes entry.

### Out of scope

- No changes to `release.yml`, signing, or Homebrew behavior — the check only
  observes completed Release runs; it never runs inside the release pipeline.
- No pre-merge network calls: the workflow has no `push`/`pull_request`
  triggers, so pre-merge CI stays hermetic.
- No large fake-command test framework, no generalized workflow meta-tests,
  no protected-identifier snapshots (issue #297 non-goals).
- No changes to the install canary; the two keep separate concerns (canary:
  daily latest-release drift detection; this check: immediate exact-tag
  publication verification).

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `ci-workflow`: published releases are verified installable from the public
  Go module proxy at the exact release tag, immediately after publication or
  on demand.

### Removed Capabilities

- None.

## Impact

- **Internal surfaces**: `.github/workflows/post-release-install-check.yml`
  (new), `AGENTS.md`.
- **Consumers**: none (internal CI only; no user-facing behavior changes).
- **Cost**: one short `ubuntu-latest` job per release plus on-demand runs
  (setup-go + one `go install` from a cold cache), `contents: read` only, no
  secrets beyond `github.token`.
- **No website issue**: CI/CD pipeline changes are exempt.

## Constitution Alignment

Assessed against the Gaze constitution (`.specify/memory/constitution.md`).

### I. Accuracy

**Assessment**: PASS. The check installs through the real public proxy path
and asserts, from the binary's build info, both the module path and the exact
version the proxy served. A pass means the published release installs exactly
as a consumer would install it. This is the only layer that can catch the
documented historical failure class (tags published with an incompatible
module path), which every offline gate passed.

### II. Minimal Assumptions

**Assessment**: PASS. No checkout, no secrets, no repository fixtures; the
job sees the environment of a consumer's first install — a fresh runner, an
empty temporary `GOBIN`/`GOMODCACHE`, and the same pinned toolchain default
(`1.25`) as the reusable AI-report workflow. The only inputs are the release
tag (from the completed Release run or the dispatch input) and
`github.token`.

### III. Actionable Output

**Assessment**: PASS. Each failure mode gets a distinct `::error::` naming
the failing phase (tag resolve / install / execute / version verify) with
expected vs. actual values. Bounded retries suppress transient proxy noise so
a red run means persistent breakage, and the failure text names the tag and
suggests the recovery path (re-dispatch; the proxy is eventually consistent).

### IV. Testability

**Assessment**: PASS. The workflow is independently exercisable on demand
via `workflow_dispatch` against any published tag, without cutting a release.
The deterministic parts are covered by local CI parity (actionlint,
yamllint) and parsing checks against sample `go version -m` output; the
network path is by definition an end-to-end check. The change adds no
production code.
