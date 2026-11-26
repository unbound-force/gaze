---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-26T22:14:23Z
identity: test-quality-20261126T221423-jay-flowers
tier: draft
as_of: "gaze@af1afdaa03a12e9c8ebac16ea545866b2e87d3a8"
evidence:
  - "dewey:.uf/artifacts/dispatch/fe4ccb8d-da26-49c7-9825-563ea6b8343b.json"
  - "dewey:cmd/gaze/ci_workflow_test.go"
  - "dewey:unbound-force/gaze#421"
---

When a YAML/shell contract test pins workflow wiring, pin the assembly-guard fragments (`if [ -n "$VAR" ]; then`) symmetrically for every conditionally assembled variable in the script, not only the ones the current change introduces. A dropped guard fails identically regardless of which change introduced the variable: the empty `--flag=` is passed on every run and correctness then depends on the consuming binary treating empty flags as absent — an unpinned cross-component coupling. Consumed-variable fragments (`$MODEL_ARG \\`) prove the variable reaches the invocation but not that it is guarded. Seen in gaze's cmd/gaze/ci_workflow_test.go reusable-workflow contract test: analyzer/language/coverprofile guards were pinned (new inputs) while the pre-existing MODEL_ARG and threshold guards were left at consumption-only pinning in the same fragment list. (Proposed by divisor-testing during the ci-external-analyzer-support review; adjacent scope tracked in unbound-force/gaze#421.)
