# Design: Bump AI Report Default to v2.4.0

## Context

Callers reference the reusable workflow by tag. GitHub resolves input
defaults from the workflow file at that tag, so `@v2.4.0` uses whatever
default the tagged commit contains. Today that would be `v2.2.0`.

## Goals / Non-Goals

### Goals

- `@v2.4.0` callers that omit `gaze-version` run v2.4.0.

### Non-Goals

- Changing the dogfood job or the scaffolded consumer workflow.
- Enforcing the default-equals-tag rule mechanically.

## Decisions

### D1: Bump before tagging, not after

Bumping after the tag would leave `@v2.4.0` defaulting to v2.2.0 for the
life of that tag. The merge commit of this change is the one to tag.

### D2: Record the rule next to the value

A YAML comment beside `default:` says the value must equal the release tag
that ships the file, so the next release's author sees it at the point of
change. A release preflight check is deferred (see proposal, Out of scope).

## Risks / Trade-offs

- A short window where `main` defaults to an unpublished tag. Mitigated by
  tagging immediately after merge; `@main` is not a supported pin.
