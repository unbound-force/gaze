---
tag: quality-external-analyzer
author: jay-flowers
category: gotcha
created_at: 2026-08-31T22:12:53Z
identity: quality-external-analyzer-20260831T221253-jay-flowers
tier: draft
---

The divisor-adversary and divisor-sre both flagged a GoDoc contradiction on `FetchTestMappings`: the docstring said "Returns nil mappings and nil error on graceful degradation (protocol errors are logged to stderr but not propagated)" but the implementation returned `nil, err`. This happened because the docstring was written describing the intended caller behavior rather than the function's own contract. Separately, the original implementation also wrote a warning to stderr AND returned the error — causing double-logging when the caller also logged. Fix: remove stderr logging from `FetchTestMappings` entirely (leave error propagation to the caller), and update the GoDoc to accurately say "Returns the error on failure. The caller is responsible for graceful degradation." This matches the `callAndUnmarshal` pattern used elsewhere in the adapter package — helpers do not log, callers decide how to surface errors.
