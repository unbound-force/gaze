---
tag: gitignore
author: jay-flowers
category: gotcha
created_at: 2026-11-24T21:38:44Z
identity: gitignore-20261124T213844-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
evidence:
  - ".uf/artifacts/dispatch/624f5216-a5f4-4ebb-b4f9-de48798bbaea.json"
---

A bare filename in .gitignore (e.g. `gaze`) matches at any depth, so it silently ignores a same-named directory's untracked files — a new test under cmd/gaze/ was hidden from git status and review until the bare `gaze` line was removed. Keep build-artifact ignores anchored (`/gaze`) so the root binary stays ignored while a same-named source directory remains tracked.
