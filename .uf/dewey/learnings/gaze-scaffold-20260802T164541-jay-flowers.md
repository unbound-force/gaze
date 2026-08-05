---
tag: gaze-scaffold
author: jay-flowers
category: gotcha
created_at: 2026-08-02T16:45:41Z
identity: gaze-scaffold-20260802T164541-jay-flowers
tier: draft
---

When porting fixes between repos in the unbound-force organization (e.g., from unbound-force/unbound-force to unbound-force/gaze), the gaze-test-generator agent prompt exists in two locations in the gaze repo: the scaffold canonical copy at internal/scaffold/assets/agents/gaze-test-generator.md and the active runtime copy at .opencode/agents/gaze-test-generator.md. The file is tool-owned (isToolOwned in internal/scaffold/scaffold.go), meaning gaze init will auto-overwrite it when content differs — so the proposal's user impact should state auto-propagation, not manual update. Both copies must be kept byte-identical. The scaffold copy is the source of truth, edited first, then the active copy is synced via cp. The SRE reviewer caught the incorrect user impact statement about skip-if-present behavior during spec review, which was corrected before implementation began.
