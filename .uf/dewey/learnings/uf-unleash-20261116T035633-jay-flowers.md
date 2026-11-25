---
tag: uf-unleash
author: jay-flowers
category: gotcha
created_at: 2026-11-16T03:56:33Z
identity: uf-unleash-20261116T035633-jay-flowers
tier: draft
as_of: "gaze@6d77d4d0f15cefa35fabf8d0de5eae2b5bffe5f1"
evidence:
  - "session:ses_e1db609caffeSiu8f4Gp3ObxhX"
  - "session:ses_e1d8bb9cfffeAHlISe2K29cwFY"
---

In this environment, uf_delegate_command children MUST be pinned to an explicit model (model: "enmaas/rits/zai-org/glm-5-3") — default child model resolution fails with "Cannot connect to API: models.corp.redhat.com" (unreachable endpoint), and the pin has been reliable across 15+ delegations once discovered. When a delegation hits its wall-clock timeout, the child session is killed but its in-flight bash processes complete as orphans and write their artifacts (logs, files); do NOT blindly re-delegate — first recover evidence from the session DB (parts of the child session id) and the artifact files, then only re-run what is genuinely missing. Also: the Task tool is unavailable in this session ("Subagent depth limit reached (1)" — this session is itself a child), so parallel [P] tasks run sequentially via pinned uf_delegate_command instead of worker spawning.
