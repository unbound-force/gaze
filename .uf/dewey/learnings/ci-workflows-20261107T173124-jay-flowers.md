---
tag: ci-workflows
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:31:24Z
identity: ci-workflows-20261107T173124-jay-flowers
tier: draft
---

`go install module@version` applies no `-ldflags` stamping, so `-X main.version`-style build-identity variables keep their source defaults — a gaze installed via `go install github.com/unbound-force/gaze/v2/cmd/gaze@v2.2.0` reports `gaze version dev (commit none, built unknown)`, identical to a bare `go build`. A `--version` echo added to such a CI install step for provenance therefore records nothing about the installed release and can mislead readers into thinking a dev build ran. When binary self-report 
