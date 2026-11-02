---
tag: review-dispatch
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:55Z
identity: review-dispatch-20261102T001755-jay-flowers
tier: draft
---

When uf's review-dispatch finalizer rejects host-fallback run provenance (unbound-force/unbound-force#901, open), the repo-side workaround is advisor-listing the affected persona for that mode. A 1:1 swap (replacing the generic divisor-testing with divisor-gaze-testing, rather than listing both) preserves the one-test-persona-per-mode shape but moves host-fallback exposure to the displaced generic persona until the upstream fix ships — the exposure moves, it does not disappear. Corollary for gaze consumers: extending syncReviewMatrix's mode loop (internal/scaffold/ufconfig.go) to a new mode also extends its all-or-nothing strictness — consumer matrices missing that advisor list regress from partial registration to a skip-whole-file warning, so the new mode's list must be treated as a hard prerequisite of the registration merge.
