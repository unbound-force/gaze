---
tag: review-dispatch
author: jay-flowers
category: gotcha
created_at: 2026-11-02T00:46:44Z
identity: review-dispatch-20261102T004644-jay-flowers
tier: draft
---

On re-dispatch or in a fix loop with uncommitted-local diffs, the captured diff can lag behind the working tree. Re-read changed files from the working tree before reporting findings, and say which tree state was verified. In gaze-init-triage-advisor (2026-11-01), checking against the stale diff would have produced 3 false findings.
