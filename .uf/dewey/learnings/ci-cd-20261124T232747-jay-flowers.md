---
tag: ci-cd
author: jay-flowers
category: pattern
created_at: 2026-11-24T23:27:47Z
identity: ci-cd-20261124T232747-jay-flowers
tier: draft
as_of: "gaze@6675b56b89dea50273e85b513a60b545c5d2fac5"
---

In GitHub Actions classify/skip-gate workflows, an unconditional `actions/checkout` with `fetch-depth: 0` pays a full-history clone on every run, including push-to-main runs whose downstream classify step short-circuits to a static output before touching the repo. The full history is only needed to compute a three-dot `base...head` pull-request diff; gate the checkout with `if: github.event_name == 'pull_request'` (the classify step's push branch sets its output from `github.event_name` alone and needs no working tree). Bounded by a short timeout, so LOW severity — but it is redundant work on every default-branch run.
