---
tag: ci-cd
author: jay-flowers
category: pattern
created_at: 2026-11-15T01:36:58Z
identity: ci-cd-20261115T013658-jay-flowers
tier: draft
---

A `workflow_run` completion trigger cannot read the triggering run's `workflow_dispatch` inputs, so a post-release verification that resolves its target from 'the most recent GitHub release' at execution time can verify the wrong tag: with a queued (cancel-in-progress: false) concurrency group or a re-run of an older Release run, it silently re-verifies whatever is newest and can give a false pass for the intended tag. Guard by asserting the resolved release's published/created timestamp is newer than `github.event.workflow_run.created_at`, or fail with a diagnostic when it is not.
