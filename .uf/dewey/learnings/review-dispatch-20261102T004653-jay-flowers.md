---
tag: review-dispatch
author: jay-flowers
category: gotcha
created_at: 2026-11-02T00:46:53Z
identity: review-dispatch-20261102T004653-jay-flowers
tier: draft
---

Before flagging test gaps found in the diff, or doc claims about tests, read the current *_test.go from disk. In uncommitted-local fix loops the diff snapshot can omit tests that already exist; this nearly produced a false MEDIUM finding in the gaze-init-triage-advisor review.
