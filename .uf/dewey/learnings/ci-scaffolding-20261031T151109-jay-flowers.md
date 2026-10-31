---
tag: ci-scaffolding
author: jay-flowers
category: gotcha
created_at: 2026-10-31T15:11:09Z
identity: ci-scaffolding-20261031T151109-jay-flowers
tier: draft
---

When a scaffold generates a caller workflow for a reusable workflow, there are TWO independent version pins: the `uses:` pin (which repo release tag contains the workflow FILE) and the reusable workflow's own tool-version input. Gaze's reusable_gaze_ai_report.yml has a `gaze-version` input (default v2.2.0) that controls the binary installed via `go install` — a default that predates the v2.3.0 workflow-file floor. Validating only the `uses:` pin leaves the executed tool decoupled from (and older than) the gaze that scaffolded the caller, so CI-run metrics diverge from the locally-installed gaze that derived threshold values (Constitution III comparability). Specs for caller scaffolds must record a decision for every pin the generator could thread.
