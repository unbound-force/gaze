# Design: Release Install Canary

## Context

The reusable AI-report workflow installs gaze for consumers with
`go install github.com/unbound-force/gaze/v2/cmd/gaze@"$GAZE_RELEASE"` and
`gaze --version` (`Install gaze (release)`). Since
`dogfood-ai-report-from-source`, gaze's own `test.yml` takes the
`Install gaze (source)` path instead, so the release path is exercised only
by consumers.

## Goals / Non-Goals

### Goals

- Detect a broken consumer install path for the latest release without
  waiting for a consumer to hit it.
- Use the exact commands of `Install gaze (release)`.

### Non-Goals

- Testing every historical release, or pre-releases.
- Running the AI report itself.
- Blocking releases or PRs on the canary.

## Decisions

### D1: Daily schedule (plus `workflow_dispatch`), not a release trigger

Alternatives considered:

- `on: release: types: [published]` — rejected. Releases are created by
  GoReleaser in `complytime/org-infra`'s reusable workflow using
  `GITHUB_TOKEN`, and events created with `GITHUB_TOKEN` do not start new
  workflow runs, so this trigger would never fire.
- `on: workflow_run` of `Release` — fires reliably, but only once per
  release, and right after it; it cannot see breakage that appears later
  (module proxy or checksum database issues, a dependency removed upstream,
  toolchain drift as the `1.25` setup-go alias moves). It would also run on
  failed or cancelled releases, needing extra gating.

A daily schedule covers both: it runs the latest release within a day of
publication and keeps checking between releases. `workflow_dispatch` lets a
maintainer run it immediately after a release. Cost is one short job per day.
A CI-012 concurrency group keeps the two triggers from overlapping:
`cancel-in-progress` is evaluated for the incoming run, so a
`workflow_dispatch` run cancels an in-flight scheduled run, while a
scheduled run queues behind an in-flight dispatch run.

The cron minute (`17 6 * * *`, UTC) avoids the top of the hour, when
scheduled runs are most often delayed.

### D2: Resolve "latest" from GitHub Releases

`gh release view --json tagName` returns the release GitHub marks as Latest
(non-draft, non-prerelease) — the same release consumers see and the one
`gaze init --ci` users are pointed at. Resolving via the module proxy's
`@latest` was rejected: it would make the canary's input depend on the very
system under test.

### D3: Verify the served module version

`go install` does not set the GoReleaser ldflags, so `gaze --version`
prints `dev` for every proxy install and cannot prove which version was
built. The canary reads `go version -m` for the
`github.com/unbound-force/gaze/v2` `mod` line and requires it to equal the
tag. `gaze --version` still runs, as in the reusable workflow, to prove the
binary executes.

### D4: Mirror the reusable workflow's environment

No checkout (consumers install from the proxy, and a `go.mod` in the working
directory could change resolution), `cache: false` (no `go.sum` to key on),
and `go-version: "1.25"` matching the reusable workflow's `go-version`
input default. Same pinned `actions/setup-go` SHA as the other workflows.
Workflow-level `permissions: {}`, job-level `contents: read`.

### D5: Bounded retry on the network-dependent commands

Review feedback (PR #331) flagged the alert-fatigue failure mode: a
single transient failure — `gh release view` against the REST API, or
`go install` resolving ~40 modules from a cold cache — turns the daily
job red and trains the notification recipient to ignore the only
signal that the consumer install path works. Both commands now carry a
bounded inline retry (3 attempts; 10 seconds between attempts for the
REST-API resolve step, and 30 seconds for the install step, where
module-proxy propagation can lag right after a release); the final
attempt stays fatal with the existing `::error::` diagnostics,
preserving the spec's MUST-fail contract. The local verify step is
deterministic and is not retried. A retry `uses:` action was rejected:
it would add a CI-001/002 SHA-pinning dependency for behavior three
lines of bash provide. The job timeout rises from 10 to 15 minutes to
cover the worst-case retry path.

The retry also makes the canary more tolerant than the single-attempt
consumer path it mirrors: a pass proves the release is eventually
installable, not that a single-shot consumer run would have succeeded.
The tradeoff is accepted — the canary exists to catch persistent
breakage, not to replicate consumer transients — and retry parity in
the reusable workflow remains a possible follow-up.

## Risks / Trade-offs

- **Up to a day of detection latency** after a release. Accepted;
  `workflow_dispatch` closes it when a maintainer wants immediate signal.
- **Scheduled workflows are disabled by GitHub after 60 days without
  repository activity.** Accepted; gaze is actively developed, and GitHub
  emails before disabling.
- **Failure notification** goes to the default scheduled-workflow recipient
  (the user who last changed the cron). Accepted for now; routing to an
  issue or channel is a possible follow-up.
- **The `go-version` default can diverge** if the reusable workflow's default
  changes and this file is not updated. Contained by a comment beside the
  value naming the source; both live in `.github/workflows/`.
