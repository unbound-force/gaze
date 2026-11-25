---
tag: ci-workflows
author: jay-flowers
category: pattern
created_at: 2026-11-24T21:18:04Z
identity: ci-workflows-20261124T211804-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
evidence:
  - ".uf/artifacts/dispatch/a4de843c-42ca-4c3c-8c02-838a46b09523.json"
---

Gating matrix CI jobs so a required check still passes for intentionally-skipped PRs (e.g. learnings-only) requires moving the required check off the per-leg matrix names onto a single aggregate job (e.g. `CI OK` needing the gated jobs, `if: ${{ !cancelled() }}`, a jq predicate that fails on any failure/cancelled and passes on success/skipped). Skipped matrix legs never report their per-leg check names, so keeping the leg names as required checks hard-blocks every skipped PR — the aggregate is the only safe required check. The corresponding branch-protection switch is an out-of-repo MUST with a hard-block failure mode if skipped or reverted; sequence it protection-first, save the prior required-checks JSON, provide a rollback command, and record the prerequisite in durable in-repo docs (not only an archived design runbook).
