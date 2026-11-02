---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T00:54:00Z
identity: test-quality-20261102T005400-jay-flowers
tier: draft
---

Fixture surgery via strings.Replace can drift silently: when a test derives a variant fixture from a base fixture by replacing a byte-exact block literal (e.g., stripping a YAML advisor list from a review-matrix fixture to build a skip-case input), the Replace pattern duplicates knowledge of the base fixture's content. A later edit to the base fixture (adding a list entry, reordering, re-indenting) no-ops the surgery, the derived fixture silently becomes byte-identical to the base, and dependent tests fail with misleading success-path diagnostics (unexpected registration, missing warning) instead of pointing at the stale pattern. Make the surgery self-verifying: compose the base fixture and the Replace pattern from one shared block constant, or assert at init/test time that each derived fixture actually differs from the base fixture.
