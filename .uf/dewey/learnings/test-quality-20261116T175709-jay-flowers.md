---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-16T17:57:09Z
identity: test-quality-20261116T175709-jay-flowers
tier: draft
as_of: "gaze@745e597410ff162c7a3ae03ddce988e471b24e3f"
---

When a behavior flip inverts an existing test's assertion from presence to absence, audit what the old assertion was the sole pin for: the behavior's surviving trigger can keep executing with zero assertion coverage. In gaze's gaze-command-tool-owned change, flipping TestRun_SkipsExisting's "--force hint present" assertion to absence (correct for the .opencode/-only re-run) left the hint's only remaining trigger — the CI-workflow skip appended to Result.Skipped (scaffold.go appendCIWorkflowResult) and counted as user-owned in printSummary — unpinned by any test in the repo, while the same change's docs newly claimed the hint "refers to this workflow only". Sweep every trigger of a shared behavior whose pinning test you invert, and pin each surviving trigger within the same change (e.g., a --ci re-run with an existing workflow asserting "1 file skipped (use --force to overwrite)").
