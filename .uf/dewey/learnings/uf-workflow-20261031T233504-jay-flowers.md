---
tag: uf-workflow
author: jay-flowers
category: gotcha
created_at: 2026-10-31T23:35:04Z
identity: uf-workflow-20261031T233504-jay-flowers
tier: draft
---

A checked MUST-gate checkbox (e.g., a Website Documentation Gate task in tasks.md) is weak review evidence: reviewers cannot distinguish "done" from "asserted done" without a durable artifact reference. Record the gate's output — the created issue number/URL, command output, or artifact path — in the change's documentation trail at the moment the task is checked, following the established precedent (the gaze divisor-gaze-testing change recorded "Website docs issue: unbound-force/website#345" in its AGENTS.md Recent Changes entry). This is especially important for gates whose evidence lives in another repository (website-docs issues), which a code review of the branch diff can never verify.
