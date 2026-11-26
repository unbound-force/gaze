---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-26T22:14:16Z
identity: test-quality-20261126T221416-jay-flowers
tier: draft
as_of: "gaze@af1afdaa03a12e9c8ebac16ea545866b2e87d3a8"
evidence:
  - "dewey:.uf/artifacts/dispatch/fe4ccb8d-da26-49c7-9825-563ea6b8343b.json"
  - "dewey:internal/scaffold/ci_test.go"
---

Pair byte-identity golden files with a variant-independent invariant scan when a template renderer substitutes per-variant placeholder lists. A cross-variant placeholder copy-pasted into the wrong template (e.g. __GO_VERSION__ inside a Node template) substitutes to the empty ciContext value and vanishes from the rendered output, so per-variant "no placeholder remains" lists cannot catch it — and byte-identity goldens only catch it if the committed golden is not blindly regenerated alongside. A generic residue scan (regexp `__[A-Z][A-Z0-9_]*__` over the rendered output) holds for any correct output regardless of which placeholders a variant is supposed to use, so blind golden regeneration cannot launder the regression. Implemented in gaze's internal/scaffold/ci_test.go (placeholderResidueRe/assertNoPlaceholders) for the Go/Node/Python CI workflow templates. (Proposed by divisor-testing during the ci-external-analyzer-support review.)
