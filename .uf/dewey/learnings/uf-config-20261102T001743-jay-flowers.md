---
tag: uf-config
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:43Z
identity: uf-config-20261102T001743-jay-flowers
tier: draft
---

Advisor-list swap as the repo-side workaround for host-fallback finalization defects: when a dispatch planner includes scope-matching discovered agents as host-fallback runs and the dispatch finalizer rejects host-run provenance (uf unbound-force/unbound-force#901, open), listing the desired persona as an advisor in every review mode converts those runs to advisor-sourced runs that finalize cleanly. Verified on gaze's gaze-init-triage-advisor change (2026-11-01): swapping divisor-gaze-testing into advisor.triage (joining code/spec/feedback) eliminated the 9-of-12 triage-panel finalization failures. Three guards make the swap safe: (1) the displaced generic persona's capabilities entry is preserved so it stays discoverable (it becomes host-fallback, exactly as it already was in the other modes — document this trade-off in design D1 with the upstream issue linked); (2) the matrix merge stays all-or-nothing — a missing or non-sequence advisor list in ANY mode skips the whole file with a warning naming the key, because in-memory yaml.Node mutations for earlier modes are discarded when the function returns before writeYAMLIfChanged, so partial registration is structurally impossible; (3) the hand-edited repo config is guarded by an end-state idempotency test (TestSyncUFReviewConfig_RepoConfigIsEndState — copy repo .uf files to a temp dir, run the sync, assert zero rewrites and zero warnings) so the git edit and the sync function cannot silently diverge.
