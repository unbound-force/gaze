# Design: Dogfood AI Report From Source

## Context

The reusable AI-report workflow installs gaze with
`go install github.com/unbound-force/gaze/v2/cmd/gaze@"$GAZE_RELEASE"`, where
`GAZE_RELEASE` is the `gaze-version` input (default `v2.2.0`). The
repository's own `test.yml` dogfood job omits `gaze-version`, so it reports
with v2.2.0 regardless of what is on `main`. Spec 040's adapter and answer
validation therefore never ran in CI, and the v2.2.0 narration-as-report
failure recurred in run 41698377720.

## Goals / Non-Goals

### Goals

- The dogfood job reports with the binary built from the commit under test,
  on both `push` and same-repo `pull_request` events.
- Zero behavior change for existing consumers.
- Record the producing binary's version in the job log.

### Non-Goals

- Changing the `gaze-version` default or cutting a release (follow-up,
  sequenced after this change is green on `main`).
- Making `build-from-source` useful for repositories other than gaze.

## Decisions

### D1: Build from the checkout, not `go install @${{ github.sha }}`

Alternative considered: pass `gaze-version: ${{ github.sha }}` from
`test.yml`. Rejected because:

- On `pull_request`, `github.sha` is the ephemeral `refs/pull/N/merge`
  commit; resolving it through `proxy.golang.org` is unreliable and would
  require `GOPROXY=direct` (and `GONOSUMDB`) workarounds.
- Every push would publish a pseudo-version of gaze to the public module
  proxy and checksum database.
- The job already checks out the exact commit; `go install ./cmd/gaze` builds
  it with no network resolution of gaze itself.

### D2: Opt-in boolean input with a module-path guard

`build-from-source` (`type: boolean`, default `false`). When `true`, the
install step first checks `go list -m` equals
`github.com/unbound-force/gaze/v2`; otherwise it fails with an `::error::`
directing the caller to `gaze-version`. This prevents a consumer from
silently building some other `./cmd/gaze` or getting a confusing
"directory not found" error.

### D3: Two mutually exclusive install steps

`Install gaze (release)` runs when `build-from-source` is false and
`Install gaze (source)` when true, rather than branching inside one script.
The step name in the log then states which path ran. Both end with
`gaze --version`. The source build stamps the same three symbols as
`.goreleaser.yaml`: `main.version` (`dev-<12-char SHA>`), `main.commit`
(`GITHUB_SHA`), and `main.date` (the commit date). `main.version` feeds the
report metadata, so the published report header names the commit as well as
the job log.

### D4: `gaze-version` default unchanged here

Callers pinned to `@vX.Y.Z` receive the default baked into that tag, so the
bump to the first spec-040 release must land in the commit that is tagged.
Bumping it now would point the default at a tag that does not yet exist and
break any caller relying on the default from `@main`. Issue #313 tracks the
bump.

## Risks / Trade-offs

- The dogfood job will now fail if the configured model cannot meet spec
  040's answer contract. Intended: it surfaces the problem before a release
  ships it to consumers.
- The dogfood job no longer validates the release-install path. Consumers
  exercise it, and `gaze init --ci` scaffolds pin it explicitly; the step is
  unchanged.
