---
tag: ci-workflow-testing
author: jay-flowers
category: gotcha
created_at: 2026-11-24T21:38:49Z
identity: ci-workflow-testing-20261124T213849-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
evidence:
  - ".uf/artifacts/dispatch/624f5216-a5f4-4ebb-b4f9-de48798bbaea.json"
---

A job-level drift guard that sorts every workflow job into merge-blocking (aggregate needs) versus advisory needs a third invariant: advisory jobs must still transitively depend on a gated job, or they silently run on exactly the PR class the gating exists to skip. An advisory allowlist entry says "cannot block a merge"; it says nothing about "inherits the skip". In gaze's skip-ci-learnings-only-prs change, the gaze-ai-report job's learnings-only skip rested entirely on its `needs: unit-and-integration` edge, which no contract test pinned — the spec's MUST (CIW-FR-002) was verified only by a one-time manual runbook step while every other wiring edge in the same change got an automated pin. Fix shape: assert each allowlisted job's needs intersects the gated set.
