---
tag: review-dispatch
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:53:57Z
identity: review-dispatch-20261102T005357-jay-flowers
tier: draft
---

Security-review heuristic for review-governance config changes (advisor lists, reviewer matrices, .uf config): swapping one reviewer persona for another is NOT gate tampering when four conditions hold — (1) coverage equivalence: the replacement persona covers the same capability scope (e.g., divisor-gaze-testing substituting divisor-testing, both test-quality reviewers); (2) the displaced persona's capabilities entry is preserved so it stays discoverable and routable; (3) the trade-off (e.g., displaced persona becomes host-fallback until upstream unbound-force/unbound-force#901 ships) is documented in the change's design Risks section with the upstream issue linked; (4) the swap goes through a spec workflow with maintainer sanction recorded as an explicit design decision (e.g., gaze-init-triage-advisor design D1 Replace-not-coexist). Gate tampering under AGENTS.md Gatekeeping Value Protection applies when a control is weakened to make an implementation pass WITHOUT human authorization; a documented, sanctioned, coverage-preserving substitution with an audit trail is a configuration change. Corollary: when extending an all-or-nothing registration transform like syncReviewMatrix (internal/scaffold/ufconfig.go) to a new mode, the skip guard must return before the write call so in-memory yaml.Node mutations for earlier modes are discarded — partial registration stays structurally impossible, and consumer matrices missing the new mode's list degrade loudly (warning naming the key) rather than silently.
