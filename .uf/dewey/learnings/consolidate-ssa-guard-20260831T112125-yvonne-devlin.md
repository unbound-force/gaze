---
tag: consolidate-ssa-guard
author: yvonne-devlin
category: gotcha
created_at: 2026-08-31T11:21:25Z
identity: consolidate-ssa-guard-20260831T112125-yvonne-devlin
tier: draft
---

When deduplicating tests during a DRY consolidation, deleting a whole *_test.go file can be the correct Zero-Waste outcome: internal/quality/pairing_test.go held ONLY the 3 duplicated TestSafeSSABuild_* tests plus a stale SC-001 comment block with no test functions. After removing the duplicated triad it would not compile (unused testing/quality imports), so deleting the entire file was cleaner than leaving a stub. Behavioral coverage was preserved because the 3 canonical tests moved to internal/ssaguard/ssaguard_test.go. Reviewers accept a net test-count decrease (-3 here: 6 duplicates -> 3 shared) as long as the acceptance criterion is framed as 'behavioral coverage is neutral-to-positive' NOT 'net test count does not decrease' — the latter wording is a self-contradicting acceptance criterion that a spec reviewer (divisor-testing) will flag HIGH. Also satisfy Constitution IV by stating the coverage strategy explicitly (unit only, 100% branch coverage, enumerate the branches) in proposal/design/tasks.
