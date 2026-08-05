---
tag: cross-repo-workflow
author: jay-flowers
category: gotcha
created_at: 2026-08-02T16:45:45Z
identity: cross-repo-workflow-20260802T164545-jay-flowers
tier: draft
---

The fix for gaze#204 (compile verification gate in gaze-test-generator) was originally applied in unbound-force/unbound-force#404, but the canonical source for the gaze-test-generator scaffold is in the gaze repo at internal/scaffold/assets/agents/gaze-test-generator.md. When an issue is filed against gaze and the fix touches agent prompts, the fix must be applied in the gaze repo where the scaffold canonical copy lives, not in the unbound-force repo which has its own separate copy. Cross-repo issue-to-PR linkage via "Fixes:" can close issues in the wrong repo, masking that the actual source of truth was never updated.
