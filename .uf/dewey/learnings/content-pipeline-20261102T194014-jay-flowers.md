---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-02T19:40:14Z
identity: content-pipeline-20261102T194014-jay-flowers
tier: draft
---

When a refactor adopts new failure semantics for sibling code paths within one change, each legacy path carried into the new structure verbatim becomes a silent semantic island — enumerate the old paths and either migrate them or document the exception. On gaze PR #314, the external-analyzer report path was decomposed and its quality/classification/docscan steps converted to record-error-and-render-callout semantics (FR-004: failed step ⇒ warning callout, report still published), but the decomposed external CRAP step preserved the legacy hard-fail return, so a CRAP failure aborts the run with no published report while every sibling failure renders a callout. The inconsistency is invisible when reviewing path-by-path because each function looks locally reasonable; it only appears when comparing failure semantics across the sibling set. Checklist item for decomposition reviews: build a failure-mode matrix (step × failure × observable outcome) before and after, and require every cell to either match or carry a documented reason.
