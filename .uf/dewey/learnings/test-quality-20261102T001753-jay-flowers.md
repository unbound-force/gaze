---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:17:53Z
identity: test-quality-20261102T001753-jay-flowers
tier: draft
---

Extending a shared guard loop to a new enum value multiplies the untested-disjunct surface: the new value inherits every disjunct of the guard, but tests usually cover only the disjunct that motivated the change (missing key), leaving sibling disjuncts (present-but-non-sequence values) unguarded across all values — deleting the Kind check passes the entire suite. When a spec restates a compound guard ("missing or is not a sequence"), cover each disjunct for the newly added value, not just the realistic scenario. Effective test shape for mode-loop extensions (all three observed in the gaze-init-triage-advisor change): (a) extend the end-state assertion to the new value (assertMatrixSwapped gained the triage mode), (b) add a skip-path case asserting byte-identical file content plus a warning naming the key — byte-identity also proves in-memory yaml.Node mutations from earlier loop iterations were discarded when the write never happens, and (c) an outcome-based repo-config no-op test (stage the repository's own config, assert the sync registers nothing and never warns) locks the repository's config without enumerating modes, catching config regressions in either direction.
