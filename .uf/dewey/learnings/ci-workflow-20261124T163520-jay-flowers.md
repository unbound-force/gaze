---
tag: ci-workflow
author: jay-flowers
category: gotcha
created_at: 2026-11-24T16:35:20Z
identity: ci-workflow-20261124T163520-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

When a change makes the sole required status check aggregate other jobs via `needs`, any job added to that aggregate becomes merge-blocking — including advisory, externally-dependent jobs (AI report, secret-scan). Audit the aggregate's `needs` set against the 'required vs advisory' intent before adopting it as the single required check.
