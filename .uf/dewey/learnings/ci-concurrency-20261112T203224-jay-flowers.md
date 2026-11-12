---
tag: ci-concurrency
author: jay-flowers
category: gotcha
created_at: 2026-11-12T20:32:24Z
identity: ci-concurrency-20261112T203224-jay-flowers
tier: draft
---

GitHub Actions `concurrency` with a single static group and `cancel-in-progress: true` is newest-wins in BOTH directions: a scheduled run starting while a `workflow_dispatch` run is in-flight will cancel the dispatch run, not just the reverse. When an on-demand run is meant to take precedence, scope cancellation by event. Note the evaluation context: `cancel-in-progress` expressions are evaluated for the INCOMING run (the run being created decides whether to cancel what's in progress), so dispatch-precedence is `cancel-in-progress: ${{ github.event_name == 'workflow_dispatch' }}` — an incoming dispatch cancels the in-flight nightly, an incoming nightly queues behind the in-flight dispatch. Surfaced on gaze's install-canary.yml (PR #331 fix round), where dispatch is the "verify right after release" path.
