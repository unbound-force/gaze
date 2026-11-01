---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-10-31T23:35:01Z
identity: test-quality-20261031T233501-jay-flowers
tier: draft
---

Golden-file and table-driven matrices pin a parser's value-level outcomes but not its scan-termination decisions. When a parser's doc comment documents a tie-breaking rule — e.g., gaze's parseGoModVersion: "the first line beginning `go ` decides the outcome; the scan does not continue past a non-matching directive line" — add a distinguishing fixture whose observable result differs under the alternative implementation (malformed directive followed by a valid one: break → documented fallback, continue → the later valid value). Without the distinguishing fixture, refactoring `break` to `continue` passes the entire matrix and the golden file while silently changing behavior for malformed-then-valid inputs. Generalizes to any first-match-wins / last-match-wins / precedence decision in a scanner: cover the decision branch with an input where the competing strategy yields a different answer, not only inputs where both strategies agree. Found in the init-ci-scaffold review (2026-10-31): every parseGoModVersion matrix case was a single-directive go.mod, so the documented break semantics had no regression protection.
