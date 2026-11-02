---
tag: uf-registration
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:46Z
identity: uf-registration-20261102T001746-jay-flowers
tier: draft
---

When uf's dispatch planner includes a discovered divisor agent that is not advisor-listed for a mode, the run is planned as host-fallback and the review-dispatch finalizer rejects its provenance (unbound-force/unbound-force#901, open) — every panel containing such a run fails Phase 4.6 finalization (observed: 9 of 12 triage panels on opsx/gaze-init-triage-advisor's predecessor state). The repo-side mitigation is to advisor-list the persona in all four modes of .uf/review-matrix.yaml. In the gaze repo, keep the repository's own .uf config in the exact end-state that gaze init's syncReviewMatrix produces and let TestSyncUFReviewConfig_RepoConfigIsEndState enforce the lockstep: when extending the registration transform to another advisor mode, update the repo matrix and the transform in the same change, or the end-state no-op test fails with "repo .uf config must already be in the sync end-state". This turns config/code drift into a CI failure instead of a silent dispatch break, and it documents the trade-off symmetrically: the persona swapped out (generic divisor-testing) becomes host-fallback in that mode until #901 ships, which must be recorded in the change's design Risks section.
