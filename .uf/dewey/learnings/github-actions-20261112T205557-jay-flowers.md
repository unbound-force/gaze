---
tag: github-actions
author: jay-flowers
category: reference
created_at: 2026-11-12T20:55:57Z
identity: github-actions-20261112T205557-jay-flowers
tier: draft
---

GitHub Actions `concurrency.cancel-in-progress` is a property of the incoming run, and `github.event_name` is available in the workflow-level `concurrency` block. `cancel-in-progress: ${{ github.event_name == 'workflow_dispatch' }}` therefore makes a dispatch run preempt an in-flight scheduled run, while a scheduled run (false) queues behind an in-flight dispatch without cancelling it. Two further rules: only one pending run per group is allowed (a newer pending run replaces the older pending run in the queue), and queued runs start when the in-flight run completes rather than re-evaluating freshness.
