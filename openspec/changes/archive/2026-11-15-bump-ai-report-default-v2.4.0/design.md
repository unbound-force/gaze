# Design: Bump AI Report Default to v2.4.0

## Context

Callers reference the reusable workflow by tag. GitHub resolves input
defaults from the workflow file at that tag, so `@v2.4.0` uses whatever
default the tagged commit contains. Today that would be `v2.2.0`.

## Goals / Non-Goals

### Goals

- `@v2.4.0` callers that omit `gaze-version` run v2.4.0.
- Every future release fails fast if the default was not bumped.

### Non-Goals

- Changing the dogfood job or the scaffolded consumer workflow.

## Decisions

### D1: Bump before tagging, not after

Bumping after the tag would leave `@v2.4.0` defaulting to v2.2.0 for the
life of that tag. The merge commit of this change is the one to tag.

### D2: Enforce the rule with an offline release gate

A YAML comment beside `default:` states the rule at the point of change, and
`validate-ai-report-default.sh` enforces it. It follows the
`validate-workflow-floor.sh` pattern: offline, unskippable, a separate job
that preflight depends on, with exit 1 for a mismatch and exit 2 for usage
or unparsable state. CI cannot catch a stale default any other way, because
the dogfood job sets `build-from-source: true` and never reaches the release
install path.

The parser is a small awk scan for the `default:` inside the
`gaze-version:` block, not a YAML library, so the gate needs nothing beyond
the runner image. A repository-state test asserts the real workflow stays
parsable, so a reformat cannot make the gate silently pass.

### D3: Exact equality, including pre-releases

A `vX.Y.Z-rc.N` tag must ship a `vX.Y.Z-rc.N` default. The final tag cannot
change what a pre-release tag already contains, and defaulting a pre-release
to the not-yet-published final would fail at `go install`.

## Risks / Trade-offs

- A short window where `main` defaults to an unpublished tag. Mitigated by
  tagging immediately after merge; `@main` is not a supported pin.
