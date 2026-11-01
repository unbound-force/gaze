---
tag: scaffolding
author: jay-flowers
category: gotcha
created_at: 2026-11-01T00:12:24Z
identity: scaffolding-20261101T001224-jay-flowers
tier: draft
---

Multi-artifact scaffolders that fail mid-sequence suppress their own completion report: when Run writes artifact group A (e.g. .opencode/ files), then fails on group B (e.g. the CI workflow write), returning (nil, err) at the B failure leaves A's files on disk with no summary printed and post-walk registration steps skipped — the operator's only signal is B's error, with no account of what exists now. Disk idempotency (skip-if-present) makes re-runs safe, but the operator's mental model of the post-failure state comes from the summary that never printed. Pattern: on a late-stage write failure in a multi-group scaffolder, print the accumulated result summary (or annotate the error with the created/skipped paths) before propagating the error, and sequence idempotent registration steps so a re-run heals them.
