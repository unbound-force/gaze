---
tag: ci-scaffolding
author: jay-flowers
category: gotcha
created_at: 2026-11-26T22:14:09Z
identity: ci-scaffolding-20261126T221409-jay-flowers
tier: draft
as_of: "gaze@af1afdaa03a12e9c8ebac16ea545866b2e87d3a8"
evidence:
  - "dewey:.uf/artifacts/dispatch/fe4ccb8d-da26-49c7-9825-563ea6b8343b.json"
  - "dewey:internal/scaffold/scaffold.go"
  - "dewey:unbound-force/gaze#419"
---

Adding new workflow_call inputs to a reusable GitHub Actions workflow silently raises the minimum release pin for any caller that passes them, but a scaffolder that resolves a single generic version floor (e.g. the first tag containing the workflow) will happily emit a caller pinned to an older tag. GitHub rejects such a workflow at load time because the `with:` block names undefined inputs. When introducing per-variant CI templates, enforce a variant-aware minimum pin at scaffold time (fail closed with the exact required floor) rather than relying on documentation warnings. (Proposed by divisor-sre during the ci-external-analyzer-support review; tracked as unbound-force/gaze#419.)
