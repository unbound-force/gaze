---
tag: ci-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-24T18:35:39Z
identity: ci-workflow-20261124T183539-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

When a CI change's safety depends on an external control plane (e.g., GitHub branch protection required status checks), the invariant is neither code-reviewable nor drift-detected: workflow-scoped contract tests pin YAML wiring but not the protection config, and the migration/rollback runbook tends to live only in the point-in-time design artifact that OpenSpec archives. Remediation: add a scheduled (or documented periodic) check that reads the protection API and asserts the required-check set, and persist the migration/rollback runbook in durable operational docs rather than the archived design. Also note the recovery path when the aggregate required check itself misbehaves.
